//go:build ignore

package codegen_test

//go:generate ../bin/xdrgen $GOFILE

// +xdr:generate
// +xdr:union=NetworkMessage
// +xdr:case=MessageTypeText
// Payload struct for text messages
type TextMessage struct {
	Content string
}

// +xdr:generate
// +xdr:union=NetworkMessage  
// +xdr:case=MessageTypeData
// Payload struct for data messages
type DataMessage struct {
	Bytes []byte
}

// +xdr:generate
// Payload structs no longer need union comments!
type TextMessagePayload struct {
	Content string // auto-detected as string
}

// +xdr:generate
type DataMessagePayload struct {
	Bytes []byte // auto-detected as bytes
}
