// Package internal contains internal code.
package internal

import "fmt"

// StringPointerToString converts a string pointer to a string. When the
// pointer is null, we return the "nil" string.
func StringPointerToString(v *string) (out string) {
	out = "nil"
	if v != nil {
		out = fmt.Sprintf("%+v", *v)
	}
	return
}

// BoolPointerToString is like StringPointerToString but for bool.
func BoolPointerToString(v *bool) (out string) {
	out = "nil"
	if v != nil {
		out = fmt.Sprintf("%+v", *v)
	}
	return
}
