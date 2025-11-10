package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scu"
	"golang.org/x/time/rate"
)

// CStoreCmd implements the DICOM C-STORE command.
type CStoreCmd struct {
	Paths      []string      `arg:"" optional:"" type:"existingfile" help:"DICOM files to store" group:"Input"`
	Dir        string        `name:"dir" type:"existingdir" help:"Directory containing DICOM files" group:"Input" xor:"Input"`
	Recursive  bool          `name:"recursive" short:"R" help:"Recursively search directories"`
	Host       string        `name:"host" required:"" help:"DICOM server hostname or IP address"`
	Port       int           `name:"port" default:"11112" help:"DICOM server port"`
	CalledAE   string        `name:"called-ae" default:"ANY-SCP" help:"Called AE Title (server)"`
	CallingAE  string        `name:"calling-ae" default:"RADX" help:"Calling AE Title (client)"`
	Timeout    time.Duration `name:"timeout" default:"5m" help:"Operation timeout"`
	MaxPDUSize uint32        `name:"max-pdu" default:"16384" help:"Maximum PDU size in bytes"`

	// Rate limiting options
	RateLimit      float64 `name:"rate-limit" help:"Rate limit in files/second (0 = unlimited)" default:"0"`
	RateLimitBytes float64 `name:"rate-limit-bytes" help:"Rate limit in MB/second (0 = unlimited)" default:"0"`
	BurstSize      int     `name:"burst" help:"Burst size for rate limiting" default:"10"`
}

// Run executes the C-STORE command.
func (c *CStoreCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM C-STORE operation")

	// Collect DICOM files
	var files []DICOMFile
	var err error

	if c.Dir != "" {
		logger.Debug("Scanning directory", "path", c.Dir, "recursive", c.Recursive)
		files, err = listDicomFiles(c.Dir, c.Recursive)
		if err != nil {
			return fmt.Errorf("failed to list DICOM files: %w", err)
		}
	} else if len(c.Paths) > 0 {
		logger.Debug("Processing files", "count", len(c.Paths))
		for _, path := range c.Paths {
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to stat file %s: %w", path, err)
			}
			files = append(files, DICOMFile{
				Path: path,
				Name: filepath.Base(path),
				Size: info.Size(),
			})
		}
	} else {
		return fmt.Errorf("no input files specified (use paths or --dir)")
	}

	if len(files) == 0 {
		logger.Warn("No DICOM files found")
		return nil
	}

	logger.Info("Found DICOM files", "count", len(files))

	// Create remote address
	remoteAddr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	logger.Debug("C-STORE parameters",
		"host", c.Host,
		"port", c.Port,
		"calling_ae", c.CallingAE,
		"called_ae", c.CalledAE,
		"timeout", c.Timeout,
		"max_pdu", c.MaxPDUSize,
		"rate_limit", c.RateLimit,
		"rate_limit_bytes", c.RateLimitBytes,
		"burst", c.BurstSize,
	)

	// Create rate limiters
	var fileLimiter *rate.Limiter
	var byteLimiter *rate.Limiter

	if c.RateLimit > 0 {
		fileLimiter = rate.NewLimiter(rate.Limit(c.RateLimit), c.BurstSize)
		logger.Info("File rate limiting enabled", "files_per_sec", c.RateLimit, "burst", c.BurstSize)
	}

	if c.RateLimitBytes > 0 {
		bytesPerSec := c.RateLimitBytes * 1024 * 1024 // Convert MB/s to bytes/s
		burstBytes := int(bytesPerSec) * c.BurstSize
		byteLimiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstBytes)
		logger.Info("Byte rate limiting enabled", "mb_per_sec", c.RateLimitBytes, "burst_mb", c.BurstSize)
	}

	// Create presentation contexts for common SOP Classes
	presentationContexts := c.buildPresentationContexts(files, logger)

	// Create SCU client
	clientConfig := scu.Config{
		CallingAETitle:       c.CallingAE,
		CalledAETitle:        c.CalledAE,
		RemoteAddr:           remoteAddr,
		MaxPDULength:         c.MaxPDUSize,
		PresentationContexts: presentationContexts,
	}

	client := scu.NewClient(clientConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// Connect to server
	logger.Info("Connecting to DICOM server", "address", remoteAddr)
	spinner := ui.NewSpinner("Connecting")
	spinner.Tick("Establishing association...")

	if err := client.Connect(ctx); err != nil {
		spinner.Stop()
		logger.Error("Failed to connect", "error", err)
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() {
		if err := client.Close(ctx); err != nil {
			logger.Warn("Failed to close connection", "error", err)
		}
	}()

	spinner.Stop()
	logger.Info("Association established successfully")

	// Store files with progress tracking
	progress := ui.NewProgressBar(len(files), "Storing")
	var successCount, failCount atomic.Uint32
	startTime := time.Now()

	for i, file := range files {
		progress.Increment(fmt.Sprintf("Storing %s", file.Name))

		// Apply rate limiting
		if fileLimiter != nil {
			if err := fileLimiter.Wait(ctx); err != nil {
				logger.Error("Rate limiter error", "error", err)
				failCount.Add(1)
				continue
			}
		}

		if byteLimiter != nil {
			// Reserve tokens for file size
			if err := byteLimiter.WaitN(ctx, int(file.Size)); err != nil {
				logger.Error("Byte rate limiter error", "error", err)
				failCount.Add(1)
				continue
			}
		}

		// Parse DICOM file
		dataset, err := dicom.ParseFile(file.Path)
		if err != nil {
			logger.Error("Failed to parse DICOM file", "file", file.Path, "error", err)
			failCount.Add(1)
			continue
		}

		// Extract SOP Class UID and SOP Instance UID
		sopClassUID, sopInstanceUID, err := extractSOPIdentifiers(dataset)
		if err != nil {
			logger.Error("Failed to extract SOP identifiers", "file", file.Path, "error", err)
			failCount.Add(1)
			continue
		}

		// Perform C-STORE
		if err := client.Store(ctx, dataset, sopClassUID, sopInstanceUID); err != nil {
			logger.Error("C-STORE failed", "file", file.Path, "error", err)
			failCount.Add(1)
			continue
		}

		successCount.Add(1)
		logger.Debug("Stored file", "file", file.Name, "index", i+1, "total", len(files))
	}

	progress.Complete("Complete")
	elapsed := time.Since(startTime)

	// Print summary
	fmt.Println()
	if failCount.Load() == 0 {
		fmt.Println(ui.SuccessStyle.Render("✓ All files stored successfully!"))
	} else {
		fmt.Println(ui.WarnStyle.Render(fmt.Sprintf("⚠ Storage completed with %d failures", failCount.Load())))
	}
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Server:"), ui.InfoStyle.Render(remoteAddr))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Total Files:"), ui.InfoStyle.Render(fmt.Sprintf("%d", len(files))))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Successful:"), ui.SuccessStyle.Render(fmt.Sprintf("%d", successCount.Load())))
	if failCount.Load() > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Failed:"), ui.ErrorStyle.Render(fmt.Sprintf("%d", failCount.Load())))
	}
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Duration:"), ui.InfoStyle.Render(elapsed.Round(time.Millisecond).String()))
	if successCount.Load() > 0 {
		throughput := float64(successCount.Load()) / elapsed.Seconds()
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Throughput:"), ui.InfoStyle.Render(fmt.Sprintf("%.2f files/sec", throughput)))
	}
	fmt.Println()

	logger.Info("C-STORE operation complete",
		"total", len(files),
		"success", successCount.Load(),
		"failed", failCount.Load(),
		"elapsed", elapsed,
	)

	if failCount.Load() > 0 {
		return fmt.Errorf("C-STORE completed with %d failures", failCount.Load())
	}

	return nil
}

