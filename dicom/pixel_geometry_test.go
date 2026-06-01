package dicom

import "testing"

func TestResolvePixelGeometryFromImagePixelModule(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagRows, VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: TagColumns, VR: VRUS, Value: NewInts(VRUS, 6)})
	ds.Set(Element{Tag: TagSamplesPerPixel, VR: VRUS, Value: NewInts(VRUS, 3)})
	ds.Set(Element{Tag: TagBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 8)})
	ds.Set(Element{Tag: TagBitsStored, VR: VRUS, Value: NewInts(VRUS, 8)})
	ds.Set(Element{Tag: TagHighBit, VR: VRUS, Value: NewInts(VRUS, 7)})
	ds.Set(Element{Tag: TagPixelRepresentation, VR: VRUS, Value: NewInts(VRUS, 0)})
	ds.Set(Element{Tag: TagPlanarConfiguration, VR: VRUS, Value: NewInts(VRUS, 1)})
	ds.Set(Element{Tag: TagNumberOfFrames, VR: VRIS, Value: mustDecimals(t, "5")})
	ds.SetString(TagPhotometricInterpretation, "RGB")

	geom, err := ResolvePixelGeometry(ds, ExplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("ResolvePixelGeometry: %v", err)
	}

	if geom.Rows != 4 || geom.Columns != 6 {
		t.Errorf("dimensions = %dx%d, want 4x6", geom.Rows, geom.Columns)
	}
	if geom.SamplesPerPixel != 3 {
		t.Errorf("SamplesPerPixel = %d, want 3", geom.SamplesPerPixel)
	}
	if geom.BitsAllocated != 8 || geom.BitsStored != 8 || geom.HighBit != 7 {
		t.Errorf("bit fields = alloc %d stored %d high %d, want 8/8/7",
			geom.BitsAllocated, geom.BitsStored, geom.HighBit)
	}
	if geom.PixelRepresentation != 0 {
		t.Errorf("PixelRepresentation = %d, want 0", geom.PixelRepresentation)
	}
	if geom.PlanarConfiguration != 1 {
		t.Errorf("PlanarConfiguration = %d, want 1", geom.PlanarConfiguration)
	}
	if geom.NumberOfFrames != 5 {
		t.Errorf("NumberOfFrames = %d, want 5", geom.NumberOfFrames)
	}
	if geom.PhotometricInterpretation != "RGB" {
		t.Errorf("PhotometricInterpretation = %q, want RGB", geom.PhotometricInterpretation)
	}
	if geom.TransferSyntax != ExplicitVRLittleEndian {
		t.Errorf("TransferSyntax = %s, want Explicit VR LE", geom.TransferSyntax)
	}
}

func TestResolvePixelGeometryDefaults(t *testing.T) {
	// A minimal monochrome dataset omits SamplesPerPixel, NumberOfFrames, and
	// PlanarConfiguration; the resolver applies the PS3.3 defaults (1, 1, 0).
	ds := NewDataSet()
	ds.Set(Element{Tag: TagRows, VR: VRUS, Value: NewInts(VRUS, 512)})
	ds.Set(Element{Tag: TagColumns, VR: VRUS, Value: NewInts(VRUS, 512)})
	ds.Set(Element{Tag: TagBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 16)})
	ds.Set(Element{Tag: TagBitsStored, VR: VRUS, Value: NewInts(VRUS, 12)})
	ds.Set(Element{Tag: TagHighBit, VR: VRUS, Value: NewInts(VRUS, 11)})

	geom, err := ResolvePixelGeometry(ds, ImplicitVRLittleEndian)
	if err != nil {
		t.Fatalf("ResolvePixelGeometry: %v", err)
	}
	if geom.SamplesPerPixel != 1 {
		t.Errorf("SamplesPerPixel default = %d, want 1", geom.SamplesPerPixel)
	}
	if geom.NumberOfFrames != 1 {
		t.Errorf("NumberOfFrames default = %d, want 1", geom.NumberOfFrames)
	}
	if geom.PlanarConfiguration != 0 {
		t.Errorf("PlanarConfiguration default = %d, want 0", geom.PlanarConfiguration)
	}
}

func TestResolvePixelGeometryRejectsMissingDimensions(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 8)})

	if _, err := ResolvePixelGeometry(ds, ExplicitVRLittleEndian); err == nil {
		t.Fatal("expected an error for a dataset missing Rows/Columns")
	}
}

func TestResolvePixelGeometryRejectsZeroBitsAllocated(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagRows, VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: TagColumns, VR: VRUS, Value: NewInts(VRUS, 4)})
	ds.Set(Element{Tag: TagBitsAllocated, VR: VRUS, Value: NewInts(VRUS, 0)})

	if _, err := ResolvePixelGeometry(ds, ExplicitVRLittleEndian); err == nil {
		t.Fatal("expected an error for BitsAllocated == 0")
	}
}

func TestPixelGeometryFrameLength(t *testing.T) {
	tests := []struct {
		name                               string
		rows, cols, samples, bitsAllocated uint16
		wantBytes                          int
	}{
		{"8-bit RGB", 4, 6, 3, 8, 4 * 6 * 3},
		{"16-bit mono", 512, 512, 1, 16, 512 * 512 * 2},
		{"1-bit packed", 512, 512, 1, 1, (512*512 + 7) / 8},
		{"1-bit packed odd", 3, 3, 1, 1, (9 + 7) / 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			geom := PixelGeometry{
				Rows: tc.rows, Columns: tc.cols,
				SamplesPerPixel: tc.samples, BitsAllocated: tc.bitsAllocated,
			}
			got := geom.FrameLength()
			if got != tc.wantBytes {
				t.Errorf("FrameLength() = %d, want %d", got, tc.wantBytes)
			}
		})
	}
}
