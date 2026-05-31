package dicom

import (
	"fmt"
	"strings"
)

// maxPNComponent is the per-component PS3.5 character cap for PN.
const maxPNComponent = 64

// NameComponents holds the five ^-delimited components of one PersonName group.
type NameComponents struct {
	FamilyName string
	GivenName  string
	MiddleName string
	Prefix     string
	Suffix     string
}

// PersonName is VR PN: up to three =-delimited component groups (alphabetic,
// ideographic, phonetic), each holding up to five ^-delimited components.
type PersonName struct {
	Alphabetic  NameComponents
	Ideographic NameComponents // empty if absent
	Phonetic    NameComponents // empty if absent
}

// ParsePersonName splits s on "=" into up to three groups, each on "^" into up to
// five components, trimming the standard pad. It errors on more than three groups or
// more than five components in a group, or a component over 64 characters.
func ParsePersonName(s string) (PersonName, error) {
	groups := strings.Split(s, "=")
	if len(groups) > 3 {
		return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN has %d component groups, max 3", len(groups))}
	}
	var pn PersonName
	dst := []*NameComponents{&pn.Alphabetic, &pn.Ideographic, &pn.Phonetic}
	for i, g := range groups {
		comps := strings.Split(g, "^")
		if len(comps) > 5 {
			return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN group %d has %d components, max 5", i+1, len(comps))}
		}
		for _, c := range comps {
			if len(strings.TrimRight(c, " ")) > maxPNComponent {
				return PersonName{}, &ValueError{VR: VRPN, Msg: fmt.Sprintf("PN group %d component exceeds 64 characters", i+1)}
			}
		}
		fields := [5]*string{&dst[i].FamilyName, &dst[i].GivenName, &dst[i].MiddleName, &dst[i].Prefix, &dst[i].Suffix}
		for j, c := range comps {
			*fields[j] = strings.TrimRight(c, " ")
		}
	}
	return pn, nil
}

// String renders the canonical "=" / "^" form, dropping trailing empty components and
// trailing empty groups (so "Doe^John" not "Doe^John^^^==").
func (p PersonName) String() string {
	groups := [3]NameComponents{p.Alphabetic, p.Ideographic, p.Phonetic}
	rendered := make([]string, 0, 3)
	for _, g := range groups {
		comps := []string{g.FamilyName, g.GivenName, g.MiddleName, g.Prefix, g.Suffix}
		end := len(comps)
		for end > 0 && comps[end-1] == "" {
			end-- // drop trailing empty components
		}
		rendered = append(rendered, strings.Join(comps[:end], "^"))
	}
	end := len(rendered)
	for end > 0 && rendered[end-1] == "" {
		end-- // drop trailing empty groups
	}
	return strings.Join(rendered[:end], "=")
}
