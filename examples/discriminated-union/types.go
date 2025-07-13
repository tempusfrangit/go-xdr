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

// +xdr:generate
// OperationResult demonstrates auto-detected discriminated union
// StatusError and StatusPending are automatically void cases (no payload structs)
type OperationResult struct {
	Status Status `xdr:"key"` // discriminant
	Data   []byte              // auto-detected union field
}

// +xdr:generate
// OpSuccessResult for successful operations (payload for StatusSuccess)
type OpSuccessResult struct {
	Message string
}

// MessageType type for NetworkMessage discriminant
type MessageType uint32

// Message type constants
const (
	MessageTypeText   MessageType = 1
	MessageTypeBinary MessageType = 2
	MessageTypeVoid   MessageType = 3
)

// +xdr:generate
// +xdr:union=MessageType
// +xdr:case=MessageTypeText
// +xdr:case=MessageTypeBinary
// NetworkMessage demonstrates discriminated union with different payload types
// MessageTypeVoid is automatically a void case
type NetworkMessage struct {
	Type    MessageType `xdr:"key"`
	Payload []byte
}

// +xdr:generate
// TextPayload for text messages
type TextPayload struct {
	Content string
	Sender  string
}

// +xdr:generate
// BinaryPayload for binary messages
type BinaryPayload struct {
	Data     []byte
	Checksum uint32
}

// OpType type for FileOperation discriminant
type OpType uint32

// Operation type constants
const (
	OpTypeRead   OpType = 1
	OpTypeWrite  OpType = 2
	OpTypeDelete OpType = 3
)

// +xdr:generate
// +xdr:union=OpType
// +xdr:case=OpTypeRead
// FileOperation demonstrates conditional result data
// OpTypeWrite and OpTypeDelete are automatically void cases
type FileOperation struct {
	OpType OpType `xdr:"key"`
	Result []byte
}

// +xdr:generate
// ReadResult contains the result of a read operation
type ReadResult struct {
	Success bool
	Data    []byte
	Size    uint32
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*OperationResult)(nil)
var _ xdr.Codec = (*OpSuccessResult)(nil)
var _ xdr.Codec = (*NetworkMessage)(nil)
var _ xdr.Codec = (*TextPayload)(nil)
var _ xdr.Codec = (*BinaryPayload)(nil)
var _ xdr.Codec = (*FileOperation)(nil)
var _ xdr.Codec = (*ReadResult)(nil)
