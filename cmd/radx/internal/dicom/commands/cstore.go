package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/codeninja55/go-radx/cmd/radx/internal/config"
	"github.com/codeninja55/go-radx/cmd/radx/internal/dicom/ui"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/pixel"
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
	MaxPDUSize uint32        `name:"max-pdu" default:"65536" help:"Maximum PDU size in bytes (max: 131072)"`

	// Error handling options
	FailFast bool `name:"fail-fast" help:"Exit immediately if any files have unsupported SOP classes instead of attempting all files"`

	// Rate limiting options
	RateLimit      float64 `name:"rate-limit" help:"Rate limit in files/second (0 = unlimited)" default:"0"`
	RateLimitBytes float64 `name:"rate-limit-bytes" help:"Rate limit in MB/second (0 = unlimited)" default:"0"`
	BurstSize      int     `name:"burst" help:"Burst size for rate limiting" default:"10"`

	// Connection recovery options
	ReconnectDelay time.Duration `name:"reconnect-delay" help:"Delay after reconnection before retry" default:"2s"`

	// Connection pooling options
	Workers int `name:"workers" default:"4" help:"Number of concurrent worker connections (1-128)"`

	// Transcoding options
	Transcode bool `name:"transcode" default:"true" help:"Automatically transcode uncompressed images to JPEG 2000 Lossless for SCP compatibility"`
}

// fileJob represents a file to be processed by a worker.
type fileJob struct {
	file  DICOMFile
	index int
}

// workerState holds shared state and configuration for workers.
type workerState struct {
	// Configuration
	cmd          *CStoreCmd
	clientConfig scu.Config
	remoteAddr   string
	logger       *log.Logger
	maxAbortsPerFile int

	// Shared state (atomic counters)
	successCount   *atomic.Uint32
	failCount      *atomic.Uint32
	reconnectCount *atomic.Uint32
	skippedCount   *atomic.Uint32

	// Shared state (mutex-protected)
	abortCounts    map[string]int
	abortCountsMux *sync.Mutex
	progress       *ui.ProgressBar
	progressMux    *sync.Mutex

	// Rate limiters (thread-safe by design)
	fileLimiter *rate.Limiter
	byteLimiter *rate.Limiter

	// Job distribution
	jobs chan fileJob
	ctx  context.Context
}

// worker processes files from the job channel using a dedicated DICOM connection.
// Each worker maintains its own persistent association for all files it processes.
func worker(id int, state *workerState) {
	logger := state.logger.With("worker_id", id)
	logger.Debug("Worker starting")

	// Create dedicated client for this worker
	client := scu.NewClient(state.clientConfig)

	// Establish connection
	if err := client.Connect(state.ctx); err != nil {
		logger.Error("Worker failed to connect", "error", err)
		// Drain remaining jobs and mark as failed
		for job := range state.jobs {
			state.failCount.Add(1)
			state.progressMux.Lock()
			state.progress.Increment(fmt.Sprintf("Storing %s (worker connection failed)", job.file.Name))
			state.progressMux.Unlock()
		}
		return
	}

	// Ensure connection is closed when worker exits
	defer func() {
		if err := client.Close(state.ctx); err != nil {
			logger.Warn("Worker failed to close connection", "error", err)
		}
		logger.Debug("Worker stopped")
	}()

	logger.Debug("Worker connection established")

	// Process jobs from channel
	for job := range state.jobs {
		state.processFile(id, client, job, logger)
	}
}

