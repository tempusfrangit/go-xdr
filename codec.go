package xdr

import (
	"fmt"
)

// Codec interface provides consistent XDR encoding/decoding for types
type Codec interface {
	// Encode encodes the type to XDR format using the provided encoder
	Encode(enc *Encoder) error

	// Decode decodes the type from XDR format using the provided decoder
	Decode(dec *Decoder) error
}

// Marshal provides generic XDR encoding for any type implementing Codec
func Marshal(codec Codec) ([]byte, error) {
	// Use a reasonable initial buffer size
	buf := make([]byte, 512)
	enc := NewEncoder(buf)

	if err := codec.Encode(enc); err != nil {
		return nil, fmt.Errorf("XDR encoding failed: %w", err)
	}

	// Return copy of encoded data (same pattern as existing NFS code)
	result := make([]byte, len(enc.Bytes()))
	copy(result, enc.Bytes())
	return result, nil
}

// MarshalRaw wraps pre-encoded XDR data in a consistent interface
// Used for exceptional cases like sparse attribute encoding where
// custom encoding logic is required
func MarshalRaw(data []byte) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	// Return copy to maintain consistent ownership semantics with Marshal
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Unmarshal provides generic XDR decoding for any type implementing Codec
func Unmarshal(data []byte, codec Codec) error {
	dec := NewDecoder(data)
	if err := codec.Decode(dec); err != nil {
		return fmt.Errorf("XDR decoding failed: %w", err)
	}
	return nil
}
