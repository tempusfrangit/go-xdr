package main

//go:generate ../../bin/xdrgen $GOFILE

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/tempusfrangit/go-xdr"
)

// +xdr:generate
// MessageHeader is auto-generated but used in manual implementations
type MessageHeader struct {
	Version   uint32
	MessageID uint32
	Timestamp uint32
}

// CustomMessage demonstrates fully manual XDR implementation
type CustomMessage struct {
	Header MessageHeader `xdr:"-"`
	Body   []byte        `xdr:"-"`
}

// Encode implements custom XDR marshaling
func (c *CustomMessage) Encode(enc *xdr.Encoder) error {
	// Manual encoding with custom logic
	if err := enc.EncodeUint32(c.Header.Version); err != nil {
		return err
	}
	if err := enc.EncodeUint32(c.Header.MessageID); err != nil {
		return err
	}
	if err := enc.EncodeUint32(c.Header.Timestamp); err != nil {
		return err
	}

	// Custom encoding: prefix body with magic number
	if err := enc.EncodeUint32(0xDEADBEEF); err != nil {
		return err
	}

	return enc.EncodeBytes(c.Body)
}

// Decode implements custom XDR unmarshaling
func (c *CustomMessage) Decode(dec *xdr.Decoder) error {
	var err error

	// Manual decoding
	if c.Header.Version, err = dec.DecodeUint32(); err != nil {
		return err
	}
	if c.Header.MessageID, err = dec.DecodeUint32(); err != nil {
		return err
	}
	if c.Header.Timestamp, err = dec.DecodeUint32(); err != nil {
		return err
	}

	// Check for magic number
	magic, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	if magic != 0xDEADBEEF {
		return fmt.Errorf("invalid magic number: 0x%08x", magic)
	}

	c.Body, err = dec.DecodeBytes()
	return err
}

// MixedStruct combines auto-generated and manual XDR
type MixedStruct struct {
	AutoField   string        // Auto-generated
	ManualField []uint32      `xdr:"-"` // Manual implementation
	Header      MessageHeader // Auto-generated
}

// Encode implements mixed XDR marshaling
func (m *MixedStruct) Encode(enc *xdr.Encoder) error {
	// Auto-generated field
	if err := enc.EncodeString(m.AutoField); err != nil {
		return err
	}

	// Manual field with custom encoding (reversed order)
	// #nosec G115
	if err := enc.EncodeUint32(uint32(len(m.ManualField))); err != nil {
		return err
	}
	for i := len(m.ManualField) - 1; i >= 0; i-- {
		if err := enc.EncodeUint32(m.ManualField[i]); err != nil {
			return err
		}
	}

	// Auto-generated struct
	return m.Header.Encode(enc)
}

// Decode implements mixed XDR unmarshaling
func (m *MixedStruct) Decode(dec *xdr.Decoder) error {
	var err error

	// Auto-generated field
	if m.AutoField, err = dec.DecodeString(); err != nil {
		return err
	}

	// Manual field with custom decoding (reversed order)
	length, err := dec.DecodeUint32()
	if err != nil {
		return err
	}

	m.ManualField = make([]uint32, length)
	for i := int(length) - 1; i >= 0; i-- {
		if m.ManualField[i], err = dec.DecodeUint32(); err != nil {
			return err
		}
	}

	// Auto-generated struct
	return m.Header.Decode(dec)
}

// ValidatedData demonstrates manual override with validation
type ValidatedData struct {
	Value    uint32 `xdr:"-"`
	Checksum uint32 `xdr:"-"`
}

// Encode implements marshaling with checksum calculation
func (v *ValidatedData) Encode(enc *xdr.Encoder) error {
	// Calculate checksum before marshaling
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, v.Value)
	v.Checksum = crc32.ChecksumIEEE(data)

	if err := enc.EncodeUint32(v.Value); err != nil {
		return err
	}
	return enc.EncodeUint32(v.Checksum)
}

// Decode implements unmarshaling with checksum validation
func (v *ValidatedData) Decode(dec *xdr.Decoder) error {
	var err error

	if v.Value, err = dec.DecodeUint32(); err != nil {
		return err
	}
	if v.Checksum, err = dec.DecodeUint32(); err != nil {
		return err
	}

	// Validate checksum
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, v.Value)
	expectedChecksum := crc32.ChecksumIEEE(data)

	if v.Checksum != expectedChecksum {
		return fmt.Errorf("checksum validation failed: expected 0x%08x, got 0x%08x",
			expectedChecksum, v.Checksum)
	}

	return nil
}

// TimestampMessage demonstrates custom encoding for special types
type TimestampMessage struct {
	Message   string // Auto-detected as string
	CreatedAt uint32 `xdr:"-"` // Custom encoded as both timestamp and readable format
}

// Encode implements custom timestamp encoding
func (t *TimestampMessage) Encode(enc *xdr.Encoder) error {
	if err := enc.EncodeString(t.Message); err != nil {
		return err
	}

	// Encode timestamp
	if err := enc.EncodeUint32(t.CreatedAt); err != nil {
		return err
	}

	// Also encode a readable format for debugging
	readableTime := fmt.Sprintf("timestamp:%d", t.CreatedAt)
	return enc.EncodeString(readableTime)
}

// Decode implements custom timestamp decoding
func (t *TimestampMessage) Decode(dec *xdr.Decoder) error {
	var err error

	if t.Message, err = dec.DecodeString(); err != nil {
		return err
	}

	if t.CreatedAt, err = dec.DecodeUint32(); err != nil {
		return err
	}

	// Read and validate the readable format
	readableTime, err := dec.DecodeString()
	if err != nil {
		return err
	}

	expectedReadable := fmt.Sprintf("timestamp:%d", t.CreatedAt)
	if readableTime != expectedReadable {
		return fmt.Errorf("timestamp validation failed: expected %s, got %s",
			expectedReadable, readableTime)
	}

	return nil
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*CustomMessage)(nil)
var _ xdr.Codec = (*MixedStruct)(nil)
var _ xdr.Codec = (*ValidatedData)(nil)
var _ xdr.Codec = (*TimestampMessage)(nil)
var _ xdr.Codec = (*MessageHeader)(nil)
