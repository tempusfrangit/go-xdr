package xdr_test

import (
	"fmt"
	"log"

	"github.com/tempusfrangit/go-xdr"
)

// Example demonstrates basic XDR encoding and decoding
func Example_basic() {
	// Create a buffer for encoding
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	// Encode various data types
	encoder.EncodeUint32(42)
	encoder.EncodeString("hello")
	encoder.EncodeBytes([]byte("world"))
	encoder.EncodeBool(true)

	// Get the encoded data
	encoded := encoder.Bytes()
	fmt.Printf("Encoded %d bytes\n", len(encoded))

	// Create a decoder
	decoder := xdr.NewDecoder(encoded)

	// Decode the data
	num, _ := decoder.DecodeUint32()
	str, _ := decoder.DecodeString()
	bytes, _ := decoder.DecodeBytes()
	flag, _ := decoder.DecodeBool()

	fmt.Printf("Decoded: %d, %s, %s, %t\n", num, str, string(bytes), flag)

	// Output:
	// Encoded 32 bytes
	// Decoded: 42, hello, world, true
}

// Example of a struct implementing the Codec interface
type Person struct {
	ID   uint32
	Name string
	Age  uint32
}

func (p *Person) Encode(enc *xdr.Encoder) error {
	if err := enc.EncodeUint32(p.ID); err != nil {
		return err
	}
	if err := enc.EncodeString(p.Name); err != nil {
		return err
	}
	if err := enc.EncodeUint32(p.Age); err != nil {
		return err
	}
	return nil
}

func (p *Person) Decode(dec *xdr.Decoder) error {
	id, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	p.ID = id

	name, err := dec.DecodeString()
	if err != nil {
		return err
	}
	p.Name = name

	age, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	p.Age = age

	return nil
}

// Example demonstrates using the Codec interface
func Example_codec() {
	person := &Person{
		ID:   1,
		Name: "Alice",
		Age:  30,
	}

	// Marshal using the Codec interface
	data, err := xdr.Marshal(person)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled %d bytes\n", len(data))

	// Unmarshal using the Codec interface
	var decoded Person
	err = xdr.Unmarshal(data, &decoded)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled: ID=%d, Name=%s, Age=%d\n", decoded.ID, decoded.Name, decoded.Age)

	// Output:
	// Marshaled 20 bytes
	// Unmarshaled: ID=1, Name=Alice, Age=30
}

// Example demonstrates fixed-size byte arrays
func Example_fixedBytes() {
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	// Encode fixed-size data (no length prefix)
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	encoder.EncodeFixedBytes(data)

	encoded := encoder.Bytes()
	fmt.Printf("Encoded %d bytes (with padding)\n", len(encoded))

	// Decode fixed-size data
	decoder := xdr.NewDecoder(encoded)
	decoded, _ := decoder.DecodeFixedBytes(5)

	fmt.Printf("Decoded: %v\n", decoded)

	// Output:
	// Encoded 8 bytes (with padding)
	// Decoded: [1 2 3 4 5]
}

// Example demonstrates streaming XDR encoding
func Example_streaming() {
	var encoded []byte

	// Create a writer that appends to our byte slice
	writer := &byteWriter{data: &encoded}
	xdrWriter := xdr.NewWriter(writer)

	// Write data using streaming interface
	xdrWriter.WriteUint32(12345)
	xdrWriter.WriteBytes([]byte("streaming example"))

	fmt.Printf("Streamed %d bytes\n", len(encoded))

	// Read back using streaming interface
	reader := &byteReader{data: encoded}
	xdrReader := xdr.NewReader(reader)

	num, _ := xdrReader.ReadUint32()
	data, _ := xdrReader.ReadBytes()

	fmt.Printf("Read back: %d, %s\n", num, string(data))

	// Output:
	// Streamed 28 bytes
	// Read back: 12345, streaming example
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

// Example demonstrates zero-copy slice extraction
func Example_zeroCopy() {
	// Create some test data
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)
	encoder.EncodeUint32(0x12345678)
	encoder.EncodeString("test")
	encoder.EncodeUint32(0x87654321)

	encoded := encoder.Bytes()
	decoder := xdr.NewDecoder(encoded)

	// Decode first uint32
	val1, _ := decoder.DecodeUint32()
	fmt.Printf("First value: 0x%08x\n", val1)

	// Get current position
	pos1 := decoder.Position()

	// Skip the string by decoding it
	decoder.DecodeString()

	// Get position after string
	pos2 := decoder.Position()

	// Extract the string bytes using zero-copy slice
	// This gives us direct access to the encoded string data
	stringBytes := decoder.GetSlice(pos1, pos2)
	fmt.Printf("String bytes (raw XDR): %v\n", stringBytes)

	// Decode final uint32
	val2, _ := decoder.DecodeUint32()
	fmt.Printf("Final value: 0x%08x\n", val2)

	// Output:
	// First value: 0x12345678
	// String bytes (raw XDR): [0 0 0 4 116 101 115 116]
	// Final value: 0x87654321
}
