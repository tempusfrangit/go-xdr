package main

//go:generate xdrgen

// Payload struct for text messages
type TextMessage struct {
	Content string `xdr:"string"`
}

// Payload struct for data messages
type DataMessage struct {
	Bytes []byte `xdr:"bytes"`
}

// Union comment on payload struct
//
//xdr:union=NetworkMessage,case=MessageTypeText
type TextMessagePayload struct {
	Content string `xdr:"string"`
}

//xdr:union=NetworkMessage,case=MessageTypeData
type DataMessagePayload struct {
	Bytes []byte `xdr:"bytes"`
}
