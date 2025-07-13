package codegen_test

//go:generate ../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// +xdr:generate
// CrossFileStruct is defined in another file in the same package
// This tests that the codegen can detect struct types from other files
type CrossFileStruct struct {
	ID   uint32 // auto-detected as uint32
	Name string // auto-detected as string
}

// +xdr:generate
// TestCrossFileArray tests array encoding/decoding with structs from other files
type TestCrossFileArray struct {
	Items []CrossFileStruct // auto-detected as struct array
	Count uint32            // auto-detected as uint32
}

// +xdr:generate
// TestCrossFileStruct tests direct struct encoding/decoding with structs from other files
type TestCrossFileStruct struct {
	Data CrossFileStruct // auto-detected as struct
	Flag bool            // auto-detected as bool
}

// Ensure types implement Codec interface
var _ xdr.Codec = (*TestCrossFileArray)(nil)
var _ xdr.Codec = (*TestCrossFileStruct)(nil)
