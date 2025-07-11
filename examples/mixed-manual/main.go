package main

import (
	"fmt"
	"log"

	"github.com/tempusfrangit/go-xdr"
)

func main() {
	fmt.Println("=== Mixed Manual XDR Encoding/Decoding Example ===")

	// Example 1: Structure with manual XDR implementation
	fmt.Println("\n1. Custom XDR implementation...")

	custom := &CustomMessage{
		Header: MessageHeader{
			Version:   1,
			MessageID: 12345,
			Timestamp: 1640995200, // 2022-01-01 00:00:00 UTC
		},
		Body: []byte("Hello, custom XDR!"),
	}

	fmt.Printf("Original custom message: %+v\n", custom)

	// Marshal using custom implementation
	data, err := xdr.Marshal(custom)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled (%d bytes): %x\n", len(data), data)

	// Unmarshal using custom implementation
	var decoded CustomMessage
	err = xdr.Unmarshal(data, &decoded)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled: %+v\n", decoded)

	// Example 2: Mixed auto-generated and manual
	fmt.Println("\n2. Mixed auto-generated and manual...")

	mixed := &MixedStruct{
		AutoField:   "auto-generated encoding",
		ManualField: []uint32{1, 2, 3, 4, 5},
		Header: MessageHeader{
			Version:   2,
			MessageID: 54321,
			Timestamp: 1640995260,
		},
	}

	fmt.Printf("Original mixed struct: %+v\n", mixed)

	data, err = xdr.Marshal(mixed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled mixed (%d bytes): %x\n", len(data), data)

	var decodedMixed MixedStruct
	err = xdr.Unmarshal(data, &decodedMixed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled mixed: %+v\n", decodedMixed)

	// Example 3: Manual override with validation
	fmt.Println("\n3. Manual override with validation...")

	validated := &ValidatedData{
		Value:    42,
		Checksum: 0, // Will be calculated during marshaling
	}

	fmt.Printf("Original validated data: %+v\n", validated)

	data, err = xdr.Marshal(validated)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled validated (%d bytes): %x\n", len(data), data)

	var decodedValidated ValidatedData
	err = xdr.Unmarshal(data, &decodedValidated)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled validated: %+v\n", decodedValidated)

	// Example 4: Custom encoding for special types
	fmt.Println("\n4. Custom encoding for special types...")

	timestamp := &TimestampMessage{
		Message:   "This is a timestamped message",
		CreatedAt: 1640995200,
	}

	fmt.Printf("Original timestamp: %+v\n", timestamp)

	data, err = xdr.Marshal(timestamp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled timestamp (%d bytes): %x\n", len(data), data)

	var decodedTimestamp TimestampMessage
	err = xdr.Unmarshal(data, &decodedTimestamp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled timestamp: %+v\n", decodedTimestamp)

	fmt.Println("\n=== Mixed manual example complete ===")
}
