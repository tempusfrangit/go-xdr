package consumer

//go:generate ../../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
	"synthetic_test/tokenlib"
)

// +xdr:generate
type Request struct {
	UserToken tokenlib.Token // Should resolve to []byte -> bytes encoding
	SessionID tokenlib.ID    // Should resolve to string -> string encoding
	Checksum  tokenlib.Hash  // Should resolve to [32]byte -> fixed bytes encoding
}

// Ensure we implement xdr.Codec
var _ xdr.Codec = (*Request)(nil)
