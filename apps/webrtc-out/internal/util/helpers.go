package util

// BoolPtr returns a pointer to the given boolean value
func BoolPtr(b bool) *bool {
	return &b
}

// Uint16Ptr returns a pointer to the given uint16 value
func Uint16Ptr(u uint16) *uint16 {
	return &u
}

// IntPtr returns a pointer to the given int value
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to the given string value
func StringPtr(s string) *string {
	return &s
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
