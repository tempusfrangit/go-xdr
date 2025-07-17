package altpkg2

import "github.com/tempusfrangit/go-xdr"

type MyString string

// MyString implements xdr.Codec for test compilation
func (s MyString) Encode(enc *xdr.Encoder) error {
	return nil
}

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

type ExportedPrivateString = privateString

// Build assertions to verify interface compliance
var _ xdr.Codec = MyString("")
var _ xdr.Codec = privateString("")
var _ xdr.Codec = ExportedPrivateString("")
