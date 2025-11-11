package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/scp"
)

// SCPCmd implements the DICOM SCP server command.
type SCPCmd struct {
	Port       int    `arg:"" default:"11112" help:"Port to listen on"`
	AETitle    string `name:"aet" default:"RADX-SCP" help:"Application Entity Title"`
	OutputDir  string `name:"output-dir" default:"./dicom-received" help:"Output directory for received files"`
	MaxPDU     uint32 `name:"max-pdu" default:"16384" help:"Maximum PDU size in bytes"`
	MaxConns   int    `name:"max-conns" default:"10" help:"Maximum concurrent connections"`
	Organize   bool   `name:"organize" default:"true" help:"Auto-organize received files by Study/Series/Instance UID"`
	AcceptEcho bool   `name:"accept-echo" default:"true" help:"Accept C-ECHO requests"`

	// SOP Class filter
	SOPClasses []string `name:"sop-class" help:"Accepted SOP Class UIDs (default: all)"`
}

// echoHandler implements scp.EchoHandler.
type echoHandler struct {
	logger        *log.Logger
	requestCount  atomic.Uint32
	acceptEcho    bool
}

// HandleEcho handles C-ECHO requests.
func (h *echoHandler) HandleEcho(ctx context.Context, req *scp.EchoRequest) *scp.EchoResponse {
	if !h.acceptEcho {
		h.logger.Warn("C-ECHO not accepted", "calling_ae", req.CallingAE)
		return &scp.EchoResponse{Status: dimse.StatusSOPClassNotSupported}
	}

	count := h.requestCount.Add(1)
	h.logger.Info("C-ECHO request received",
		"calling_ae", req.CallingAE,
		"called_ae", req.CalledAE,
		"count", count,
	)

	return &scp.EchoResponse{Status: dimse.StatusSuccess}
}

// storeHandler implements scp.StoreHandler.
type storeHandler struct {
	logger        *log.Logger
	outputDir     string
	organize      bool
	sopClasses    map[string]bool
	storeCount    atomic.Uint32
	failCount     atomic.Uint32
}

// HandleStore handles C-STORE requests.
func (h *storeHandler) HandleStore(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
	count := h.storeCount.Add(1)

	h.logger.Info("C-STORE request received",
		"calling_ae", req.CallingAE,
		"called_ae", req.CalledAE,
		"sop_class", req.SOPClassUID,
		"sop_instance", req.SOPInstanceUID,
		"count", count,
	)

	// Check SOP Class filter
	if len(h.sopClasses) > 0 && !h.sopClasses[req.SOPClassUID] {
		h.logger.Warn("SOP Class not accepted", "sop_class", req.SOPClassUID)
		h.failCount.Add(1)
		return &scp.StoreResponse{Status: dimse.StatusSOPClassNotSupported}
	}

	// Determine output path
	var outputPath string
	if h.organize {
		// Organize by Study/Series/Instance UID
		studyUID, seriesUID, err := extractOrganizationUIDs(req.DataSet)
		if err != nil {
			h.logger.Error("Failed to extract UIDs for organization", "error", err)
			h.failCount.Add(1)
			return &scp.StoreResponse{Status: dimse.StatusProcessingFailure}
		}

		outputPath = filepath.Join(
			h.outputDir,
			studyUID,
			seriesUID,
			req.SOPInstanceUID+".dcm",
		)
	} else {
		// Flat structure
		outputPath = filepath.Join(h.outputDir, req.SOPInstanceUID+".dcm")
	}

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		h.logger.Error("Failed to create output directory", "dir", outputDir, "error", err)
		h.failCount.Add(1)
		return &scp.StoreResponse{Status: dimse.StatusOutOfResources}
	}

	// Write DICOM file
	if err := dicom.WriteFile(outputPath, req.DataSet); err != nil {
		h.logger.Error("Failed to write DICOM file", "path", outputPath, "error", err)
		h.failCount.Add(1)
		return &scp.StoreResponse{Status: dimse.StatusProcessingFailure}
	}

	h.logger.Info("Stored DICOM file",
		"path", outputPath,
		"sop_instance", req.SOPInstanceUID,
	)

	return &scp.StoreResponse{Status: dimse.StatusSuccess}
}

