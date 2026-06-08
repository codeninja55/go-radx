package command

import "strings"

// splitKeyValue splits a "key=value" flag value on its FIRST '=', so a value may itself contain
// '=' (a DICOM date range or a base64 token). It returns ok == false when no '=' is present or the
// key is empty, which the caller reports as a usage error. The value may be empty (a universal
// match key, or an explicit empty insert).
func splitKeyValue(raw string) (key, value string, ok bool) {
	idx := strings.IndexByte(raw, '=')
	if idx <= 0 {
		return "", "", false
	}
	return raw[:idx], raw[idx+1:], true
}