// buildPresentationContexts creates presentation contexts for the files to be stored.
func (c *CStoreCmd) buildPresentationContexts(files []DICOMFile, logger *log.Logger) []dul.PresentationContextRQ {
	// Common transfer syntaxes
	transferSyntaxes := []string{
		"1.2.840.10008.1.2",     // Implicit VR Little Endian
		"1.2.840.10008.1.2.1",   // Explicit VR Little Endian
		"1.2.840.10008.1.2.2",   // Explicit VR Big Endian
		"1.2.840.10008.1.2.4.90", // JPEG 2000 Lossless
		"1.2.840.10008.1.2.4.91", // JPEG 2000
	}

	// Collect unique SOP Class UIDs from files
	sopClassMap := make(map[string]bool)
	for _, file := range files {
		dataset, err := dicom.ParseFile(file.Path)
		if err != nil {
			logger.Warn("Failed to parse file for SOP Class", "file", file.Path, "error", err)
			continue
		}

		sopClassUID, _, err := extractSOPIdentifiers(dataset)
		if err != nil {
			logger.Warn("Failed to extract SOP Class UID", "file", file.Path, "error", err)
			continue
		}

		sopClassMap[sopClassUID] = true
	}

	// Build presentation contexts
	contexts := make([]dul.PresentationContextRQ, 0, len(sopClassMap))
	contextID := uint8(1)

	for sopClassUID := range sopClassMap {
		contexts = append(contexts, dul.PresentationContextRQ{
			ID:               contextID,
			AbstractSyntax:   sopClassUID,
			TransferSyntaxes: transferSyntaxes,
		})
		contextID += 2 // Presentation context IDs must be odd
	}

	logger.Debug("Built presentation contexts", "count", len(contexts))
	return contexts
}

// extractSOPIdentifiers extracts SOP Class UID and SOP Instance UID from a dataset.
func extractSOPIdentifiers(dataset *dicom.DataSet) (sopClassUID, sopInstanceUID string, err error) {
	// Find SOP Class UID (0008,0016)
	for _, elem := range dataset.Elements() {
		tag := elem.Tag()
		if tag.Group == 0x0008 && tag.Element == 0x0016 {
			sopClassUID = elem.Value().String()
		}
		if tag.Group == 0x0008 && tag.Element == 0x0018 {
			sopInstanceUID = elem.Value().String()
		}
	}

	if sopClassUID == "" {
		return "", "", fmt.Errorf("SOP Class UID (0008,0016) not found")
	}
	if sopInstanceUID == "" {
		return "", "", fmt.Errorf("SOP Instance UID (0008,0018) not found")
	}

	return sopClassUID, sopInstanceUID, nil
}
