package logging_test

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/codeninja55/go-radx/logging"
)

// phiSentinel is a clearly-synthetic stand-in for a patient value. It must never
// appear in any log output the field helpers produce.
const phiSentinel = "SENTINEL^PHI^DONOTLOG"

func TestFieldHelpersLogStructureNotPHI(t *testing.T) {
	tests := []struct {
		name       string
		field      zap.Field
		wantTokens []string
		wantAbsent []string
	}{
		{
			name:       "dicom tag renders keyword and coordinate",
			field:      logging.DICOMTag(0x0010, 0x0010, "PatientName"),
			wantTokens: []string{"PatientName", "(0010,0010)", "dicom_tag"},
			wantAbsent: []string{phiSentinel},
		},
		{
			name:       "hl7 field renders segment-field locator",
			field:      logging.HL7Field("PID", 5),
			wantTokens: []string{"PID", "PID-5", "hl7_field"},
			wantAbsent: []string{phiSentinel},
		},
		{
			name:       "fhir path renders element path",
			field:      logging.FHIRPath("Patient.name.family"),
			wantTokens: []string{"Patient.name.family", "fhir_path"},
			wantAbsent: []string{phiSentinel},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			logger.Info("processing element", tc.field)

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("expected exactly 1 entry, got %d", len(entries))
			}

			rendered := renderEntry(t, entries[0])
			for _, want := range tc.wantTokens {
				if !strings.Contains(rendered, want) {
					t.Errorf("output %q missing structural token %q", rendered, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(rendered, absent) {
					t.Errorf("output %q leaked forbidden token %q", rendered, absent)
				}
			}
		})
	}
}

func TestFieldHelpersRedactNonStructuralInput(t *testing.T) {
	tests := []struct {
		name  string
		field zap.Field
	}{
		{
			name:  "dicom keyword carrying a patient value is redacted",
			field: logging.DICOMTag(0x0010, 0x0010, phiSentinel),
		},
		{
			name:  "hl7 segment carrying a raw segment dump is redacted",
			field: logging.HL7Field("PID|"+phiSentinel, 5),
		},
		{
			name:  "fhir path carrying a value literal is redacted",
			field: logging.FHIRPath("Patient.name.where(family='" + phiSentinel + "')"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			logger.Info("processing element", tc.field)

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("expected exactly 1 entry, got %d", len(entries))
			}

			rendered := renderEntry(t, entries[0])
			if strings.Contains(rendered, phiSentinel) {
				t.Errorf("output %q leaked PHI sentinel through a non-structural locator", rendered)
			}
			if !strings.Contains(rendered, "[redacted-non-structural]") {
				t.Errorf("output %q did not redact the non-structural locator", rendered)
			}
		})
	}
}

func TestHL7FieldRedactsNonPositiveIndex(t *testing.T) {
	tests := []struct {
		name  string
		field int
	}{
		{name: "zero index", field: 0},
		{name: "negative index", field: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			logger.Info("processing element", logging.HL7Field("PID", tc.field))

			rendered := renderEntry(t, logs.All()[0])
			if !strings.Contains(rendered, "[redacted-non-structural]") {
				t.Errorf("output %q did not redact field index %d", rendered, tc.field)
			}
		})
	}
}

func TestConcurrentLoggingDoesNotRace(t *testing.T) {
	logger, err := logging.NewLogger(io.Discard, logging.Config{})
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 64; j++ {
				logger.Info("concurrent", logging.DICOMTag(0x0010, 0x0010, "PatientName"))
			}
		}()
	}
	wg.Wait()
}

// TestShapeValidationAcceptsIdentifierShapedToken documents the deliberate
// boundary of shape validation: a bare identifier-shaped token is lexically
// indistinguishable from a real keyword, so it passes. Closing this requires the
// caller to bind the locator to the canonical vocabulary at the domain-package
// boundary; this package stays a dependency-free leaf and cannot embed those
// dictionaries.
func TestShapeValidationAcceptsIdentifierShapedToken(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	logger.Info("processing element", logging.DICOMTag(0x0010, 0x0010, "Smith"))

	rendered := renderEntry(t, logs.All()[0])
	if !strings.Contains(rendered, "Smith") {
		t.Errorf("identifier-shaped token unexpectedly redacted: %q", rendered)
	}
}

// renderEntry encodes a captured entry to JSON so the assertions see exactly the
// bytes a real sink would receive, including the structured fields.
func renderEntry(t *testing.T, e observer.LoggedEntry) string {
	t.Helper()
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	buf, err := encoder.EncodeEntry(e.Entry, e.Context)
	if err != nil {
		t.Fatalf("encoding entry: %v", err)
	}
	return buf.String()
}

func TestLoggerComesFromContext(t *testing.T) {
	t.Run("FromContext returns the injected logger", func(t *testing.T) {
		injected := zap.NewExample()
		ctx := logging.WithContext(context.Background(), injected)

		if got := logging.FromContext(ctx); got != injected {
			t.Errorf("FromContext = %p, want injected logger %p", got, injected)
		}
	})

	t.Run("FromContext on a bare context returns a non-nil no-op logger", func(t *testing.T) {
		got := logging.FromContext(context.Background())
		if got == nil {
			t.Fatal("FromContext returned nil; want a no-op logger")
		}
		// A no-op logger must not panic and must drop entries silently.
		got.Info("should be discarded", logging.FHIRPath("Patient.id"))
	})

	t.Run("WithContext replaces a nil logger with a no-op", func(t *testing.T) {
		ctx := logging.WithContext(context.Background(), nil)
		if got := logging.FromContext(ctx); got == nil {
			t.Fatal("FromContext returned nil after WithContext(nil); want a no-op logger")
		}
	})
}
