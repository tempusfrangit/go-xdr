package xdr

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// XDR errors
var (
	ErrBufferTooSmall = errors.New("buffer too small")
	ErrInvalidData    = errors.New("invalid XDR data")
	ErrUnexpectedEOF  = errors.New("unexpected end of data")
)

// Encoder provides methods for encoding data in XDR format
type Encoder struct {
	buf []byte
	pos int
}

// NewEncoder creates a new XDR encoder with the provided buffer
func NewEncoder(buf []byte) *Encoder {
	return &Encoder{buf: buf}
}

// Bytes returns the encoded data
func (e *Encoder) Bytes() []byte {
	return e.buf[:e.pos]
}

// Len returns the number of bytes encoded
func (e *Encoder) Len() int {
	return e.pos
}

// Reset resets the encoder to use a new buffer
func (e *Encoder) Reset(buf []byte) {
	e.buf = buf
	e.pos = 0
}

// EncodeUint32 encodes a 32-bit unsigned integer
func (e *Encoder) EncodeUint32(v uint32) error {
	if e.pos+4 > len(e.buf) {
		return ErrBufferTooSmall
	}
	binary.BigEndian.PutUint32(e.buf[e.pos:], v)
	e.pos += 4
	return nil
}

// EncodeUint64 encodes a 64-bit unsigned integer
func (e *Encoder) EncodeUint64(v uint64) error {
	if e.pos+8 > len(e.buf) {
		return ErrBufferTooSmall
	}
	binary.BigEndian.PutUint64(e.buf[e.pos:], v)
	e.pos += 8
	return nil
}

// EncodeInt32 encodes a 32-bit signed integer
func (e *Encoder) EncodeInt32(v int32) error {
	return e.EncodeUint32(uint32(v))
}

// EncodeInt64 encodes a 64-bit signed integer
func (e *Encoder) EncodeInt64(v int64) error {
	return e.EncodeUint64(uint64(v))
}

// EncodeBool encodes a boolean value
func (e *Encoder) EncodeBool(v bool) error {
	if v {
		return e.EncodeUint32(1)
	}
	return e.EncodeUint32(0)
}

// EncodeBytes encodes a variable-length byte array with length prefix
func (e *Encoder) EncodeBytes(v []byte) error {
	if err := e.EncodeUint32(uint32(len(v))); err != nil {
		return err
	}
	return e.EncodeFixedBytes(v)
}

// EncodeFixedBytes encodes a fixed-length byte array without length prefix
func (e *Encoder) EncodeFixedBytes(v []byte) error {
	// Calculate padding needed for 4-byte alignment
	padLen := (4 - (len(v) % 4)) % 4
	totalLen := len(v) + padLen

	if e.pos+totalLen > len(e.buf) {
		return ErrBufferTooSmall
	}

	copy(e.buf[e.pos:], v)
	e.pos += len(v)

	// Add padding bytes (zeros)
	for i := 0; i < padLen; i++ {
		e.buf[e.pos] = 0
		e.pos++
	}

	return nil
}

// EncodeString encodes a string
func (e *Encoder) EncodeString(v string) error {
	return e.EncodeBytes([]byte(v))
}

// Decoder provides methods for decoding XDR format data
type Decoder struct {
	buf []byte
	pos int
}

// NewDecoder creates a new XDR decoder with the provided data
func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf: buf}
}

// Remaining returns the number of bytes remaining to be decoded
func (d *Decoder) Remaining() int {
	return len(d.buf) - d.pos
}

// Position returns the current decode position
func (d *Decoder) Position() int {
	return d.pos
}

// Reset resets the decoder to use new data
func (d *Decoder) Reset(buf []byte) {
	d.buf = buf
	d.pos = 0
}

// GetSlice returns a slice into the decoder's buffer from start to end positions.
// WARNING: The returned slice is ONLY valid until the next decoder operation
// or Reset() call. Callers MUST consume the data immediately.
// This method enables zero-copy extraction of operation-specific XDR bytes.
func (d *Decoder) GetSlice(start, end int) []byte {
	if start < 0 || end > len(d.buf) || start > end {
		return nil
	}
	return d.buf[start:end]
}

