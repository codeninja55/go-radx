// AUTO-GENERATED - DO NOT EDIT
// Generated from DICOM PS3.6 Part 6 - Data Dictionary
// DICOM Standard Version: 2024b
//
// This file contains all SOP Class UID constants for convenient access.
// Total: 320 SOP Class UIDs

package uid

// SOP Class UIDs (including Meta SOP Classes)
// These are all the SOP classes defined in the DICOM standard.
var (
	// Verification SOP Class
	VerificationSOPClass = MustParse("1.2.840.10008.1.1")

	// Storage Commitment Push Model SOP Class
	StorageCommitmentPushModelSOPClass = MustParse("1.2.840.10008.1.20.1")

	// Storage Commitment Pull Model SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StorageCommitmentPullModelSOPClass = MustParse("1.2.840.10008.1.20.2")

	// Media Storage Directory Storage
	MediaStorageDirectoryStorage = MustParse("1.2.840.10008.1.3.10")

	// Procedural Event Logging SOP Class
	ProceduralEventLoggingSOPClass = MustParse("1.2.840.10008.1.40")

	// Substance Administration Logging SOP Class
	SubstanceAdministrationLoggingSOPClass = MustParse("1.2.840.10008.1.42")

	// Basic Study Content Notification SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	BasicStudyContentNotificationSOPClass = MustParse("1.2.840.10008.1.9")

	// Video Endoscopic Image Real-Time Communication
	VideoEndoscopicImageRealTimeCommunication = MustParse("1.2.840.10008.10.1")

	// Video Photographic Image Real-Time Communication
	VideoPhotographicImageRealTimeCommunication = MustParse("1.2.840.10008.10.2")

	// Audio Waveform Real-Time Communication
	AudioWaveformRealTimeCommunication = MustParse("1.2.840.10008.10.3")

	// Rendition Selection Document Real-Time Communication
	RenditionSelectionDocumentRealTimeCommunication = MustParse("1.2.840.10008.10.4")

	// Detached Patient Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedPatientManagementSOPClass = MustParse("1.2.840.10008.3.1.2.1.1")

	// Detached Patient Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedPatientManagementMetaSOPClass = MustParse("1.2.840.10008.3.1.2.1.4")

	// Detached Visit Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedVisitManagementSOPClass = MustParse("1.2.840.10008.3.1.2.2.1")

	// Detached Study Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedStudyManagementSOPClass = MustParse("1.2.840.10008.3.1.2.3.1")

	// Study Component Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StudyComponentManagementSOPClass = MustParse("1.2.840.10008.3.1.2.3.2")

	// Modality Performed Procedure Step SOP Class
	ModalityPerformedProcedureStepSOPClass = MustParse("1.2.840.10008.3.1.2.3.3")

	// Modality Performed Procedure Step Retrieve SOP Class
	ModalityPerformedProcedureStepRetrieveSOPClass = MustParse("1.2.840.10008.3.1.2.3.4")

	// Modality Performed Procedure Step Notification SOP Class
	ModalityPerformedProcedureStepNotificationSOPClass = MustParse("1.2.840.10008.3.1.2.3.5")

	// Detached Results Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedResultsManagementSOPClass = MustParse("1.2.840.10008.3.1.2.5.1")

	// Detached Results Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedResultsManagementMetaSOPClass = MustParse("1.2.840.10008.3.1.2.5.4")

	// Detached Study Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedStudyManagementMetaSOPClass = MustParse("1.2.840.10008.3.1.2.5.5")

	// Detached Interpretation Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetachedInterpretationManagementSOPClass = MustParse("1.2.840.10008.3.1.2.6.1")

	// Basic Film Session SOP Class
	BasicFilmSessionSOPClass = MustParse("1.2.840.10008.5.1.1.1")

	// Print Job SOP Class
	PrintJobSOPClass = MustParse("1.2.840.10008.5.1.1.14")

	// Basic Annotation Box SOP Class
	BasicAnnotationBoxSOPClass = MustParse("1.2.840.10008.5.1.1.15")

	// Printer SOP Class
	PrinterSOPClass = MustParse("1.2.840.10008.5.1.1.16")

	// Printer Configuration Retrieval SOP Class
	PrinterConfigurationRetrievalSOPClass = MustParse("1.2.840.10008.5.1.1.16.376")

	// Basic Color Print Management Meta SOP Class
	BasicColorPrintManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.1.18")

	// Referenced Color Print Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ReferencedColorPrintManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.1.18.1")

	// Basic Film Box SOP Class
	BasicFilmBoxSOPClass = MustParse("1.2.840.10008.5.1.1.2")

	// VOI LUT Box SOP Class
	VoiLutBoxSOPClass = MustParse("1.2.840.10008.5.1.1.22")

	// Presentation LUT SOP Class
	PresentationLutSOPClass = MustParse("1.2.840.10008.5.1.1.23")

	// Image Overlay Box SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ImageOverlayBoxSOPClass = MustParse("1.2.840.10008.5.1.1.24")

	// Basic Print Image Overlay Box SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	BasicPrintImageOverlayBoxSOPClass = MustParse("1.2.840.10008.5.1.1.24.1")

	// Print Queue Management SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PrintQueueManagementSOPClass = MustParse("1.2.840.10008.5.1.1.26")

	// Stored Print Storage SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StoredPrintStorageSOPClass = MustParse("1.2.840.10008.5.1.1.27")

	// Hardcopy Grayscale Image Storage SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	HardcopyGrayscaleImageStorageSOPClass = MustParse("1.2.840.10008.5.1.1.29")

	// Hardcopy Color Image Storage SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	HardcopyColorImageStorageSOPClass = MustParse("1.2.840.10008.5.1.1.30")

	// Pull Print Request SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PullPrintRequestSOPClass = MustParse("1.2.840.10008.5.1.1.31")

	// Pull Stored Print Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PullStoredPrintManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.1.32")

	// Media Creation Management SOP Class UID
	MediaCreationManagementSOPClassUID = MustParse("1.2.840.10008.5.1.1.33")

	// Basic Grayscale Image Box SOP Class
	BasicGrayscaleImageBoxSOPClass = MustParse("1.2.840.10008.5.1.1.4")

	// Basic Color Image Box SOP Class
	BasicColorImageBoxSOPClass = MustParse("1.2.840.10008.5.1.1.4.1")

	// Referenced Image Box SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ReferencedImageBoxSOPClass = MustParse("1.2.840.10008.5.1.1.4.2")

	// Display System SOP Class
	DisplaySystemSOPClass = MustParse("1.2.840.10008.5.1.1.40")

	// Basic Grayscale Print Management Meta SOP Class
	BasicGrayscalePrintManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.1.9")

	// Referenced Grayscale Print Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ReferencedGrayscalePrintManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.1.9.1")

	// Computed Radiography Image Storage
	ComputedRadiographyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.1")

	// Digital X-Ray Image Storage - For Presentation
	DigitalXRayImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.1.1")

	// Digital X-Ray Image Storage - For Processing
	DigitalXRayImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.1.1.1")

	// Digital Mammography X-Ray Image Storage - For Presentation
	DigitalMammographyXRayImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.1.2")

	// Digital Mammography X-Ray Image Storage - For Processing
	DigitalMammographyXRayImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.1.2.1")

	// Digital Intra-Oral X-Ray Image Storage - For Presentation
	DigitalIntraOralXRayImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.1.3")

	// Digital Intra-Oral X-Ray Image Storage - For Processing
	DigitalIntraOralXRayImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.1.3.1")

	// Standalone Modality LUT Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StandaloneModalityLutStorage = MustParse("1.2.840.10008.5.1.4.1.1.10")

	// Encapsulated PDF Storage
	EncapsulatedPDFStorage = MustParse("1.2.840.10008.5.1.4.1.1.104.1")

	// Encapsulated CDA Storage
	EncapsulatedCDAStorage = MustParse("1.2.840.10008.5.1.4.1.1.104.2")

	// Encapsulated STL Storage
	EncapsulatedSTLStorage = MustParse("1.2.840.10008.5.1.4.1.1.104.3")

	// Encapsulated OBJ Storage
	EncapsulatedOBJStorage = MustParse("1.2.840.10008.5.1.4.1.1.104.4")

	// Encapsulated MTL Storage
	EncapsulatedMTLStorage = MustParse("1.2.840.10008.5.1.4.1.1.104.5")

	// Standalone VOI LUT Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StandaloneVoiLutStorage = MustParse("1.2.840.10008.5.1.4.1.1.11")

	// Grayscale Softcopy Presentation State Storage
	GrayscaleSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.1")

	// Segmented Volume Rendering Volumetric Presentation State Storage
	SegmentedVolumeRenderingVolumetricPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.10")

	// Multiple Volume Rendering Volumetric Presentation State Storage
	MultipleVolumeRenderingVolumetricPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.11")

	// Variable Modality LUT Softcopy Presentation State Storage
	VariableModalityLutSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.12")

	// Color Softcopy Presentation State Storage
	ColorSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.2")

	// Pseudo-Color Softcopy Presentation State Storage
	PseudoColorSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.3")

	// Blending Softcopy Presentation State Storage
	BlendingSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.4")

	// XA/XRF Grayscale Softcopy Presentation State Storage
	XAXrfGrayscaleSoftcopyPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.5")

	// Grayscale Planar MPR Volumetric Presentation State Storage
	GrayscalePlanarMprVolumetricPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.6")

	// Compositing Planar MPR Volumetric Presentation State Storage
	CompositingPlanarMprVolumetricPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.7")

	// Advanced Blending Presentation State Storage
	AdvancedBlendingPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.8")

	// Volume Rendering Volumetric Presentation State Storage
	VolumeRenderingVolumetricPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.11.9")

	// X-Ray Angiographic Image Storage
	XRayAngiographicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.12.1")

	// Enhanced XA Image Storage
	EnhancedXAImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.12.1.1")

	// X-Ray Radiofluoroscopic Image Storage
	XRayRadiofluoroscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.12.2")

	// Enhanced XRF Image Storage
	EnhancedXrfImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.12.2.1")

	// X-Ray Angiographic Bi-Plane Image Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	XRayAngiographicBiPlaneImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.12.3")

	// Positron Emission Tomography Image Storage
	PositronEmissionTomographyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.128")

	// Legacy Converted Enhanced PET Image Storage
	LegacyConvertedEnhancedPETImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.128.1")

	// Standalone PET Curve Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StandalonePETCurveStorage = MustParse("1.2.840.10008.5.1.4.1.1.129")

	// X-Ray 3D Angiographic Image Storage
	XRay3dAngiographicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.13.1.1")

	// X-Ray 3D Craniofacial Image Storage
	XRay3dCraniofacialImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.13.1.2")

	// Breast Tomosynthesis Image Storage
	BreastTomosynthesisImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.13.1.3")

	// Breast Projection X-Ray Image Storage - For Presentation
	BreastProjectionXRayImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.13.1.4")

	// Breast Projection X-Ray Image Storage - For Processing
	BreastProjectionXRayImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.13.1.5")

	// Enhanced PET Image Storage
	EnhancedPETImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.130")

	// Basic Structured Display Storage
	BasicStructuredDisplayStorage = MustParse("1.2.840.10008.5.1.4.1.1.131")

	// Intravascular Optical Coherence Tomography Image Storage - For Presentation
	IntravascularOpticalCoherenceTomographyImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.14.1")

	// Intravascular Optical Coherence Tomography Image Storage - For Processing
	IntravascularOpticalCoherenceTomographyImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.14.2")

	// CT Image Storage
	CTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.2")

	// Enhanced CT Image Storage
	EnhancedCTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.2.1")

	// Legacy Converted Enhanced CT Image Storage
	LegacyConvertedEnhancedCTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.2.2")

	// Nuclear Medicine Image Storage
	NuclearMedicineImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.20")

	// CT Defined Procedure Protocol Storage
	CTDefinedProcedureProtocolStorage = MustParse("1.2.840.10008.5.1.4.1.1.200.1")

	// CT Performed Procedure Protocol Storage
	CTPerformedProcedureProtocolStorage = MustParse("1.2.840.10008.5.1.4.1.1.200.2")

	// Protocol Approval Storage
	ProtocolApprovalStorage = MustParse("1.2.840.10008.5.1.4.1.1.200.3")

	// Protocol Approval Information Model - FIND
	ProtocolApprovalInformationModelFind = MustParse("1.2.840.10008.5.1.4.1.1.200.4")

	// Protocol Approval Information Model - MOVE
	ProtocolApprovalInformationModelMove = MustParse("1.2.840.10008.5.1.4.1.1.200.5")

	// Protocol Approval Information Model - GET
	ProtocolApprovalInformationModelGet = MustParse("1.2.840.10008.5.1.4.1.1.200.6")

	// XA Defined Procedure Protocol Storage
	XADefinedProcedureProtocolStorage = MustParse("1.2.840.10008.5.1.4.1.1.200.7")

	// XA Performed Procedure Protocol Storage
	XAPerformedProcedureProtocolStorage = MustParse("1.2.840.10008.5.1.4.1.1.200.8")

	// Inventory Storage
	InventoryStorage = MustParse("1.2.840.10008.5.1.4.1.1.201.1")

	// Inventory - FIND
	InventoryFind = MustParse("1.2.840.10008.5.1.4.1.1.201.2")

	// Inventory - MOVE
	InventoryMove = MustParse("1.2.840.10008.5.1.4.1.1.201.3")

	// Inventory - GET
	InventoryGet = MustParse("1.2.840.10008.5.1.4.1.1.201.4")

	// Inventory Creation
	InventoryCreation = MustParse("1.2.840.10008.5.1.4.1.1.201.5")

	// Repository Query
	RepositoryQuery = MustParse("1.2.840.10008.5.1.4.1.1.201.6")

	// Ultrasound Multi-frame Image Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UltrasoundMultiFrameImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.3")

	// Ultrasound Multi-frame Image Storage
	UltrasoundMultiFrameImageStorage_1 = MustParse("1.2.840.10008.5.1.4.1.1.3.1")

	// Parametric Map Storage
	ParametricMapStorage = MustParse("1.2.840.10008.5.1.4.1.1.30")

	// MR Image Storage
	MRImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.4")

	// Enhanced MR Image Storage
	EnhancedMRImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.4.1")

	// MR Spectroscopy Storage
	MRSpectroscopyStorage = MustParse("1.2.840.10008.5.1.4.1.1.4.2")

	// Enhanced MR Color Image Storage
	EnhancedMRColorImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.4.3")

	// Legacy Converted Enhanced MR Image Storage
	LegacyConvertedEnhancedMRImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.4.4")

	// RT Image Storage
	RTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.1")

	// RT Physician Intent Storage
	RTPhysicianIntentStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.10")

	// RT Segment Annotation Storage
	RTSegmentAnnotationStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.11")

	// RT Radiation Set Storage
	RTRadiationSetStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.12")

	// C-Arm Photon-Electron Radiation Storage
	CArmPhotonElectronRadiationStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.13")

	// Tomotherapeutic Radiation Storage
	TomotherapeuticRadiationStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.14")

	// Robotic-Arm Radiation Storage
	RoboticArmRadiationStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.15")

	// RT Radiation Record Set Storage
	RTRadiationRecordSetStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.16")

	// RT Radiation Salvage Record Storage
	RTRadiationSalvageRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.17")

	// Tomotherapeutic Radiation Record Storage
	TomotherapeuticRadiationRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.18")

	// C-Arm Photon-Electron Radiation Record Storage
	CArmPhotonElectronRadiationRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.19")

	// RT Dose Storage
	RTDoseStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.2")

	// Robotic Radiation Record Storage
	RoboticRadiationRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.20")

	// RT Radiation Set Delivery Instruction Storage
	RTRadiationSetDeliveryInstructionStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.21")

	// RT Treatment Preparation Storage
	RTTreatmentPreparationStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.22")

	// Enhanced RT Image Storage
	EnhancedRTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.23")

	// Enhanced Continuous RT Image Storage
	EnhancedContinuousRTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.24")

	// RT Patient Position Acquisition Instruction Storage
	RTPatientPositionAcquisitionInstructionStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.25")

	// RT Structure Set Storage
	RTStructureSetStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.3")

	// RT Beams Treatment Record Storage
	RTBeamsTreatmentRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.4")

	// RT Plan Storage
	RTPlanStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.5")

	// RT Brachy Treatment Record Storage
	RTBrachyTreatmentRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.6")

	// RT Treatment Summary Record Storage
	RTTreatmentSummaryRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.7")

	// RT Ion Plan Storage
	RTIonPlanStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.8")

	// RT Ion Beams Treatment Record Storage
	RTIonBeamsTreatmentRecordStorage = MustParse("1.2.840.10008.5.1.4.1.1.481.9")

	// Nuclear Medicine Image Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	NuclearMedicineImageStorage_5 = MustParse("1.2.840.10008.5.1.4.1.1.5")

	// DICOS CT Image Storage
	DicosCTImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.501.1")

	// DICOS Digital X-Ray Image Storage - For Presentation
	DicosDigitalXRayImageStorageForPresentation = MustParse("1.2.840.10008.5.1.4.1.1.501.2.1")

	// DICOS Digital X-Ray Image Storage - For Processing
	DicosDigitalXRayImageStorageForProcessing = MustParse("1.2.840.10008.5.1.4.1.1.501.2.2")

	// DICOS Threat Detection Report Storage
	DicosThreatDetectionReportStorage = MustParse("1.2.840.10008.5.1.4.1.1.501.3")

	// DICOS 2D AIT Storage
	Dicos2dAitStorage = MustParse("1.2.840.10008.5.1.4.1.1.501.4")

	// DICOS 3D AIT Storage
	Dicos3dAitStorage = MustParse("1.2.840.10008.5.1.4.1.1.501.5")

	// DICOS Quadrupole Resonance (QR) Storage
	DicosQuadrupoleResonanceQRStorage = MustParse("1.2.840.10008.5.1.4.1.1.501.6")

	// Ultrasound Image Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UltrasoundImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.6")

	// Ultrasound Image Storage
	UltrasoundImageStorage_1 = MustParse("1.2.840.10008.5.1.4.1.1.6.1")

	// Enhanced US Volume Storage
	EnhancedUSVolumeStorage = MustParse("1.2.840.10008.5.1.4.1.1.6.2")

	// Photoacoustic Image Storage
	PhotoacousticImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.6.3")

	// Eddy Current Image Storage
	EddyCurrentImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.601.1")

	// Eddy Current Multi-frame Image Storage
	EddyCurrentMultiFrameImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.601.2")

	// Thermography Image Storage
	ThermographyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.601.3")

	// Thermography Multi-frame Image Storage
	ThermographyMultiFrameImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.601.4")

	// Ultrasound Waveform Storage
	UltrasoundWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.601.5")

	// Raw Data Storage
	RawDataStorage = MustParse("1.2.840.10008.5.1.4.1.1.66")

	// Spatial Registration Storage
	SpatialRegistrationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.1")

	// Spatial Fiducials Storage
	SpatialFiducialsStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.2")

	// Deformable Spatial Registration Storage
	DeformableSpatialRegistrationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.3")

	// Segmentation Storage
	SegmentationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.4")

	// Surface Segmentation Storage
	SurfaceSegmentationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.5")

	// Tractography Results Storage
	TractographyResultsStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.6")

	// Label Map Segmentation Storage
	LabelMapSegmentationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.7")

	// Height Map Segmentation Storage
	HeightMapSegmentationStorage = MustParse("1.2.840.10008.5.1.4.1.1.66.8")

	// Real World Value Mapping Storage
	RealWorldValueMappingStorage = MustParse("1.2.840.10008.5.1.4.1.1.67")

	// Surface Scan Mesh Storage
	SurfaceScanMeshStorage = MustParse("1.2.840.10008.5.1.4.1.1.68.1")

	// Surface Scan Point Cloud Storage
	SurfaceScanPointCloudStorage = MustParse("1.2.840.10008.5.1.4.1.1.68.2")

	// Secondary Capture Image Storage
	SecondaryCaptureImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.7")

	// Multi-frame Single Bit Secondary Capture Image Storage
	MultiFrameSingleBitSecondaryCaptureImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.7.1")

	// Multi-frame Grayscale Byte Secondary Capture Image Storage
	MultiFrameGrayscaleByteSecondaryCaptureImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.7.2")

	// Multi-frame Grayscale Word Secondary Capture Image Storage
	MultiFrameGrayscaleWordSecondaryCaptureImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.7.3")

	// Multi-frame True Color Secondary Capture Image Storage
	MultiFrameTrueColorSecondaryCaptureImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.7.4")

	// VL Image Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	VlImageStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.77.1")

	// VL Endoscopic Image Storage
	VlEndoscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.1")

	// Video Endoscopic Image Storage
	VideoEndoscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.1.1")

	// VL Microscopic Image Storage
	VlMicroscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.2")

	// Video Microscopic Image Storage
	VideoMicroscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.2.1")

	// VL Slide-Coordinates Microscopic Image Storage
	VlSlideCoordinatesMicroscopicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.3")

	// VL Photographic Image Storage
	VlPhotographicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.4")

	// Video Photographic Image Storage
	VideoPhotographicImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.4.1")

	// Ophthalmic Photography 8 Bit Image Storage
	OphthalmicPhotography8BitImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.1")

	// Ophthalmic Photography 16 Bit Image Storage
	OphthalmicPhotography16BitImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.2")

	// Stereometric Relationship Storage
	StereometricRelationshipStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.3")

	// Ophthalmic Tomography Image Storage
	OphthalmicTomographyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.4")

	// Wide Field Ophthalmic Photography Stereographic Projection Image Storage
	WideFieldOphthalmicPhotographyStereographicProjectionImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.5")

	// Wide Field Ophthalmic Photography 3D Coordinates Image Storage
	WideFieldOphthalmicPhotography3dCoordinatesImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.6")

	// Ophthalmic Optical Coherence Tomography En Face Image Storage
	OphthalmicOpticalCoherenceTomographyEnFaceImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.7")

	// Ophthalmic Optical Coherence Tomography B-scan Volume Analysis Storage
	OphthalmicOpticalCoherenceTomographyBScanVolumeAnalysisStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.5.8")

	// VL Whole Slide Microscopy Image Storage
	VlWholeSlideMicroscopyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.6")

	// Dermoscopic Photography Image Storage
	DermoscopicPhotographyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.7")

	// Confocal Microscopy Image Storage
	ConfocalMicroscopyImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.8")

	// Confocal Microscopy Tiled Pyramidal Image Storage
	ConfocalMicroscopyTiledPyramidalImageStorage = MustParse("1.2.840.10008.5.1.4.1.1.77.1.9")

	// VL Multi-frame Image Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	VlMultiFrameImageStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.77.2")

	// Lensometry Measurements Storage
	LensometryMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.1")

	// Autorefraction Measurements Storage
	AutorefractionMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.2")

	// Keratometry Measurements Storage
	KeratometryMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.3")

	// Subjective Refraction Measurements Storage
	SubjectiveRefractionMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.4")

	// Visual Acuity Measurements Storage
	VisualAcuityMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.5")

	// Spectacle Prescription Report Storage
	SpectaclePrescriptionReportStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.6")

	// Ophthalmic Axial Measurements Storage
	OphthalmicAxialMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.7")

	// Intraocular Lens Calculations Storage
	IntraocularLensCalculationsStorage = MustParse("1.2.840.10008.5.1.4.1.1.78.8")

	// Macular Grid Thickness and Volume Report Storage
	MacularGridThicknessAndVolumeReportStorage = MustParse("1.2.840.10008.5.1.4.1.1.79.1")

	// Standalone Overlay Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StandaloneOverlayStorage = MustParse("1.2.840.10008.5.1.4.1.1.8")

	// Ophthalmic Visual Field Static Perimetry Measurements Storage
	OphthalmicVisualFieldStaticPerimetryMeasurementsStorage = MustParse("1.2.840.10008.5.1.4.1.1.80.1")

	// Ophthalmic Thickness Map Storage
	OphthalmicThicknessMapStorage = MustParse("1.2.840.10008.5.1.4.1.1.81.1")

	// Corneal Topography Map Storage
	CornealTopographyMapStorage = MustParse("1.2.840.10008.5.1.4.1.1.82.1")

	// Text SR Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	TextSRStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.88.1")

	// Basic Text SR Storage
	BasicTextSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.11")

	// Audio SR Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	AudioSRStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.88.2")

	// Enhanced SR Storage
	EnhancedSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.22")

	// Detail SR Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	DetailSRStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.88.3")

	// Comprehensive SR Storage
	ComprehensiveSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.33")

	// Comprehensive 3D SR Storage
	Comprehensive3dSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.34")

	// Extensible SR Storage
	ExtensibleSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.35")

	// Comprehensive SR Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	ComprehensiveSRStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.88.4")

	// Procedure Log Storage
	ProcedureLogStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.40")

	// Mammography CAD SR Storage
	MammographyCadSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.50")

	// Key Object Selection Document Storage
	KeyObjectSelectionDocumentStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.59")

	// Chest CAD SR Storage
	ChestCadSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.65")

	// X-Ray Radiation Dose SR Storage
	XRayRadiationDoseSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.67")

	// Radiopharmaceutical Radiation Dose SR Storage
	RadiopharmaceuticalRadiationDoseSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.68")

	// Colon CAD SR Storage
	ColonCadSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.69")

	// Implantation Plan SR Storage
	ImplantationPlanSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.70")

	// Acquisition Context SR Storage
	AcquisitionContextSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.71")

	// Simplified Adult Echo SR Storage
	SimplifiedAdultEchoSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.72")

	// Patient Radiation Dose SR Storage
	PatientRadiationDoseSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.73")

	// Planned Imaging Agent Administration SR Storage
	PlannedImagingAgentAdministrationSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.74")

	// Performed Imaging Agent Administration SR Storage
	PerformedImagingAgentAdministrationSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.75")

	// Enhanced X-Ray Radiation Dose SR Storage
	EnhancedXRayRadiationDoseSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.76")

	// Waveform Annotation SR Storage
	WaveformAnnotationSRStorage = MustParse("1.2.840.10008.5.1.4.1.1.88.77")

	// Standalone Curve Storage (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	StandaloneCurveStorage = MustParse("1.2.840.10008.5.1.4.1.1.9")

	// Waveform Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	WaveformStorageTrial = MustParse("1.2.840.10008.5.1.4.1.1.9.1")

	// 12-lead ECG Waveform Storage
	UID12LeadEcgWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.1.1")

	// General ECG Waveform Storage
	GeneralEcgWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.1.2")

	// Ambulatory ECG Waveform Storage
	AmbulatoryEcgWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.1.3")

	// General 32-bit ECG Waveform Storage
	General32BitEcgWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.1.4")

	// Waveform Presentation State Storage
	WaveformPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.100.1")

	// Waveform Acquisition Presentation State Storage
	WaveformAcquisitionPresentationStateStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.100.2")

	// Hemodynamic Waveform Storage
	HemodynamicWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.2.1")

	// Cardiac Electrophysiology Waveform Storage
	CardiacElectrophysiologyWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.3.1")

	// Basic Voice Audio Waveform Storage
	BasicVoiceAudioWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.4.1")

	// General Audio Waveform Storage
	GeneralAudioWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.4.2")

	// Arterial Pulse Waveform Storage
	ArterialPulseWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.5.1")

	// Respiratory Waveform Storage
	RespiratoryWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.6.1")

	// Multi-channel Respiratory Waveform Storage
	MultiChannelRespiratoryWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.6.2")

	// Routine Scalp Electroencephalogram Waveform Storage
	RoutineScalpElectroencephalogramWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.7.1")

	// Electromyogram Waveform Storage
	ElectromyogramWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.7.2")

	// Electrooculogram Waveform Storage
	ElectrooculogramWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.7.3")

	// Sleep Electroencephalogram Waveform Storage
	SleepElectroencephalogramWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.7.4")

	// Body Position Waveform Storage
	BodyPositionWaveformStorage = MustParse("1.2.840.10008.5.1.4.1.1.9.8.1")

	// Content Assessment Results Storage
	ContentAssessmentResultsStorage = MustParse("1.2.840.10008.5.1.4.1.1.90.1")

	// Microscopy Bulk Simple Annotations Storage
	MicroscopyBulkSimpleAnnotationsStorage = MustParse("1.2.840.10008.5.1.4.1.1.91.1")

	// Patient Root Query/Retrieve Information Model - FIND
	PatientRootQueryRetrieveInformationModelFind = MustParse("1.2.840.10008.5.1.4.1.2.1.1")

	// Patient Root Query/Retrieve Information Model - MOVE
	PatientRootQueryRetrieveInformationModelMove = MustParse("1.2.840.10008.5.1.4.1.2.1.2")

	// Patient Root Query/Retrieve Information Model - GET
	PatientRootQueryRetrieveInformationModelGet = MustParse("1.2.840.10008.5.1.4.1.2.1.3")

	// Study Root Query/Retrieve Information Model - FIND
	StudyRootQueryRetrieveInformationModelFind = MustParse("1.2.840.10008.5.1.4.1.2.2.1")

	// Study Root Query/Retrieve Information Model - MOVE
	StudyRootQueryRetrieveInformationModelMove = MustParse("1.2.840.10008.5.1.4.1.2.2.2")

	// Study Root Query/Retrieve Information Model - GET
	StudyRootQueryRetrieveInformationModelGet = MustParse("1.2.840.10008.5.1.4.1.2.2.3")

	// Patient/Study Only Query/Retrieve Information Model - FIND (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PatientStudyOnlyQueryRetrieveInformationModelFind = MustParse("1.2.840.10008.5.1.4.1.2.3.1")

	// Patient/Study Only Query/Retrieve Information Model - MOVE (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PatientStudyOnlyQueryRetrieveInformationModelMove = MustParse("1.2.840.10008.5.1.4.1.2.3.2")

	// Patient/Study Only Query/Retrieve Information Model - GET (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	PatientStudyOnlyQueryRetrieveInformationModelGet = MustParse("1.2.840.10008.5.1.4.1.2.3.3")

	// Composite Instance Root Retrieve - MOVE
	CompositeInstanceRootRetrieveMove = MustParse("1.2.840.10008.5.1.4.1.2.4.2")

	// Composite Instance Root Retrieve - GET
	CompositeInstanceRootRetrieveGet = MustParse("1.2.840.10008.5.1.4.1.2.4.3")

	// Composite Instance Retrieve Without Bulk Data - GET
	CompositeInstanceRetrieveWithoutBulkDataGet = MustParse("1.2.840.10008.5.1.4.1.2.5.3")

	// Defined Procedure Protocol Information Model - FIND
	DefinedProcedureProtocolInformationModelFind = MustParse("1.2.840.10008.5.1.4.20.1")

	// Defined Procedure Protocol Information Model - MOVE
	DefinedProcedureProtocolInformationModelMove = MustParse("1.2.840.10008.5.1.4.20.2")

	// Defined Procedure Protocol Information Model - GET
	DefinedProcedureProtocolInformationModelGet = MustParse("1.2.840.10008.5.1.4.20.3")

	// Modality Worklist Information Model - FIND
	ModalityWorklistInformationModelFind = MustParse("1.2.840.10008.5.1.4.31")

	// General Purpose Worklist Management Meta SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	GeneralPurposeWorklistManagementMetaSOPClass = MustParse("1.2.840.10008.5.1.4.32")

	// General Purpose Worklist Information Model - FIND (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	GeneralPurposeWorklistInformationModelFind = MustParse("1.2.840.10008.5.1.4.32.1")

	// General Purpose Scheduled Procedure Step SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	GeneralPurposeScheduledProcedureStepSOPClass = MustParse("1.2.840.10008.5.1.4.32.2")

	// General Purpose Performed Procedure Step SOP Class (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	GeneralPurposePerformedProcedureStepSOPClass = MustParse("1.2.840.10008.5.1.4.32.3")

	// Instance Availability Notification SOP Class
	InstanceAvailabilityNotificationSOPClass = MustParse("1.2.840.10008.5.1.4.33")

	// RT Beams Delivery Instruction Storage - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	RTBeamsDeliveryInstructionStorageTrial = MustParse("1.2.840.10008.5.1.4.34.1")

	// RT Brachy Application Setup Delivery Instruction Storage
	RTBrachyApplicationSetupDeliveryInstructionStorage = MustParse("1.2.840.10008.5.1.4.34.10")

	// RT Conventional Machine Verification - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	RTConventionalMachineVerificationTrial = MustParse("1.2.840.10008.5.1.4.34.2")

	// RT Ion Machine Verification - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	RTIonMachineVerificationTrial = MustParse("1.2.840.10008.5.1.4.34.3")

	// Unified Procedure Step - Push SOP Class - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UnifiedProcedureStepPushSOPClassTrial = MustParse("1.2.840.10008.5.1.4.34.4.1")

	// Unified Procedure Step - Watch SOP Class - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UnifiedProcedureStepWatchSOPClassTrial = MustParse("1.2.840.10008.5.1.4.34.4.2")

	// Unified Procedure Step - Pull SOP Class - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UnifiedProcedureStepPullSOPClassTrial = MustParse("1.2.840.10008.5.1.4.34.4.3")

	// Unified Procedure Step - Event SOP Class - Trial (RETIRED)
	//
	// Deprecated: This UID has been retired from the DICOM standard.
	UnifiedProcedureStepEventSOPClassTrial = MustParse("1.2.840.10008.5.1.4.34.4.4")

	// Unified Procedure Step - Push SOP Class
	UnifiedProcedureStepPushSOPClass = MustParse("1.2.840.10008.5.1.4.34.6.1")

	// Unified Procedure Step - Watch SOP Class
	UnifiedProcedureStepWatchSOPClass = MustParse("1.2.840.10008.5.1.4.34.6.2")

	// Unified Procedure Step - Pull SOP Class
	UnifiedProcedureStepPullSOPClass = MustParse("1.2.840.10008.5.1.4.34.6.3")

	// Unified Procedure Step - Event SOP Class
	UnifiedProcedureStepEventSOPClass = MustParse("1.2.840.10008.5.1.4.34.6.4")

	// Unified Procedure Step - Query SOP Class
	UnifiedProcedureStepQuerySOPClass = MustParse("1.2.840.10008.5.1.4.34.6.5")

	// RT Beams Delivery Instruction Storage
	RTBeamsDeliveryInstructionStorage = MustParse("1.2.840.10008.5.1.4.34.7")

	// RT Conventional Machine Verification
	RTConventionalMachineVerification = MustParse("1.2.840.10008.5.1.4.34.8")

	// RT Ion Machine Verification
	RTIonMachineVerification = MustParse("1.2.840.10008.5.1.4.34.9")

	// General Relevant Patient Information Query
	GeneralRelevantPatientInformationQuery = MustParse("1.2.840.10008.5.1.4.37.1")

	// Breast Imaging Relevant Patient Information Query
	BreastImagingRelevantPatientInformationQuery = MustParse("1.2.840.10008.5.1.4.37.2")

	// Cardiac Relevant Patient Information Query
	CardiacRelevantPatientInformationQuery = MustParse("1.2.840.10008.5.1.4.37.3")

	// Hanging Protocol Storage
	HangingProtocolStorage = MustParse("1.2.840.10008.5.1.4.38.1")

	// Hanging Protocol Information Model - FIND
	HangingProtocolInformationModelFind = MustParse("1.2.840.10008.5.1.4.38.2")

	// Hanging Protocol Information Model - MOVE
	HangingProtocolInformationModelMove = MustParse("1.2.840.10008.5.1.4.38.3")

	// Hanging Protocol Information Model - GET
	HangingProtocolInformationModelGet = MustParse("1.2.840.10008.5.1.4.38.4")

	// Color Palette Storage
	ColorPaletteStorage = MustParse("1.2.840.10008.5.1.4.39.1")

	// Color Palette Query/Retrieve Information Model - FIND
	ColorPaletteQueryRetrieveInformationModelFind = MustParse("1.2.840.10008.5.1.4.39.2")

	// Color Palette Query/Retrieve Information Model - MOVE
	ColorPaletteQueryRetrieveInformationModelMove = MustParse("1.2.840.10008.5.1.4.39.3")

	// Color Palette Query/Retrieve Information Model - GET
	ColorPaletteQueryRetrieveInformationModelGet = MustParse("1.2.840.10008.5.1.4.39.4")

	// Product Characteristics Query SOP Class
	ProductCharacteristicsQuerySOPClass = MustParse("1.2.840.10008.5.1.4.41")

	// Substance Approval Query SOP Class
	SubstanceApprovalQuerySOPClass = MustParse("1.2.840.10008.5.1.4.42")

	// Generic Implant Template Storage
	GenericImplantTemplateStorage = MustParse("1.2.840.10008.5.1.4.43.1")

	// Generic Implant Template Information Model - FIND
	GenericImplantTemplateInformationModelFind = MustParse("1.2.840.10008.5.1.4.43.2")

	// Generic Implant Template Information Model - MOVE
	GenericImplantTemplateInformationModelMove = MustParse("1.2.840.10008.5.1.4.43.3")

	// Generic Implant Template Information Model - GET
	GenericImplantTemplateInformationModelGet = MustParse("1.2.840.10008.5.1.4.43.4")

	// Implant Assembly Template Storage
	ImplantAssemblyTemplateStorage = MustParse("1.2.840.10008.5.1.4.44.1")

	// Implant Assembly Template Information Model - FIND
	ImplantAssemblyTemplateInformationModelFind = MustParse("1.2.840.10008.5.1.4.44.2")

	// Implant Assembly Template Information Model - MOVE
	ImplantAssemblyTemplateInformationModelMove = MustParse("1.2.840.10008.5.1.4.44.3")

	// Implant Assembly Template Information Model - GET
	ImplantAssemblyTemplateInformationModelGet = MustParse("1.2.840.10008.5.1.4.44.4")

	// Implant Template Group Storage
	ImplantTemplateGroupStorage = MustParse("1.2.840.10008.5.1.4.45.1")

	// Implant Template Group Information Model - FIND
	ImplantTemplateGroupInformationModelFind = MustParse("1.2.840.10008.5.1.4.45.2")

	// Implant Template Group Information Model - MOVE
	ImplantTemplateGroupInformationModelMove = MustParse("1.2.840.10008.5.1.4.45.3")

	// Implant Template Group Information Model - GET
	ImplantTemplateGroupInformationModelGet = MustParse("1.2.840.10008.5.1.4.45.4")
)
