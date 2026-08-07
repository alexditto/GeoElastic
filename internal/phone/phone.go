// Package phone normalizes phone numbers so callers can match across
// formatting differences (dashes, parens, spaces, a leading country code)
// without treating them as meaningfully different numbers.
package phone

// Normalize strips everything but digits and drops a leading US country
// code ("1" prefix on an 11-digit number), returning a canonical
// digits-only representation. Numbers that don't fit that shape (wrong
// length, non-US) are returned digits-only as-is.
func Normalize(raw string) string {
	digits := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			digits = append(digits, raw[i])
		}
	}
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}
	return string(digits)
}

// Local returns the last 7 digits of a normalized number — the central
// office code (prefix) plus subscriber line number — deliberately excluding
// the area code, which is shared by too many unrelated businesses in the
// same region to be a meaningful match signal on its own. Returns "" if
// normalized doesn't carry enough digits to extract one.
func Local(normalized string) string {
	if len(normalized) < 7 {
		return ""
	}
	return normalized[len(normalized)-7:]
}
