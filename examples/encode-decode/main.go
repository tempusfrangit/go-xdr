package main

import (
	"fmt"
	"log"

	"github.com/tempusfrangit/go-xdr"
)

func main() {
	fmt.Println("=== Basic XDR Encoding/Decoding Example ===")

	// Create a buffer for encoding
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	// Encode various primitive types
	fmt.Println("\n1. Encoding primitive types...")
	err := encoder.EncodeUint32(42)
	if err != nil {
		log.Fatal(err)
	}
	err = encoder.EncodeString("hello world")
	if err != nil {
		log.Fatal(err)
	}
	err = encoder.EncodeBytes([]byte("example data"))
	if err != nil {
		log.Fatal(err)
	}
	err = encoder.EncodeBool(true)
	if err != nil {
		log.Fatal(err)
	}
	err = encoder.EncodeInt64(-123456789)
	if err != nil {
		log.Fatal(err)
	}

	// Get the encoded data
	encoded := encoder.Bytes()
	fmt.Printf("Encoded %d bytes: %x\n", len(encoded), encoded)

	// Create a decoder
	decoder := xdr.NewDecoder(encoded)

	// Decode the data back
	fmt.Println("\n2. Decoding primitive types...")
	num, err := decoder.DecodeUint32()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded uint32: %d\n", num)

	str, err := decoder.DecodeString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded string: %s\n", str)

	bytes, err := decoder.DecodeBytes()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded bytes: %s\n", string(bytes))

	flag, err := decoder.DecodeBool()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded bool: %t\n", flag)

	bigNum, err := decoder.DecodeInt64()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded int64: %d\n", bigNum)

	// Example with fixed-size data
	fmt.Println("\n3. Fixed-size data example...")
	encoder.Reset(buf)
	fixedData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	err = encoder.EncodeFixedBytes(fixedData)
	if err != nil {
		log.Fatal(err)
	}

	encoded = encoder.Bytes()
	fmt.Printf("Fixed bytes encoded (%d bytes with padding): %x\n", len(encoded), encoded)

	decoder.Reset(encoded)
	decoded, err := decoder.DecodeFixedBytes(8)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Fixed bytes decoded: %x\n", decoded)

	// Example with arrays
	fmt.Println("\n4. Array example...")
	encoder.Reset(buf)
	numbers := []uint32{100, 200, 300, 400, 500}

	// Encode array manually (length + elements)
	// #nosec G115
	err = encoder.EncodeUint32(uint32(len(numbers)))
	if err != nil {
		log.Fatal(err)
	}
	for _, num := range numbers {
		err = encoder.EncodeUint32(num)
		if err != nil {
			log.Fatal(err)
		}
	}

	encoded = encoder.Bytes()
	fmt.Printf("Array encoded (%d bytes): %x\n", len(encoded), encoded)

	decoder.Reset(encoded)
	arrayLen, err := decoder.DecodeUint32()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Array length: %d\n", arrayLen)

	decodedNumbers := make([]uint32, arrayLen)
	for i := uint32(0); i < arrayLen; i++ {
		decodedNumbers[i], err = decoder.DecodeUint32()
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Array decoded: %v\n", decodedNumbers)

	// Example with streaming
	fmt.Println("\n5. Streaming example...")
	demonstrateStreaming()

	fmt.Println("\n=== Example complete ===")
}

func demonstrateStreaming() {
	var encoded []byte

	// Create a writer that appends to our byte slice
	writer := &byteWriter{data: &encoded}
	xdrWriter := xdr.NewWriter(writer)

	// Write data using streaming interface
	err := xdrWriter.WriteUint32(98765)
	if err != nil {
		log.Fatal(err)
	}
	err = xdrWriter.WriteBytes([]byte("streaming test"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Streamed %d bytes: %x\n", len(encoded), encoded)

	// Read back using streaming interface
	reader := &byteReader{data: encoded}
	xdrReader := xdr.NewReader(reader)

	num, err := xdrReader.ReadUint32()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Stream read uint32: %d\n", num)

	data, err := xdrReader.ReadBytes()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Stream read bytes: %s\n", string(data))
}

// Helper types for streaming example
type byteWriter struct {
	data *[]byte
}

func (w *byteWriter) Write(p []byte) (int, error) {
	*w.data = append(*w.data, p...)
	return len(p), nil
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
