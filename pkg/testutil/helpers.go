package testutil

// StringPtr returns a pointer to the string value
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the int value
func IntPtr(i int) *int {
	return &i
}