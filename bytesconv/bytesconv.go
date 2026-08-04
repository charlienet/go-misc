package bytesconv

import (
	"unsafe"
)

// StringToBytes converts string to byte slice without a memory allocation.
//
// WARNING: The returned byte slice shares memory with the original string.
// Do NOT modify the returned byte slice, as strings in Go are immutable.
// Modifying it will cause undefined behavior and may corrupt other data.
//
// This function uses unsafe.Pointer to bypass Go's type system for zero-copy conversion.
// Only use this when you need read-only access to string data as bytes and performance is critical.
func StringToBytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// BytesToString converts byte slice to string without a memory allocation.
//
// WARNING: If the original byte slice is modified after this call, the returned string
// may also change (undefined behavior). Only use this when you can guarantee the byte
// slice will not be modified, or when you need read-only access.
//
// This function uses unsafe.Pointer to bypass Go's type system for zero-copy conversion.
// Only use this when performance is critical and you understand the implications.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
