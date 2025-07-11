package main

//go:generate ../../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// Status values for OperationResult
const (
	StatusSuccess = 0
	StatusError   = 1
	StatusPending = 2
)

// OperationResult demonstrates a discriminated union based on Status
// When Status is StatusSuccess, Data field contains the result
// When Status is StatusError or StatusPending, Data field is void (nil)
type OperationResult struct {
	Status uint32 `xdr:"discriminant"`
	Data   []byte `xdr:"union:0,void:default"`
}

// Message type constants
const (
	MessageTypeText   = 1
	MessageTypeBinary = 2
	MessageTypeVoid   = 3
)

// NetworkMessage demonstrates discriminated union with different payload types
type NetworkMessage struct {
	Type    uint32 `xdr:"discriminant"`
	Payload []byte `xdr:"union:1,2,void:default"`
}

// TextPayload for text messages
type TextPayload struct {
	Content string `xdr:"string"`
	Sender  string `xdr:"string"`
}

// BinaryPayload for binary messages
type BinaryPayload struct {
	Data     []byte `xdr:"bytes"`
	Checksum uint32 `xdr:"uint32"`
}

// Operation type constants
const (
	OpTypeRead   = 1
	OpTypeWrite  = 2
	OpTypeDelete = 3
)

// FileOperation demonstrates conditional result data
// Read operations have ReadResult data
// Write and Delete operations have no result data (void)
type FileOperation struct {
	OpType uint32 `xdr:"discriminant"`
	Result []byte `xdr:"union:1,void:2,void:3"`
}

// ReadResult contains the result of a read operation
type ReadResult struct {
	Success bool   `xdr:"bool"`
	Data    []byte `xdr:"bytes"`
	Size    uint32 `xdr:"uint32"`
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*OperationResult)(nil)
var _ xdr.Codec = (*NetworkMessage)(nil)
var _ xdr.Codec = (*TextPayload)(nil)
var _ xdr.Codec = (*BinaryPayload)(nil)
var _ xdr.Codec = (*FileOperation)(nil)
var _ xdr.Codec = (*ReadResult)(nil)
