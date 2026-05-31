package dicom

import "fmt"

// UID is a dotted-numeric ISO OID of at most 64 characters (VR UI). It identifies
// SOP Classes and Instances, Studies, Series, and transfer syntaxes.
type UID string

// ParseUID validates s per PS3.5 sec 9.1 and returns the trimmed UID. It rejects
// empty components ("1..2"), leading-zero multi-digit components ("1.02"), leading
// and trailing dots, non-numeric characters, and lengths over 64 characters.
func ParseUID(s string) (UID, error) {
	u := UID(s)
	if err := u.Validate(); err != nil {
		return "", err
	}
	return u, nil
}

// Validate reports a typed ValueError when u is not a conformant UID.
func (u UID) Validate() error {
	s := string(u)
	if s == "" {
		return &ValueError{VR: VRUI, Msg: "UID is empty"}
	}
	if len(s) > 64 {
		return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID exceeds 64 characters (%d)", len(s))}
	}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			comp := s[start:i]
			if comp == "" {
				return &ValueError{VR: VRUI, Msg: "UID has an empty component"}
			}
			if len(comp) > 1 && comp[0] == '0' {
				return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID component %q has a leading zero", comp)}
			}
			for j := 0; j < len(comp); j++ {
				if comp[j] < '0' || comp[j] > '9' {
					return &ValueError{VR: VRUI, Msg: fmt.Sprintf("UID component %q is not numeric", comp)}
				}
			}
			start = i + 1
		}
	}
	return nil
}

// IsValid reports whether u passes Validate.
func (u UID) IsValid() bool { return u.Validate() == nil }

// String returns the raw UID text.
func (u UID) String() string { return string(u) }

// Name returns the registered name for a known UID or the UID itself if unregistered.
func (u UID) Name() string {
	if name, ok := uidNames[u]; ok {
		return name
	}
	return string(u)
}

// TODO: superseded by the generated uid_values.go dictionary (Task 1.4).
var uidNames = map[UID]string{
	"1.2.840.10008.1.2.1": "Explicit VR Little Endian",
}
