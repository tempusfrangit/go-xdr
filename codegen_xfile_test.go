package xdr_test

//go:generate ./bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// CrossFileStruct is defined in another file in the same package
// This tests that the codegen can detect struct types from other files
type CrossFileStruct struct {
	ID   uint32 `xdr:"uint32"`
	Name string `xdr:"string"`
}

// TestCrossFileArray tests array encoding/decoding with structs from other files
type TestCrossFileArray struct {
	Items []CrossFileStruct `xdr:"array"`
	Count uint32            `xdr:"uint32"`
}

// TestCrossFileStruct tests direct struct encoding/decoding with structs from other files
type TestCrossFileStruct struct {
	Data CrossFileStruct `xdr:"struct"`
	Flag bool            `xdr:"bool"`
}

// Ensure types implement Codec interface
var _ xdr.Codec = (*TestCrossFileArray)(nil)
var _ xdr.Codec = (*TestCrossFileStruct)(nil)
