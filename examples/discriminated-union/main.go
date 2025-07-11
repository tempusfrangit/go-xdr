package main

//go:generate ../../bin/xdrgen types.go

import (
	"fmt"
	"log"

	"github.com/tempusfrangit/go-xdr"
)

func main() {
	fmt.Println("=== Discriminated Union XDR Example ===")

	// Example 1: Operation Result with discriminated union
	fmt.Println("\n1. Operation Result Example...")

	// Success case
	successResult := &OperationResult{
		Status: StatusSuccess,
		Data:   []byte("operation completed successfully"),
	}

	fmt.Printf("Success result: status=%d, data=%s\n", successResult.Status, string(successResult.Data))

	data, err := xdr.Marshal(successResult)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled success result (%d bytes): %x\n", len(data), data)

	var decodedSuccess OperationResult
	err = xdr.Unmarshal(data, &decodedSuccess)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled: status=%d, data=%s\n", decodedSuccess.Status, string(decodedSuccess.Data))

	// Error case (void - no data)
	errorResult := &OperationResult{
		Status: StatusError,
		Data:   nil, // No data for error case
	}

	fmt.Printf("\nError result: status=%d, data=nil\n", errorResult.Status)

	data, err = xdr.Marshal(errorResult)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled error result (%d bytes): %x\n", len(data), data)

	var decodedError OperationResult
	err = xdr.Unmarshal(data, &decodedError)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled: status=%d, data=%v\n", decodedError.Status, decodedError.Data)

	// Example 2: Network Message with different payload types
	fmt.Println("\n2. Network Message Example...")

	// Text message
	textMsg := &NetworkMessage{
		Type:    MessageTypeText,
		Payload: []byte("Hello, world!"),
	}

	fmt.Printf("Text message: %+v\n", textMsg)

	data, err = xdr.Marshal(textMsg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled text message (%d bytes)\n", len(data))

	var decodedTextMsg NetworkMessage
	err = xdr.Unmarshal(data, &decodedTextMsg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled text message: %+v\n", decodedTextMsg)

	// Binary message
	binaryMsg := &NetworkMessage{
		Type:    MessageTypeBinary,
		Payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	fmt.Printf("\nBinary message: %+v\n", binaryMsg)

	data, err = xdr.Marshal(binaryMsg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled binary message (%d bytes)\n", len(data))

	var decodedBinaryMsg NetworkMessage
	err = xdr.Unmarshal(data, &decodedBinaryMsg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled binary message: %+v\n", decodedBinaryMsg)

	// Example 3: File Operation with conditional data
	fmt.Println("\n3. File Operation Example...")

	// Read operation (has result data)
	readOp := &FileOperation{
		OpType: OpTypeRead,
		Result: []byte("file content here"),
	}

	fmt.Printf("Read operation: %+v\n", readOp)

	data, err = xdr.Marshal(readOp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled read operation (%d bytes)\n", len(data))

	var decodedReadOp FileOperation
	err = xdr.Unmarshal(data, &decodedReadOp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled read operation: %+v\n", decodedReadOp)

	// Delete operation (no result data - void case)
	deleteOp := &FileOperation{
		OpType: OpTypeDelete,
		Result: nil, // No result data for delete
	}

	fmt.Printf("\nDelete operation: %+v\n", deleteOp)

	data, err = xdr.Marshal(deleteOp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled delete operation (%d bytes)\n", len(data))

	var decodedDeleteOp FileOperation
	err = xdr.Unmarshal(data, &decodedDeleteOp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Unmarshaled delete operation: %+v\n", decodedDeleteOp)

	fmt.Println("\n=== Discriminated Union example complete ===")
}