// DecodeUint32 decodes a 32-bit unsigned integer
func (d *Decoder) DecodeUint32() (uint32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint32(d.buf[d.pos:])
	d.pos += 4
	return v, nil
}

// DecodeUint64 decodes a 64-bit unsigned integer
func (d *Decoder) DecodeUint64() (uint64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint64(d.buf[d.pos:])
	d.pos += 8
	return v, nil
}

// DecodeInt32 decodes a 32-bit signed integer
func (d *Decoder) DecodeInt32() (int32, error) {
	v, err := d.DecodeUint32()
	return int32(v), err
}

// DecodeInt64 decodes a 64-bit signed integer
func (d *Decoder) DecodeInt64() (int64, error) {
	v, err := d.DecodeUint64()
	return int64(v), err
}

// DecodeBool decodes a boolean value
func (d *Decoder) DecodeBool() (bool, error) {
	v, err := d.DecodeUint32()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// DecodeBytes decodes a variable-length byte array
func (d *Decoder) DecodeBytes() ([]byte, error) {
	length, err := d.DecodeUint32()
	if err != nil {
		return nil, err
	}

	if length > math.MaxInt32 {
		return nil, ErrInvalidData
	}

	return d.DecodeFixedBytes(int(length))
}

// DecodeFixedBytes decodes a fixed-length byte array
func (d *Decoder) DecodeFixedBytes(length int) ([]byte, error) {
	// Calculate total length including padding
	padLen := (4 - (length % 4)) % 4
	totalLen := length + padLen

	if d.pos+totalLen > len(d.buf) {
		return nil, ErrUnexpectedEOF
	}

	// Extract the actual data (without padding)
	data := make([]byte, length)
	copy(data, d.buf[d.pos:d.pos+length])
	d.pos += totalLen // Skip data and padding

	return data, nil
}

// DecodeString decodes a string
func (d *Decoder) DecodeString() (string, error) {
	data, err := d.DecodeBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Writer wraps an io.Writer for streaming XDR encoding
type Writer struct {
	w   io.Writer
	buf [8]byte // Temporary buffer for encoding primitives
}

// NewWriter creates a new XDR writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteUint32 writes a 32-bit unsigned integer
func (w *Writer) WriteUint32(v uint32) error {
	binary.BigEndian.PutUint32(w.buf[:4], v)
	_, err := w.w.Write(w.buf[:4])
	return err
}

// WriteBytes writes a variable-length byte array
func (w *Writer) WriteBytes(v []byte) error {
	if err := w.WriteUint32(uint32(len(v))); err != nil {
		return err
	}

	if _, err := w.w.Write(v); err != nil {
		return err
	}

	// Write padding
	padLen := (4 - (len(v) % 4)) % 4
	if padLen > 0 {
		padding := [4]byte{}
		_, err := w.w.Write(padding[:padLen])
		return err
	}

	return nil
}

// Reader wraps an io.Reader for streaming XDR decoding
type Reader struct {
	r   io.Reader
	buf [8]byte // Temporary buffer for decoding primitives
}

// NewReader creates a new XDR reader
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// ReadUint32 reads a 32-bit unsigned integer
func (r *Reader) ReadUint32() (uint32, error) {
	if _, err := io.ReadFull(r.r, r.buf[:4]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(r.buf[:4]), nil
}

// ReadBytes reads a variable-length byte array
func (r *Reader) ReadBytes() ([]byte, error) {
	length, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}

	if length > math.MaxInt32 {
		return nil, ErrInvalidData
	}

	// Calculate total length including padding
	padLen := (4 - (int(length) % 4)) % 4
	totalLen := int(length) + padLen

	// Read data and padding
	buf := make([]byte, totalLen)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return nil, err
	}

	// Return only the actual data (without padding)
	return buf[:length], nil
}
