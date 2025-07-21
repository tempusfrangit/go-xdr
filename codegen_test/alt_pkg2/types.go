// Package altpkg2 provides additional test types for multi-depth alias resolution
package altpkg2

import "github.com/tempusfrangit/go-xdr"

// MyString represents a string type that implements xdr.Codec
type MyString string

// Encode implements xdr.Codec for test compilation
func (s MyString) Encode(enc *xdr.Encoder) error {
	return nil
}

// Decode implements xdr.Codec for test compilation
func (s MyString) Decode(dec *xdr.Decoder) error {
	return nil
}

type privateString string

// privateString implements xdr.Codec for test compilation
func (s privateString) Encode(enc *xdr.Encoder) error {
	return nil
}

func (s privateString) Decode(dec *xdr.Decoder) error {
	return nil
}

// ExportedPrivateString exposes the private string type
type ExportedPrivateString = privateString

// Build assertions to verify interface compliance
var _ xdr.Codec = MyString("")
var _ xdr.Codec = privateString("")
var _ xdr.Codec = ExportedPrivateString("")
