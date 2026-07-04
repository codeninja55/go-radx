package dimse

import (
	"errors"
	"testing"
	"time"
)

func TestNewAEDefaults(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	cfg := ae.config()
	// pynetdicom-default timeouts (dimse.md "Application Entity").
	if cfg.acseTimeout != 30*time.Second {
		t.Errorf("default ACSE timeout = %v, want 30s", cfg.acseTimeout)
	}
	if cfg.dimseTimeout != 30*time.Second {
		t.Errorf("default DIMSE timeout = %v, want 30s", cfg.dimseTimeout)
	}
	if cfg.networkTimeout != 60*time.Second {
		t.Errorf("default network timeout = %v, want 60s", cfg.networkTimeout)
	}
	if cfg.maxPDULength != 16382 {
		t.Errorf("default max PDU length = %d, want 16382", cfg.maxPDULength)
	}
	if ae.Title() != AETitle("RADX-SCU") {
		t.Errorf("AE title = %q, want RADX-SCU", ae.Title())
	}
}

func TestNewAEValidatesTitle(t *testing.T) {
	if _, err := NewAE(AETitle("")); err == nil {
		t.Error("NewAE with an empty title = nil error, want rejection")
	}
	_, err := NewAE(AETitle("WAY-TOO-LONG-AE-TITLE-OVER-16"))
	if err == nil {
		t.Fatal("NewAE with a 17+ char title = nil error, want rejection")
	}
	if _, ok := errors.AsType[*ValidationError](err); !ok {
		t.Errorf("NewAE error = %T, want *ValidationError", err)
	}
}

func TestAEOptionsOverrideDefaults(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCU"),
		WithACSETimeout(10*time.Second),
		WithDIMSETimeout(15*time.Second),
		WithNetworkTimeout(45*time.Second),
		WithMaxPDULength(MaxPDULength(0)), // 0 = unlimited (DIMSE-005)
	)
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	cfg := ae.config()
	if cfg.acseTimeout != 10*time.Second {
		t.Errorf("ACSE timeout = %v, want 10s", cfg.acseTimeout)
	}
	if cfg.dimseTimeout != 15*time.Second {
		t.Errorf("DIMSE timeout = %v, want 15s", cfg.dimseTimeout)
	}
	if cfg.networkTimeout != 45*time.Second {
		t.Errorf("network timeout = %v, want 45s", cfg.networkTimeout)
	}
	if cfg.maxPDULength != 0 {
		t.Errorf("max PDU length = %d, want 0 (unlimited)", cfg.maxPDULength)
	}
}
