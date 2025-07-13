package main

//go:generate xdrgen $GOFILE

type MessageType uint32

const (
	MessageTypeText MessageType = 1
	MessageTypeData MessageType = 2
	MessageTypeVoid MessageType = 3
)

// Container struct with key field
type NetworkMessage struct {
	Type MessageType `xdr:"key"`   // Operation code (discriminant)
	Data []byte      `xdr:"union"` // XDR-encoded operation-specific arguments
}
