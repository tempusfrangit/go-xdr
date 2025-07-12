package main

//go:generate ../../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// Status type for OperationResult discriminant
type Status uint32

// Status values for OperationResult
const (
	StatusSuccess Status = 0
	StatusError   Status = 1
	StatusPending Status = 2
)

// OperationResult demonstrates a discriminated union based on Status
// The Data field is a union: for StatusSuccess, it is OpSuccessResult; for StatusError and StatusPending, it is nil.
type OperationResult struct {
	Status Status `xdr:"key"`
	Data   []byte `xdr:"union,default=nil"`
}

// OpSuccessResult for successful operations
// (payload for StatusSuccess)
//
//xdr:union=OperationResult,case=StatusSuccess
type OpSuccessResult struct {
	Message string `xdr:"string"`
}

// MessageType type for NetworkMessage discriminant
type MessageType uint32

// Message type constants
const (
	MessageTypeText   MessageType = 1
	MessageTypeBinary MessageType = 2
	MessageTypeVoid   MessageType = 3
)

// NetworkMessage demonstrates discriminated union with different payload types
type NetworkMessage struct {
	Type    MessageType `xdr:"key"`
	Payload []byte      `xdr:"union,default=nil"`
}

// TextPayload for text messages
//
//xdr:union=NetworkMessage,case=MessageTypeText
type TextPayload struct {
	Content string `xdr:"string"`
	Sender  string `xdr:"string"`
}

// BinaryPayload for binary messages
//
//xdr:union=NetworkMessage,case=MessageTypeBinary
type BinaryPayload struct {
	Data     []byte `xdr:"bytes"`
	Checksum uint32 `xdr:"uint32"`
}

// OpType type for FileOperation discriminant
type OpType uint32

// Operation type constants
const (
	OpTypeRead   OpType = 1
	OpTypeWrite  OpType = 2
	OpTypeDelete OpType = 3
)

// FileOperation demonstrates conditional result data
type FileOperation struct {
	OpType OpType `xdr:"key"`
	Result []byte `xdr:"union,default=nil"`
}

// ReadResult contains the result of a read operation
//
//xdr:union=FileOperation,case=OpTypeRead
type ReadResult struct {
	Success bool   `xdr:"bool"`
	Data    []byte `xdr:"bytes"`
	Size    uint32 `xdr:"uint32"`
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*OperationResult)(nil)
var _ xdr.Codec = (*OpSuccessResult)(nil)
var _ xdr.Codec = (*NetworkMessage)(nil)
var _ xdr.Codec = (*TextPayload)(nil)
var _ xdr.Codec = (*BinaryPayload)(nil)
var _ xdr.Codec = (*FileOperation)(nil)
var _ xdr.Codec = (*ReadResult)(nil)
