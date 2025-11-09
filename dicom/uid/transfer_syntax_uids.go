// AUTO-GENERATED - DO NOT EDIT
// Generated from DICOM PS3.6 Part 6 - Data Dictionary
// DICOM Standard Version: 2024b
//
// This file contains all Transfer Syntax UID constants for convenient access.
// Total: 63 Transfer Syntax UIDs

package uid

// Transfer Syntax UIDs
// These are all the transfer syntaxes defined in the DICOM standard.
var (
	// Implicit VR Little Endian
	ImplicitVRLittleEndian = MustParse("1.2.840.10008.1.2")

	// Explicit VR Little Endian
	ExplicitVRLittleEndian = MustParse("1.2.840.10008.1.2.1")

	// Encapsulated Uncompressed Explicit VR Little Endian
	EncapsulatedUncompressedExplicitVRLittleEndian = MustParse("1.2.840.10008.1.2.1.98")

	// Deflated Explicit VR Little Endian
	DeflatedExplicitVRLittleEndian = MustParse("1.2.840.10008.1.2.1.99")

	// Explicit VR Big Endian (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ExplicitVRBigEndian = MustParse("1.2.840.10008.1.2.2")

	// MPEG2 Main Profile / Main Level
	Mpeg2MainProfileMainLevel = MustParse("1.2.840.10008.1.2.4.100")

	// Fragmentable MPEG2 Main Profile / Main Level
	FragmentableMpeg2MainProfileMainLevel = MustParse("1.2.840.10008.1.2.4.100.1")

	// MPEG2 Main Profile / High Level
	Mpeg2MainProfileHighLevel = MustParse("1.2.840.10008.1.2.4.101")

	// Fragmentable MPEG2 Main Profile / High Level
	FragmentableMpeg2MainProfileHighLevel = MustParse("1.2.840.10008.1.2.4.101.1")

	// MPEG-4 AVC/H.264 High Profile / Level 4.1
	MPEG4AvcH264HighProfileLevel41 = MustParse("1.2.840.10008.1.2.4.102")

	// Fragmentable MPEG-4 AVC/H.264 High Profile / Level 4.1
	FragmentableMPEG4AvcH264HighProfileLevel41 = MustParse("1.2.840.10008.1.2.4.102.1")

	// MPEG-4 AVC/H.264 BD-compatible High Profile / Level 4.1
	MPEG4AvcH264BdCompatibleHighProfileLevel41 = MustParse("1.2.840.10008.1.2.4.103")

	// Fragmentable MPEG-4 AVC/H.264 BD-compatible High Profile / Level 4.1
	FragmentableMPEG4AvcH264BdCompatibleHighProfileLevel41 = MustParse("1.2.840.10008.1.2.4.103.1")

	// MPEG-4 AVC/H.264 High Profile / Level 4.2 For 2D Video
	MPEG4AvcH264HighProfileLevel42For2dVideo = MustParse("1.2.840.10008.1.2.4.104")

	// Fragmentable MPEG-4 AVC/H.264 High Profile / Level 4.2 For 2D Video
	FragmentableMPEG4AvcH264HighProfileLevel42For2dVideo = MustParse("1.2.840.10008.1.2.4.104.1")

	// MPEG-4 AVC/H.264 High Profile / Level 4.2 For 3D Video
	MPEG4AvcH264HighProfileLevel42For3dVideo = MustParse("1.2.840.10008.1.2.4.105")

	// Fragmentable MPEG-4 AVC/H.264 High Profile / Level 4.2 For 3D Video
	FragmentableMPEG4AvcH264HighProfileLevel42For3dVideo = MustParse("1.2.840.10008.1.2.4.105.1")

	// MPEG-4 AVC/H.264 Stereo High Profile / Level 4.2
	MPEG4AvcH264StereoHighProfileLevel42 = MustParse("1.2.840.10008.1.2.4.106")

	// Fragmentable MPEG-4 AVC/H.264 Stereo High Profile / Level 4.2
	FragmentableMPEG4AvcH264StereoHighProfileLevel42 = MustParse("1.2.840.10008.1.2.4.106.1")

	// HEVC/H.265 Main Profile / Level 5.1
	HevcH265MainProfileLevel51 = MustParse("1.2.840.10008.1.2.4.107")

	// HEVC/H.265 Main 10 Profile / Level 5.1
	HevcH265Main10ProfileLevel51 = MustParse("1.2.840.10008.1.2.4.108")

	// JPEG XL Lossless
	JPEGXlLossless = MustParse("1.2.840.10008.1.2.4.110")

	// JPEG XL JPEG Recompression
	JPEGXlJPEGRecompression = MustParse("1.2.840.10008.1.2.4.111")

	// JPEG XL
	JPEGXl = MustParse("1.2.840.10008.1.2.4.112")

	// High-Throughput JPEG 2000 Image Compression (Lossless Only)
	HighThroughputJPEG2000ImageCompressionLosslessOnly = MustParse("1.2.840.10008.1.2.4.201")

	// High-Throughput JPEG 2000 with RPCL Options Image Compression (Lossless Only)
	HighThroughputJPEG2000WithRpclOptionsImageCompressionLosslessOnly = MustParse("1.2.840.10008.1.2.4.202")

	// High-Throughput JPEG 2000 Image Compression
	HighThroughputJPEG2000ImageCompression = MustParse("1.2.840.10008.1.2.4.203")

	// JPIP HTJ2K Referenced
	JpipHtj2kReferenced = MustParse("1.2.840.10008.1.2.4.204")

	// JPIP HTJ2K Referenced Deflate
	JpipHtj2kReferencedDeflate = MustParse("1.2.840.10008.1.2.4.205")

	// JPEG Baseline (Process 1)
	JPEGBaselineProcess1 = MustParse("1.2.840.10008.1.2.4.50")

	// JPEG Extended (Process 2 and 4)
	JPEGExtendedProcess2And4 = MustParse("1.2.840.10008.1.2.4.51")

	// JPEG Extended (Process 3 and 5) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGExtendedProcess3And5 = MustParse("1.2.840.10008.1.2.4.52")

	// JPEG Spectral Selection, Non-Hierarchical (Process 6 and 8) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGSpectralSelectionNonHierarchicalProcess6And8 = MustParse("1.2.840.10008.1.2.4.53")

	// JPEG Spectral Selection, Non-Hierarchical (Process 7 and 9) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGSpectralSelectionNonHierarchicalProcess7And9 = MustParse("1.2.840.10008.1.2.4.54")

	// JPEG Full Progression, Non-Hierarchical (Process 10 and 12) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGFullProgressionNonHierarchicalProcess10And12 = MustParse("1.2.840.10008.1.2.4.55")

	// JPEG Full Progression, Non-Hierarchical (Process 11 and 13) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGFullProgressionNonHierarchicalProcess11And13 = MustParse("1.2.840.10008.1.2.4.56")

	// JPEG Lossless, Non-Hierarchical (Process 14)
	JPEGLosslessNonHierarchicalProcess14 = MustParse("1.2.840.10008.1.2.4.57")

	// JPEG Lossless, Non-Hierarchical (Process 15) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGLosslessNonHierarchicalProcess15 = MustParse("1.2.840.10008.1.2.4.58")

	// JPEG Extended, Hierarchical (Process 16 and 18) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGExtendedHierarchicalProcess16And18 = MustParse("1.2.840.10008.1.2.4.59")

	// JPEG Extended, Hierarchical (Process 17 and 19) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGExtendedHierarchicalProcess17And19 = MustParse("1.2.840.10008.1.2.4.60")

	// JPEG Spectral Selection, Hierarchical (Process 20 and 22) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGSpectralSelectionHierarchicalProcess20And22 = MustParse("1.2.840.10008.1.2.4.61")

	// JPEG Spectral Selection, Hierarchical (Process 21 and 23) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGSpectralSelectionHierarchicalProcess21And23 = MustParse("1.2.840.10008.1.2.4.62")

	// JPEG Full Progression, Hierarchical (Process 24 and 26) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGFullProgressionHierarchicalProcess24And26 = MustParse("1.2.840.10008.1.2.4.63")

	// JPEG Full Progression, Hierarchical (Process 25 and 27) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGFullProgressionHierarchicalProcess25And27 = MustParse("1.2.840.10008.1.2.4.64")

	// JPEG Lossless, Hierarchical (Process 28) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGLosslessHierarchicalProcess28 = MustParse("1.2.840.10008.1.2.4.65")

	// JPEG Lossless, Hierarchical (Process 29) (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	JPEGLosslessHierarchicalProcess29 = MustParse("1.2.840.10008.1.2.4.66")

	// JPEG Lossless, Non-Hierarchical, First-Order Prediction (Process 14 [Selection Value 1])
	JPEGLosslessNonHierarchicalFirstOrderPredictionProcess14SelectionValue1 = MustParse("1.2.840.10008.1.2.4.70")

	// JPEG-LS Lossless Image Compression
	JPEGLsLosslessImageCompression = MustParse("1.2.840.10008.1.2.4.80")

	// JPEG-LS Lossy (Near-Lossless) Image Compression
	JPEGLsLossyNearLosslessImageCompression = MustParse("1.2.840.10008.1.2.4.81")

	// JPEG 2000 Image Compression (Lossless Only)
	JPEG2000ImageCompressionLosslessOnly = MustParse("1.2.840.10008.1.2.4.90")

	// JPEG 2000 Image Compression
	JPEG2000ImageCompression = MustParse("1.2.840.10008.1.2.4.91")

	// JPEG 2000 Part 2 Multi-component Image Compression (Lossless Only)
	JPEG2000Part2MultiComponentImageCompressionLosslessOnly = MustParse("1.2.840.10008.1.2.4.92")

	// JPEG 2000 Part 2 Multi-component Image Compression
	JPEG2000Part2MultiComponentImageCompression = MustParse("1.2.840.10008.1.2.4.93")

	// JPIP Referenced
	JpipReferenced = MustParse("1.2.840.10008.1.2.4.94")

	// JPIP Referenced Deflate
	JpipReferencedDeflate = MustParse("1.2.840.10008.1.2.4.95")

	// RLE Lossless
	RLELossless = MustParse("1.2.840.10008.1.2.5")

	// RFC 2557 MIME encapsulation (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	Rfc2557MimeEncapsulation = MustParse("1.2.840.10008.1.2.6.1")

	// XML Encoding (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	XMLEncoding = MustParse("1.2.840.10008.1.2.6.2")

	// SMPTE ST 2110-20 Uncompressed Progressive Active Video
	SMPTESt211020UncompressedProgressiveActiveVideo = MustParse("1.2.840.10008.1.2.7.1")

	// SMPTE ST 2110-20 Uncompressed Interlaced Active Video
	SMPTESt211020UncompressedInterlacedActiveVideo = MustParse("1.2.840.10008.1.2.7.2")

	// SMPTE ST 2110-30 PCM Digital Audio
	SMPTESt211030PcmDigitalAudio = MustParse("1.2.840.10008.1.2.7.3")

	// Deflated Image Frame Compression
	DeflatedImageFrameCompression = MustParse("1.2.840.10008.1.2.8.1")

	// Papyrus 3 Implicit VR Little Endian (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	Papyrus3ImplicitVRLittleEndian = MustParse("1.2.840.10008.1.20")
)
