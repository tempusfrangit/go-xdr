package codegen_test

//go:generate ../bin/xdrgen $GOFILE

type MessageType uint32

const (
	MessageTypeText MessageType = 1
	MessageTypeData MessageType = 2
	MessageTypeVoid MessageType = 3
)

// +xdr:generate  
// Container struct with auto-detected union
type NetworkMessage struct {
	Type MessageType `xdr:"key"` // discriminant field
	Data []byte                 // auto-detected union field
}
