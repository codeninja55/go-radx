package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Paths      []string      `arg:"" optional:"" type:"path" help:"DICOM files or directories to store"`
	Recursive  bool          `name:"recursive" short:"R" help:"Recursively search directories"`
	Host       string        `name:"host" required:"" help:"DICOM server hostname or IP address"`
	Port       int           `name:"port" default:"11112" help:"DICOM server port"`
	CalledAE   string        `name:"called-ae" default:"ANY-SCP" help:"Called AE Title (server)"`
	CallingAE  string        `name:"calling-ae" default:"RADX" help:"Calling AE Title (client)"`
	Timeout    time.Duration `name:"timeout" default:"5m" help:"Operation timeout"`
	MaxPDUSize uint32        `name:"max-pdu" default:"16384" help:"Maximum PDU size in bytes"`

	// Error handling options
	FailFast bool `name:"fail-fast" help:"Exit immediately if any files have unsupported SOP classes instead of attempting all files"`

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

	if len(c.Paths) == 0 {
		return fmt.Errorf("no input paths specified")
	}

	logger.Debug("Processing paths", "count", len(c.Paths))
	for _, path := range c.Paths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat path %s: %w", path, err)
		}

		if info.IsDir() {
			// Path is a directory - scan for DICOM files
			logger.Debug("Scanning directory", "path", path, "recursive", c.Recursive)
			dirFiles, err := listDicomFiles(path, c.Recursive)
			if err != nil {
				return fmt.Errorf("failed to list DICOM files in %s: %w", path, err)
			}
			files = append(files, dirFiles...)
		} else {
			// Path is a file - add directly
			files = append(files, DICOMFile{
				Path: path,
				Name: filepath.Base(path),
				Size: info.Size(),
			})
		}
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

	// Validate that all files have accepted presentation contexts
	logger.Debug("Validating presentation contexts for all files")
	sopClassMap := make(map[string]struct{})
	var rejectedSOPs []string

	for _, file := range files {
		dataset, err := dicom.ParseFile(file.Path)
		if err != nil {
			logger.Warn("Failed to parse file for validation", "file", file.Path, "error", err)
			continue
		}

		sopClassUID, _, err := extractSOPIdentifiers(dataset)
		if err != nil {
			logger.Warn("Failed to extract SOP Class UID for validation", "file", file.Path, "error", err)
			continue
		}

		// Track unique SOP classes and check if they're accepted
		if _, seen := sopClassMap[sopClassUID]; !seen {
			sopClassMap[sopClassUID] = struct{}{}
			exists, accepted, result := client.GetAssociation().GetPresentationContextResult(sopClassUID)
			if exists && !accepted {
				// Log detailed rejection reason at debug level
				var reason string
				switch result {
				case 3: // PresentationContextAbstractSyntaxNotSupported
					reason = "abstract syntax not supported"
				case 4: // PresentationContextTransferSyntaxesNotSupported
					reason = "transfer syntaxes not supported"
				case 1: // PresentationContextUserRejection
					reason = "user rejection"
				case 2: // PresentationContextProviderRejection
					reason = "provider rejection"
				default:
					reason = fmt.Sprintf("unknown reason (result=0x%02X)", result)
				}
				logger.Debug("Presentation context rejected",
					"sop_class_uid", sopClassUID,
					"reason", reason,
					"result_code", result)
				rejectedSOPs = append(rejectedSOPs, sopClassUID)
			} else if exists && accepted {
				logger.Debug("Presentation context accepted", "sop_class_uid", sopClassUID)
			}
		}
	}

	if len(rejectedSOPs) > 0 {
		if c.FailFast {
			logger.Error("Server does not support required SOP Classes", "unsupported_sop_classes", rejectedSOPs)
			return fmt.Errorf("association succeeded but server does not support required SOP Classes: %v", rejectedSOPs)
		} else {
			logger.Warn("Some files have unsupported SOP Classes and will fail during transfer",
				"rejected_classes", rejectedSOPs,
				"action", "Will attempt all files (use --fail-fast to exit early)")
		}
	} else {
		logger.Debug("All presentation contexts validated successfully")
	}

	// Store files with progress tracking
	progress := ui.NewProgressBar(len(files), "Storing")
	var successCount, failCount, reconnectCount, skippedCount atomic.Uint32
	startTime := time.Now()

	// Track ABORT counts per file to avoid infinite retry loops
	abortCounts := make(map[string]int)
	const maxAbortsPerFile = 2 // Maximum ABORTs before skipping a file

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

		// Check if this file has exceeded ABORT threshold
		if abortCounts[file.Path] >= maxAbortsPerFile {
			logger.Warn("Skipping file that consistently causes SCP to abort",
				"file", file.Path,
				"abort_count", abortCounts[file.Path],
				"reason", "File may be malformed or unsupported by SCP")
			skippedCount.Add(1)
			failCount.Add(1)
			continue
		}

		// Perform C-STORE with automatic reconnection on connection errors
		maxRetries := 1 // One retry after reconnection
		var storeErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			storeErr = client.Store(ctx, dataset, sopClassUID, sopInstanceUID)
			if storeErr == nil {
				// Success
				successCount.Add(1)
				logger.Debug("Stored file", "file", file.Name, "index", i+1, "total", len(files))
				break
			}

			// Track if this is an ABORT error
			if isAbortError(storeErr) {
				abortCounts[file.Path]++
				logger.Debug("File caused SCP ABORT", "file", file.Path, "abort_count", abortCounts[file.Path])
			}

			// Check if error is a connection error
			if !isConnectionError(storeErr) {
				// Not a connection error, don't retry
				logger.Error("C-STORE failed", "file", file.Path, "error", storeErr)
				failCount.Add(1)
				break
			}

			// Check if this file has now exceeded ABORT threshold after this error
			if abortCounts[file.Path] >= maxAbortsPerFile {
				logger.Warn("File consistently causes SCP to abort, skipping further attempts",
					"file", file.Path,
					"abort_count", abortCounts[file.Path],
					"error", storeErr)
				failCount.Add(1)
				skippedCount.Add(1)
				break
			}

			// Connection error - attempt to reconnect if we haven't exceeded retries
			if attempt < maxRetries {
				logger.Warn("Connection error detected, attempting reconnection", "file", file.Path, "error", storeErr)
				newClient, err := c.reconnectClient(ctx, client, clientConfig, logger, remoteAddr)
				if err != nil {
					logger.Error("Reconnection failed, skipping file", "file", file.Path, "error", err)
					failCount.Add(1)
					break
				}
				client = newClient
				reconnectCount.Add(1)
				logger.Info("Retrying file after reconnection", "file", file.Path)
				continue
			}

			// Exceeded retries
			logger.Error("C-STORE failed after reconnection attempt", "file", file.Path, "error", storeErr)
			failCount.Add(1)
		}
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
	if skippedCount.Load() > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Skipped (ABORT):"), ui.WarnStyle.Render(fmt.Sprintf("%d", skippedCount.Load())))
	}
	if reconnectCount.Load() > 0 {
		fmt.Printf("  %s %s\n", ui.SubtleStyle.Render("Reconnections:"), ui.WarnStyle.Render(fmt.Sprintf("%d", reconnectCount.Load())))
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
		"skipped_abort", skippedCount.Load(),
		"reconnections", reconnectCount.Load(),
		"elapsed", elapsed,
	)

	if failCount.Load() > 0 && c.FailFast {
		return fmt.Errorf("C-STORE completed with %d failures", failCount.Load())
	}

	return nil
}

