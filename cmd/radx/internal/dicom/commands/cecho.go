package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scu"
)

// CEchoCmd implements the DICOM C-ECHO command.
type CEchoCmd struct {
	Host       string        `arg:"" required:"" help:"DICOM server hostname or IP address"`
	Port       int           `arg:"" default:"11112" help:"DICOM server port"`
	CalledAE   string        `name:"called-ae" default:"ANY-SCP" help:"Called AE Title (server)"`
	CallingAE  string        `name:"calling-ae" default:"RADX" help:"Calling AE Title (client)"`
	Timeout    time.Duration `name:"timeout" default:"30s" help:"Connection timeout"`
	MaxPDUSize uint32        `name:"max-pdu" default:"16384" help:"Maximum PDU size in bytes"`
}

// Run executes the C-ECHO command.
func (c *CEchoCmd) Run(cfg *config.GlobalConfig) error {
	// Print banner
	ui.PrintBanner()

	logger := log.Default()
	logger.Info("Starting DICOM C-ECHO verification")

	// Create remote address
	remoteAddr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	logger.Debug("C-ECHO parameters",
		"host", c.Host,
		"port", c.Port,
		"calling_ae", c.CallingAE,
		"called_ae", c.CalledAE,
		"timeout", c.Timeout,
		"max_pdu", c.MaxPDUSize,
	)

	// Create presentation context for Verification SOP Class
	presentationContexts := []dul.PresentationContextRQ{
		{
			ID:             1,
			AbstractSyntax: "1.2.840.10008.1.1", // Verification SOP Class
			TransferSyntaxes: []string{
				"1.2.840.10008.1.2",   // Implicit VR Little Endian
				"1.2.840.10008.1.2.1", // Explicit VR Little Endian
				"1.2.840.10008.1.2.2", // Explicit VR Big Endian
			},
		},
	}

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

	// Perform C-ECHO
	logger.Info("Sending C-ECHO request")
	spinner = ui.NewSpinner("Verifying")
	spinner.Tick("Sending C-ECHO...")

	startTime := time.Now()
	if err := client.Echo(ctx); err != nil {
		spinner.Stop()
		logger.Error("C-ECHO failed", "error", err)
		return fmt.Errorf("C-ECHO failed: %w", err)
	}
	elapsed := time.Since(startTime)

	spinner.Stop()

	// Print success message with styling
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✓ C-ECHO successful!"))
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Server:"), ui.InfoStyle.Render(remoteAddr))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Called AE:"), ui.InfoStyle.Render(c.CalledAE))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Calling AE:"), ui.InfoStyle.Render(c.CallingAE))
	fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Response Time:"), ui.InfoStyle.Render(elapsed.Round(time.Millisecond).String()))
	fmt.Println()

	logger.Info("C-ECHO verification complete",
		"elapsed", elapsed,
		"server", remoteAddr,
	)

	return nil
}