// Run executes the SCP server command.
func (c *SCPCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM SCP server")

	// Create output directory
	if err := createOutputDirectory(c.OutputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build SOP Class filter map
	sopClassMap := make(map[string]bool)
	if len(c.SOPClasses) > 0 {
		for _, uid := range c.SOPClasses {
			sopClassMap[uid] = true
		}
		logger.Info("SOP Class filter enabled", "classes", c.SOPClasses)
	} else {
		logger.Info("Accepting all SOP Classes")
	}

	// Create handlers
	echoHandler := &echoHandler{
		logger:     logger,
		acceptEcho: c.AcceptEcho,
	}

	storeHandler := &storeHandler{
		logger:     logger,
		outputDir:  c.OutputDir,
		organize:   c.Organize,
		sopClasses: sopClassMap,
	}

	// Build supported presentation contexts
	supportedContexts := c.buildSupportedContexts()

	// Create server config
	listenAddr := fmt.Sprintf(":%d", c.Port)
	serverConfig := scp.Config{
		AETitle:           c.AETitle,
		ListenAddr:        listenAddr,
		MaxPDULength:      c.MaxPDU,
		MaxAssociations:   c.MaxConns,
		SupportedContexts: supportedContexts,
		EchoHandler:       echoHandler,
		StoreHandler:      storeHandler,
	}

	// Create and start server
	server, err := scp.NewServer(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("SCP server listening",
		"address", listenAddr,
		"ae_title", c.AETitle,
		"max_pdu", c.MaxPDU,
		"max_conns", c.MaxConns,
	)

	if err := server.Listen(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Print server info
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✓ DICOM SCP server started"))
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Listen Address:"), ui.InfoStyle.Render(listenAddr))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("AE Title:"), ui.InfoStyle.Render(c.AETitle))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Output Directory:"), ui.InfoStyle.Render(c.OutputDir))
	if c.Organize {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Organization:"), ui.SuccessStyle.Render("Enabled (Study/Series/Instance)"))
	} else {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Organization:"), ui.SubtleStyle.Render("Disabled (flat structure)"))
	}
	if c.AcceptEcho {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("C-ECHO:"), ui.SuccessStyle.Render("Enabled"))
	} else {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("C-ECHO:"), ui.WarnStyle.Render("Disabled"))
	}
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Max Connections:"), ui.InfoStyle.Render(fmt.Sprintf("%d", c.MaxConns)))
	fmt.Println()
	fmt.Println(ui.SubtleStyle.Render("Press Ctrl+C to stop the server..."))
	fmt.Println()

	// Setup signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Print statistics periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			logger.Info("Shutdown signal received")
			cancel()

			// Shutdown server
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Warn("Server shutdown error", "error", err)
			}

			// Print final statistics
			fmt.Println()
			fmt.Println(ui.InfoStyle.Render("Server shutting down..."))
			fmt.Println()
			fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("C-ECHO Requests:"), ui.InfoStyle.Render(fmt.Sprintf("%d", echoHandler.requestCount.Load())))
			fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("C-STORE Requests:"), ui.InfoStyle.Render(fmt.Sprintf("%d", storeHandler.storeCount.Load())))
			fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Successful Stores:"), ui.SuccessStyle.Render(fmt.Sprintf("%d", storeHandler.storeCount.Load()-storeHandler.failCount.Load())))
			if storeHandler.failCount.Load() > 0 {
				fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Failed Stores:"), ui.ErrorStyle.Render(fmt.Sprintf("%d", storeHandler.failCount.Load())))
			}
			fmt.Println()

			return nil

		case <-ticker.C:
			logger.Info("Server statistics",
				"echo_count", echoHandler.requestCount.Load(),
				"store_count", storeHandler.storeCount.Load(),
				"store_failures", storeHandler.failCount.Load(),
			)
		}
	}
}

// buildSupportedContexts creates the supported presentation contexts map.
func (c *SCPCmd) buildSupportedContexts() map[string][]string {
	transferSyntaxes := []string{
		"1.2.840.10008.1.2",     // Implicit VR Little Endian
		"1.2.840.10008.1.2.1",   // Explicit VR Little Endian
		"1.2.840.10008.1.2.2",   // Explicit VR Big Endian
		"1.2.840.10008.1.2.4.90", // JPEG 2000 Lossless
		"1.2.840.10008.1.2.4.91", // JPEG 2000
	}

	contexts := make(map[string][]string)

	// Always support Verification SOP Class for C-ECHO
	if c.AcceptEcho {
		contexts["1.2.840.10008.1.1"] = transferSyntaxes
	}

	// Support specified SOP Classes or common ones
	if len(c.SOPClasses) > 0 {
		for _, sopClass := range c.SOPClasses {
			contexts[sopClass] = transferSyntaxes
		}
	} else {
		// Support common SOP Classes
		commonSOPClasses := []string{
			"1.2.840.10008.5.1.4.1.1.7",     // Secondary Capture
			"1.2.840.10008.5.1.4.1.1.1",     // CR Image Storage
			"1.2.840.10008.5.1.4.1.1.2",     // CT Image Storage
			"1.2.840.10008.5.1.4.1.1.4",     // MR Image Storage
			"1.2.840.10008.5.1.4.1.1.6.1",   // US Image Storage
			"1.2.840.10008.5.1.4.1.1.481.1", // RT Image Storage
		}

		for _, sopClass := range commonSOPClasses {
			contexts[sopClass] = transferSyntaxes
		}
	}

	return contexts
}

// extractOrganizationUIDs extracts Study and Series UIDs from a dataset.
func extractOrganizationUIDs(dataset *dicom.DataSet) (studyUID, seriesUID string, err error) {
	for _, elem := range dataset.Elements() {
		tag := elem.Tag()
		if tag.Group == 0x0020 && tag.Element == 0x000D {
			studyUID = elem.Value().String()
		}
		if tag.Group == 0x0020 && tag.Element == 0x000E {
			seriesUID = elem.Value().String()
		}
	}

	if studyUID == "" {
		return "", "", fmt.Errorf("study instance UID (0020,000D) not found")
	}
	if seriesUID == "" {
		return "", "", fmt.Errorf("series instance UID (0020,000E) not found")
	}

	return studyUID, seriesUID, nil
}
