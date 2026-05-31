package dicom

import "testing"

func TestReadConfigDefaults(t *testing.T) {
	cfg := newReadConfig()
	if cfg.maxElementLen != defaultMaxElementLen {
		t.Errorf("maxElementLen = %d, want %d", cfg.maxElementLen, defaultMaxElementLen)
	}
	if cfg.maxSequenceDepth != defaultMaxSequenceDepth {
		t.Errorf("maxSequenceDepth = %d, want %d", cfg.maxSequenceDepth, defaultMaxSequenceDepth)
	}
	if cfg.stopAtPixelData {
		t.Error("stopAtPixelData should default false")
	}
}

func TestReadOptionsApply(t *testing.T) {
	cfg := newReadConfig(
		WithMaxElementLen(4096),
		WithMaxSequenceDepth(8),
		WithStopAtPixelData(),
	)
	if cfg.maxElementLen != 4096 {
		t.Errorf("maxElementLen = %d, want 4096", cfg.maxElementLen)
	}
	if cfg.maxSequenceDepth != 8 {
		t.Errorf("maxSequenceDepth = %d, want 8", cfg.maxSequenceDepth)
	}
	if !cfg.stopAtPixelData {
		t.Error("stopAtPixelData should be true after WithStopAtPixelData")
	}
}

func TestFileMetaZeroValue(t *testing.T) {
	var m FileMeta
	if m.TransferSyntaxUID != "" {
		t.Errorf("zero FileMeta TransferSyntaxUID = %q, want empty", m.TransferSyntaxUID)
	}
}