// processFile handles a single file transfer with automatic transcoding if needed.
func (state *workerState) processFile(workerID int, client *scu.Client, job fileJob, logger *log.Logger) {
	file := job.file

	// Update progress
	state.progressMux.Lock()
	state.progress.Increment(fmt.Sprintf("Storing %s", file.Name))
	state.progressMux.Unlock()

	// Apply rate limiting
	if state.fileLimiter != nil {
		if err := state.fileLimiter.Wait(state.ctx); err != nil {
			logger.Error("Rate limiter error", "file", file.Path, "error", err)
			state.failCount.Add(1)
			return
		}
	}

	if state.byteLimiter != nil {
		if err := state.byteLimiter.WaitN(state.ctx, int(file.Size)); err != nil {
			logger.Error("Byte rate limiter error", "file", file.Path, "error", err)
			state.failCount.Add(1)
			return
		}
	}

	// Check if file has exceeded ABORT threshold
	state.abortCountsMux.Lock()
	abortCount := state.abortCounts[file.Path]
	state.abortCountsMux.Unlock()

	if abortCount >= state.maxAbortsPerFile {
		logger.Warn("Skipping file that consistently causes SCP to abort",
			"file", file.Path,
			"abort_count", abortCount,
			"reason", "File may be malformed or unsupported by SCP")
		state.skippedCount.Add(1)
		state.failCount.Add(1)
		return
	}

	// Parse DICOM file
	dataset, err := dicom.ParseFile(file.Path)
	if err != nil {
		logger.Error("Failed to parse DICOM file", "file", file.Path, "error", err)
		state.failCount.Add(1)
		return
	}

	// Apply transcoding if enabled and needed
	if state.cmd.Transcode {
		transcoded, err := pixel.TranscodeToJPEG2000Lossless(dataset)
		if err != nil {
			logger.Error("Failed to transcode file", "file", file.Path, "error", err)
			state.failCount.Add(1)
			return
		}
		if transcoded {
			logger.Debug("Transcoded file to JPEG 2000 Lossless", "file", file.Path)
		}
	}

	// Extract SOP identifiers
	sopClassUID, sopInstanceUID, err := extractSOPIdentifiers(dataset)
	if err != nil {
		logger.Error("Failed to extract SOP identifiers", "file", file.Path, "error", err)
		state.failCount.Add(1)
		return
	}

	// Perform C-STORE with retry logic
	maxRetries := 1
	var storeErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		storeErr = client.Store(state.ctx, dataset, sopClassUID, sopInstanceUID)

		if storeErr == nil {
			// Success
			state.successCount.Add(1)
			logger.Debug("Stored file", "file", file.Name, "index", job.index+1)
			return
		}

		// Track ABORT errors
		if isAbortError(storeErr) {
			state.abortCountsMux.Lock()
			state.abortCounts[file.Path]++
			currentAborts := state.abortCounts[file.Path]
			state.abortCountsMux.Unlock()

			logger.Debug("File caused SCP ABORT", "file", file.Path, "abort_count", currentAborts)

			// Check if threshold exceeded
			if currentAborts >= state.maxAbortsPerFile {
				logger.Warn("File consistently causes SCP to abort, skipping",
					"file", file.Path,
					"abort_count", currentAborts,
					"error", storeErr)
				state.failCount.Add(1)
				state.skippedCount.Add(1)
				return
			}
		}

		// Check if error is connection error
		if !isConnectionError(storeErr) {
			// Not a connection error, don't retry
			logger.Error("C-STORE failed", "file", file.Path, "error", storeErr)
			state.failCount.Add(1)
			return
		}

		// Connection error - attempt reconnection if retries remain
		if attempt < maxRetries {
			logger.Warn("Connection error, attempting reconnection", "file", file.Path, "error", storeErr)
			newClient, err := state.cmd.reconnectClient(state.ctx, client, state.clientConfig, logger, state.remoteAddr)
			if err != nil {
				logger.Error("Reconnection failed, skipping file", "file", file.Path, "error", err)
				state.failCount.Add(1)
				return
			}
			client = newClient
			state.reconnectCount.Add(1)

			// Wait before retry
			if state.cmd.ReconnectDelay > 0 {
				logger.Debug("Waiting before retry", "delay", state.cmd.ReconnectDelay)
				time.Sleep(state.cmd.ReconnectDelay)
			}

			logger.Info("Retrying file after reconnection", "file", file.Path)
			continue
		}

		// Exceeded retries
		logger.Error("C-STORE failed after reconnection", "file", file.Path, "error", storeErr)
		state.failCount.Add(1)
	}
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
		"reconnect_delay", c.ReconnectDelay,
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

	// Validate workers count
	if c.Workers < 1 || c.Workers > 128 {
		logger.Error("Invalid workers count", "workers", c.Workers)
		return fmt.Errorf("workers must be between 1 and 128, got %d", c.Workers)
	}

	logger.Info("Worker configuration", "workers", c.Workers, "transcode", c.Transcode)

	// Create presentation contexts for common SOP Classes
	presentationContexts := c.buildPresentationContexts(files, logger)

	// Create SCU client config (used by all workers)
	clientConfig := scu.Config{
		CallingAETitle:       c.CallingAE,
		CalledAETitle:        c.CalledAE,
		RemoteAddr:           remoteAddr,
		MaxPDULength:         c.MaxPDUSize,
		PresentationContexts: presentationContexts,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// Initialize shared state
	progress := ui.NewProgressBar(len(files), "Storing")
	var successCount, failCount, reconnectCount, skippedCount atomic.Uint32
	abortCounts := make(map[string]int)
	abortCountsMux := &sync.Mutex{}
	progressMux := &sync.Mutex{}
	const maxAbortsPerFile = 2

	// Create job channel (buffered to avoid blocking)
	jobs := make(chan fileJob, len(files))

	// Create worker state
	state := &workerState{
		cmd:              c,
		clientConfig:     clientConfig,
		remoteAddr:       remoteAddr,
		logger:           logger,
		maxAbortsPerFile: maxAbortsPerFile,
		successCount:     &successCount,
		failCount:        &failCount,
		reconnectCount:   &reconnectCount,
		skippedCount:     &skippedCount,
		abortCounts:      abortCounts,
		abortCountsMux:   abortCountsMux,
		progress:         progress,
		progressMux:      progressMux,
		fileLimiter:      fileLimiter,
		byteLimiter:      byteLimiter,
		jobs:             jobs,
		ctx:              ctx,
	}

	// Start workers
	logger.Info("Starting worker pool", "workers", c.Workers)
	var wg sync.WaitGroup
	startTime := time.Now()

	for workerID := 1; workerID <= c.Workers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(id, state)
		}(workerID)
	}

	// Send jobs to workers
	logger.Debug("Distributing files to workers", "file_count", len(files))
	for i, file := range files {
		jobs <- fileJob{
			file:  file,
			index: i,
		}
	}

	// Close job channel to signal workers to exit after completing their work
	close(jobs)
	logger.Debug("All jobs distributed, waiting for workers to complete")

	// Wait for all workers to finish
	wg.Wait()
	logger.Debug("All workers completed")

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
