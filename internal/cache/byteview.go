// Package cache implements the core caching functionality
package cache

// ByteView holds an immutable view of bytes
type ByteView struct {
	bytes []byte // actual data stored as bytes
}

// Len returns the view's length in bytes
func (v ByteView) Len() int {
	return len(v.bytes)
}

// ByteSlice returns a copy of the data as a byte slice
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.bytes)
}

// String returns the data as a string
func (v ByteView) String() string {
	return string(v.bytes)
}

// cloneBytes creates a copy of the input byte slice
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
