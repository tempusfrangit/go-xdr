// Package altpkg provides test types for cross-package type resolution
package altpkg

import (
	"github.com/tempusfrangit/go-xdr"
	altpkg2 "github.com/tempusfrangit/go-xdr/codegen_test/alt_pkg2"
)

// MyBytes represents a byte slice type alias that might have its own Encode method
type MyBytes []byte

// Encode provides custom encoding for MyBytes (NOT compatible with xdr.Codec interface)
func (m *MyBytes) Encode() string {
	return "custom encoding"
}

// Decode provides custom decoding for MyBytes
func (m *MyBytes) Decode() string {
	return "custom decoding"
}

// MyString represents another alias for testing
type MyString string

// Encode provides non-conforming methods on MyString too
func (m MyString) Encode() int {
	return 42
}

// Decode provides custom decoding for MyString
func (m MyString) Decode() bool {
	return true
}

// AnotherString is a simple string alias
type AnotherString string

// MyID represents a uint32 alias
type MyID uint32

// MyStruct represents a test struct with a field
type MyStruct struct {
	Field string
}

// MultiDepthAlias represents a multi-level type alias
type MultiDepthAlias = altpkg2.MyString

// Build assertion to verify MultiDepthAlias inherits interface compliance from altpkg2.MyString
var _ xdr.Codec = MultiDepthAlias("")