// buildPresentationContexts creates presentation contexts for the files to be stored.
func (c *CStoreCmd) buildPresentationContexts(files []DICOMFile, logger *log.Logger) []dul.PresentationContextRQ {
	// Common transfer syntaxes
	transferSyntaxes := []string{
		"1.2.840.10008.1.2",      // Implicit VR Little Endian
		"1.2.840.10008.1.2.1",    // Explicit VR Little Endian
		"1.2.840.10008.1.2.2",    // Explicit VR Big Endian
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

// isConnectionError checks if an error indicates a broken connection that requires reconnection.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for connection-related error messages
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "aborted association") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "connection refused") ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

// isAbortError checks if an error is specifically an SCP-initiated ABORT.
func isAbortError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "aborted association")
}

// reconnectClient attempts to reconnect the DICOM client.
func (c *CStoreCmd) reconnectClient(ctx context.Context, client *scu.Client, clientConfig scu.Config, logger *log.Logger, remoteAddr string) (*scu.Client, error) {
	logger.Info("Connection lost, attempting to reconnect", "address", remoteAddr)

	// Close existing connection (ignore errors)
	_ = client.Close(ctx)

	// Create new client
	newClient := scu.NewClient(clientConfig)

	// Attempt to connect with retries
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Debug("Reconnection attempt", "attempt", attempt, "max", maxRetries)

		if err := newClient.Connect(ctx); err != nil {
			logger.Warn("Reconnection attempt failed", "attempt", attempt, "error", err)
			if attempt < maxRetries {
				// Exponential backoff: 1s, 2s, 4s
				backoff := time.Duration(1<<uint(attempt-1)) * time.Second
				logger.Debug("Waiting before retry", "backoff", backoff)
				time.Sleep(backoff)
				continue
			}
			return nil, fmt.Errorf("failed to reconnect after %d attempts: %w", maxRetries, err)
		}

		logger.Info("Reconnection successful", "attempt", attempt)
		return newClient, nil
	}

	return nil, fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}
