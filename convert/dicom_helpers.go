package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// combineDateTime builds a FHIR dateTime string from a DICOM DA (date) and an
// optional TM (time) at the dataset's two tags. It preserves the source precision
// rather than fabricating components the source omitted: a date-only source
// yields "YYYY-MM-DD". A date-with-time source yields "YYYY-MM-DDThh:mm:ss" with a
// timezone — FHIR dateTime requires a timezone whenever a time is present — taking
// the offset from TimezoneOffsetFromUTC (0008,0201) when it is present. Absent any
// offset, the converter does NOT fabricate one (the source never stated its zone):
// it falls back to the date-only form and records the dropped time precision on
// the report under timePath. An absent or partial date yields "".
func combineDateTime(ds *dicom.DataSet, dateTag, timeTag dicom.Tag, report *Report, timePath string) string {
	dateStr, ok := ds.GetString(dateTag)
	if !ok || dateStr == "" {
		return ""
	}
	da, err := dicom.ParseDA(dateStr)
	if err != nil || da.Precision() != dicom.DatePrecisionDay {
		// A partial (lenient) date has no full FHIR date form; do not fabricate
		// the missing month/day.
		return ""
	}
	out := isoDate(da)

	timeStr, hasTime := ds.GetString(timeTag)
	if !hasTime || timeStr == "" {
		return out
	}
	tm, terr := dicom.ParseTM(timeStr)
	if terr != nil {
		return out
	}
	iso := isoTime(tm)
	if iso == "" {
		return out
	}

	offset, hasOffset := fhirTimezoneOffset(ds)
	if !hasOffset {
		// FHIR dateTime forbids a timezone-less time; the source supplied no
		// offset, so emit date-only rather than fabricate a zone, and record the
		// dropped time precision so the loss is honest.
		report.dropped(dicomTagSource(timeTag),
			"time present but no TimezoneOffsetFromUTC (0008,0201); FHIR dateTime requires a timezone, so the time was dropped")
		return out
	}
	return out + "T" + iso + offset
}

// fhirTimezoneOffset reads TimezoneOffsetFromUTC (0008,0201), a DICOM "&ZZXX"
// signed offset such as "-0500", and renders it as a FHIR timezone suffix ("Z"
// for +0000, otherwise "+hh:mm"/"-hh:mm"). ok is false when the attribute is
// absent or malformed.
func fhirTimezoneOffset(ds *dicom.DataSet) (string, bool) {
	raw, ok := ds.GetString(dicom.TagTimezoneOffsetFromUTC)
	if !ok || len(raw) != 5 {
		return "", false
	}
	sign := raw[0]
	if sign != '+' && sign != '-' {
		return "", false
	}
	hh, mm := raw[1:3], raw[3:5]
	if !validOffset(hh, mm) {
		return "", false
	}
	if sign == '+' && hh == "00" && mm == "00" {
		return "Z", true
	}
	return string(sign) + hh + ":" + mm, true
}

// validOffset reports whether the two-digit hour and minute form a valid timezone
// offset (hours 00-14, minutes 00-59). A digit-only but out-of-range offset such
// as "+2460" must be rejected, not emitted as an invalid FHIR suffix.
func validOffset(hh, mm string) bool {
	if !isDigits(hh) || !isDigits(mm) || len(hh) != 2 || len(mm) != 2 {
		return false
	}
	h := (int(hh[0]-'0'))*10 + int(hh[1]-'0')
	m := (int(mm[0]-'0'))*10 + int(mm[1]-'0')
	return h <= 14 && m <= 59
}

// isDigits reports whether every byte of s is an ASCII digit.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// dicomTagSource renders a DICOM tag as the report Source locus "DICOM (gggg,eeee)
// Keyword", naming the concept without any patient value.
func dicomTagSource(t dicom.Tag) string {
	src := "DICOM " + t.String()
	if info, ok := dicom.Lookup(t); ok && info.Keyword != "" {
		src += " " + info.Keyword
	}
	return src
}

// fractionOf returns the fractional-second suffix (including the leading dot) of a
// DICOM TM/DT lexical form, or "" when none is present. Only the run of digits after
// the dot is taken, so a DT's trailing &ZZXX UTC offset is not folded into the
// fraction. The fraction is preserved verbatim so a clinically relevant sub-second
// timestamp is not silently truncated to whole seconds.
func fractionOf(lexical string) string {
	for i := 0; i < len(lexical); i++ {
		if lexical[i] == '.' {
			end := i + 1
			for end < len(lexical) && lexical[end] >= '0' && lexical[end] <= '9' {
				end++
			}
			// A bare dot with no following digits carries no fraction; drop it.
			if end == i+1 {
				return ""
			}
			return lexical[i:end]
		}
	}
	return ""
}

// isoDate renders a full-day DA as YYYY-MM-DD.
func isoDate(da dicom.DA) string {
	return pad4(da.Year()) + "-" + pad2(da.Month()) + "-" + pad2(da.Day())
}

// isoTime renders a TM as hh:mm:ss to the precision the source carried, or ""
// when the time is empty. FHIR dateTime requires at least seconds when a time is
// present, so a less precise TM is widened to seconds by zero-filling the lower
// components — but only when at least the hour is present.
func isoTime(tm dicom.TM) string {
	t, ok := tm.Time()
	if !ok {
		return ""
	}
	return pad2(t.Hour()) + ":" + pad2(t.Minute()) + ":" + pad2(t.Second()) + fractionOf(tm.String())
}

// pad2 renders a non-negative integer as a zero-padded two-digit string.
func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// pad4 renders a non-negative integer as a zero-padded four-digit string.
func pad4(n int) string {
	switch {
	case n < 10:
		return "000" + itoa(n)
	case n < 100:
		return "00" + itoa(n)
	case n < 1000:
		return "0" + itoa(n)
	default:
		return itoa(n)
	}
}

// itoa renders a non-negative integer without importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// dicomPatientSubjectR5 resolves the Patient subject Reference for a DICOM-sourced
// converter. WithSubjectR5 takes precedence. Otherwise the DICOM PatientID
// (0010,0020) and IssuerOfPatientID (0010,0021) are carried as a logical
// Reference.identifier — never a fabricated Reference.reference URL (the identity
// rule). When the dataset carries no PatientID, subject is left unset (nil) and a
// Defaulted entry records the absence.
func dicomPatientSubjectR5(cfg config, ds *dicom.DataSet, report *Report, targetPath string) *r5.Reference {
	if cfg.subjectR5 != nil {
		ref := *cfg.subjectR5
		return &ref
	}

	patientID, ok := ds.GetString(dicom.TagPatientID)
	if !ok || patientID == "" {
		report.defaulted(targetPath, "",
			"dataset carries no PatientID (0010,0020) and no WithSubjectR5 was supplied; subject left unset")
		return nil
	}

	value := patientID
	id := r5.Identifier{Value: &value}
	if issuer, has := ds.GetString(dicom.TagIssuerOfPatientID); has && issuer != "" {
		system := issuer
		id.System = &system
	}
	refType := patientReferenceType
	return &r5.Reference{Type: &refType, Identifier: &id}
}
