package altpkg

import (
	"github.com/tempusfrangit/go-xdr"
	altpkg2 "github.com/tempusfrangit/go-xdr/codegen_test/alt_pkg2"
)

// Type alias that might have its own Encode method
type MyBytes []byte

// This Encode method is NOT compatible with xdr.Codec interface
func (m *MyBytes) Encode() string {
	return "custom encoding"
}

func (m *MyBytes) Decode() string {
	return "custom decoding"
}

// Another alias for testing
type MyString string

// Non-conforming methods on MyString too
func (m MyString) Encode() int {
	return 42
}

func (m MyString) Decode() bool {
	return true
}

type AnotherString string

type MyStruct struct {
	Field string
}

type MultiDepthAlias = altpkg2.MyString

// Build assertion to verify MultiDepthAlias inherits interface compliance from altpkg2.MyString
var _ xdr.Codec = MultiDepthAlias("")
