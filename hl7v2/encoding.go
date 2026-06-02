package hl7v2

// EncodingCharacters carries the five HL7 delimiters in force for a message.
// Field is MSH-1; the remaining four are the characters of MSH-2 in the spec
// order component/repetition/escape/subcomponent. The set is derived per
// message and never read from a package global, so a sender using non-standard
// delimiters round-trips correctly (the non-standard-sender footgun, PRD §9.4).
type EncodingCharacters struct {
	Field        byte // MSH-1, default '|'
	Component    byte // MSH-2[0], default '^'
	Repetition   byte // MSH-2[1], default '~'
	Escape       byte // MSH-2[2], default '\'
	Subcomponent byte // MSH-2[3], default '&'
}

// The standard HL7 v2 delimiters (Chapter 2 §2.5). A header of "MSH|^~\&" is
// complete; shorter MSH-2 values fall back to these per position.
const (
	defaultFieldSep        = '|'
	defaultComponentSep    = '^'
	defaultRepetitionSep   = '~'
	defaultEscapeChar      = '\\'
	defaultSubcomponentSep = '&'
)

// DefaultEncoding returns the standard delimiters: | ^ ~ \ &
func DefaultEncoding() EncodingCharacters {
	return EncodingCharacters{
		Field:        defaultFieldSep,
		Component:    defaultComponentSep,
		Repetition:   defaultRepetitionSep,
		Escape:       defaultEscapeChar,
		Subcomponent: defaultSubcomponentSep,
	}
}

// DeriveEncoding reads the delimiters from a raw message header. It reads MSH-1
// as the byte at offset 3 and MSH-2 as the bytes up to the next MSH-1. Missing
// trailing characters fall back to the standard defaults (a header of "MSH|^~\&"
// is complete; "MSH|^" fills repetition/escape/subcomponent from the defaults).
// The segment terminator is always '\r'.
func DeriveEncoding(header []byte) (EncodingCharacters, error) {
	if len(header) < 4 || header[0] != 'M' || header[1] != 'S' || header[2] != 'H' {
		return EncodingCharacters{}, &ParseError{Offset: 0, Reason: "header does not begin with MSH"}
	}

	enc := DefaultEncoding()
	enc.Field = header[3]

	// MSH-2 runs from offset 4 to the next field separator (or end of header).
	end := indexByte(header[4:], enc.Field)
	var msh2 []byte
	if end < 0 {
		msh2 = header[4:]
	} else {
		msh2 = header[4 : 4+end]
	}

	if len(msh2) > 0 {
		enc.Component = msh2[0]
	}
	if len(msh2) > 1 {
		enc.Repetition = msh2[1]
	}
	if len(msh2) > 2 {
		enc.Escape = msh2[2]
	}
	if len(msh2) > 3 {
		enc.Subcomponent = msh2[3]
	}

	return enc, nil
}

// indexByte returns the index of the first occurrence of c in b, or -1.
func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}
