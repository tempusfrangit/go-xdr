package xdr

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// failingWriter fails after N successful writes
type failingWriter struct {
	failAfter int
	writes    int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > w.failAfter {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

// failingReader fails after N successful reads
type failingReader struct {
	data      []byte
	pos       int
	failAfter int
	reads     int
}

func (r *failingReader) Read(p []byte) (int, error) {
	r.reads++
	if r.reads > r.failAfter {
		return 0, errors.New("read failed")
	}

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestEncoder(t *testing.T) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	t.Run("EncodeUint32", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeUint32(0x12345678)
		if err != nil {
			t.Fatalf("EncodeUint32 failed: %v", err)
		}

		expected := []byte{0x12, 0x34, 0x56, 0x78}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeUint64", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeUint64(0x123456789ABCDEF0)
		if err != nil {
			t.Fatalf("EncodeUint64 failed: %v", err)
		}

		expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeInt32", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeInt32(-1)
		if err != nil {
			t.Fatalf("EncodeInt32 failed: %v", err)
		}

		expected := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeInt64", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeInt64(-1)
		if err != nil {
			t.Fatalf("EncodeInt64 failed: %v", err)
		}

		expected := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeBool", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeBool(true)
		if err != nil {
			t.Fatalf("EncodeBool failed: %v", err)
		}

		expected := []byte{0x00, 0x00, 0x00, 0x01}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}

		encoder.Reset(buf)
		err = encoder.EncodeBool(false)
		if err != nil {
			t.Fatalf("EncodeBool failed: %v", err)
		}

		expected = []byte{0x00, 0x00, 0x00, 0x00}
		result = encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeString", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeString("test")
		if err != nil {
			t.Fatalf("EncodeString failed: %v", err)
		}

		// Length (4) + "test" + padding (0)
		expected := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeStringWithPadding", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeString("hello")
		if err != nil {
			t.Fatalf("EncodeString failed: %v", err)
		}

		// Length (5) + "hello" + padding (3 bytes to align to 4)
		expected := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeBytes", func(t *testing.T) {
		encoder.Reset(buf)
		data := []byte{0x01, 0x02, 0x03}
		err := encoder.EncodeBytes(data)
		if err != nil {
			t.Fatalf("EncodeBytes failed: %v", err)
		}

		// Length (3) + data + padding (1 byte)
		expected := []byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03, 0x00}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("EncodeFixedBytes", func(t *testing.T) {
		encoder.Reset(buf)
		data := []byte{0x01, 0x02, 0x03}
		err := encoder.EncodeFixedBytes(data)
		if err != nil {
			t.Fatalf("EncodeFixedBytes failed: %v", err)
		}

		// Data + padding (1 byte) - no length prefix
		expected := []byte{0x01, 0x02, 0x03, 0x00}
		result := encoder.Bytes()
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("BufferTooSmall", func(t *testing.T) {
		t.Run("EncodeUint32", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeUint32(0x12345678)
			if err != ErrBufferTooSmall {
				t.Errorf("Expected ErrBufferTooSmall, got %v", err)
			}
		})

		t.Run("EncodeUint64", func(t *testing.T) {
			smallBuf := make([]byte, 4)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeUint64(0x123456789ABCDEF0)
			if err != ErrBufferTooSmall {
				t.Errorf("Expected ErrBufferTooSmall, got %v", err)
			}
		})

		t.Run("EncodeBytes", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeBytes([]byte("test"))
			if err != ErrBufferTooSmall {
				t.Errorf("Expected ErrBufferTooSmall, got %v", err)
			}
		})

		t.Run("EncodeString", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeString("test")
			if err != ErrBufferTooSmall {
				t.Errorf("Expected ErrBufferTooSmall, got %v", err)
			}
		})

		t.Run("EncodeFixedBytes", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeFixedBytes([]byte("test"))
			if err != ErrBufferTooSmall {
				t.Errorf("Expected ErrBufferTooSmall, got %v", err)
			}
		})
	})
}

func TestDecoder(t *testing.T) {
	t.Run("DecodeUint32", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeUint32()
		if err != nil {
			t.Fatalf("DecodeUint32 failed: %v", err)
		}

		expected := uint32(0x12345678)
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeUint64", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeUint64()
		if err != nil {
			t.Fatalf("DecodeUint64 failed: %v", err)
		}

		expected := uint64(0x123456789ABCDEF0)
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeInt32", func(t *testing.T) {
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeInt32()
		if err != nil {
			t.Fatalf("DecodeInt32 failed: %v", err)
		}

		expected := int32(-1)
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeInt64", func(t *testing.T) {
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeInt64()
		if err != nil {
			t.Fatalf("DecodeInt64 failed: %v", err)
		}

		expected := int64(-1)
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeBool", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x01}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeBool()
		if err != nil {
			t.Fatalf("DecodeBool failed: %v", err)
		}

		if !result {
			t.Errorf("Expected true, got false")
		}

		decoder.Reset([]byte{0x00, 0x00, 0x00, 0x00})
		result, err = decoder.DecodeBool()
		if err != nil {
			t.Fatalf("DecodeBool failed: %v", err)
		}

		if result {
			t.Errorf("Expected false, got true")
		}
	})

	t.Run("DecodeString", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeString()
		if err != nil {
			t.Fatalf("DecodeString failed: %v", err)
		}

		expected := "test"
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeStringWithPadding", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeString()
		if err != nil {
			t.Fatalf("DecodeString failed: %v", err)
		}

		expected := "hello"
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeBytes", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeBytes()
		if err != nil {
			t.Fatalf("DecodeBytes failed: %v", err)
		}

		expected := []byte{0x01, 0x02, 0x03}
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("DecodeFixedBytes", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeFixedBytes(3)
		if err != nil {
			t.Fatalf("DecodeFixedBytes failed: %v", err)
		}

		expected := []byte{0x01, 0x02, 0x03}
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("UnexpectedEOF", func(t *testing.T) {
		t.Run("DecodeUint32", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeUint32()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeUint64", func(t *testing.T) {
			data := []byte{0x01, 0x02, 0x03, 0x04}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeUint64()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeBool", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBool()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeBytes_Length", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBytes()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeBytes_Data", func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x08, 0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBytes()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeString_Length", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeString()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeString_Data", func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x08, 0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeString()
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})

		t.Run("DecodeFixedBytes", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeFixedBytes(8)
			if err != ErrUnexpectedEOF {
				t.Errorf("Expected ErrUnexpectedEOF, got %v", err)
			}
		})
	})

	t.Run("InvalidData", func(t *testing.T) {
		// Create data with length that exceeds MaxInt32
		data := make([]byte, 8)
		data[0] = 0xFF // Set length to > MaxInt32
		data[1] = 0xFF
		data[2] = 0xFF
		data[3] = 0xFF
		decoder := NewDecoder(data)

		_, err := decoder.DecodeBytes()
		if err != ErrInvalidData {
			t.Errorf("Expected ErrInvalidData, got %v", err)
		}
	})
}

func TestRoundTrip(t *testing.T) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	// Encode various data types
	values := []any{
		uint32(0x12345678),
		uint64(0x123456789ABCDEF0),
		int32(-12345),
		int64(-123456789),
		true,
		false,
		"hello world",
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	for _, value := range values {
		switch v := value.(type) {
		case uint32:
			encoder.EncodeUint32(v)
		case uint64:
			encoder.EncodeUint64(v)
		case int32:
			encoder.EncodeInt32(v)
		case int64:
			encoder.EncodeInt64(v)
		case bool:
			encoder.EncodeBool(v)
		case string:
			encoder.EncodeString(v)
		case []byte:
			encoder.EncodeBytes(v)
		}
	}

	// Decode and verify
	decoder := NewDecoder(encoder.Bytes())

	for i, expected := range values {
		switch exp := expected.(type) {
		case uint32:
			result, err := decoder.DecodeUint32()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case uint64:
			result, err := decoder.DecodeUint64()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case int32:
			result, err := decoder.DecodeInt32()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case int64:
			result, err := decoder.DecodeInt64()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case bool:
			result, err := decoder.DecodeBool()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case string:
			result, err := decoder.DecodeString()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if result != exp {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		case []byte:
			result, err := decoder.DecodeBytes()
			if err != nil {
				t.Fatalf("Value %d decode failed: %v", i, err)
			}
			if !bytes.Equal(result, exp) {
				t.Errorf("Value %d: expected %v, got %v", i, exp, result)
			}
		}
	}
}

func TestGetSlice(t *testing.T) {
	// Test data: two uint32 values
	data := []byte{0x00, 0x00, 0x12, 0x34, 0x00, 0x00, 0x56, 0x78}
	decoder := NewDecoder(data)

	t.Run("GetSlice with valid range", func(t *testing.T) {
		// Extract first 4 bytes
		slice := decoder.GetSlice(0, 4)
		expected := []byte{0x00, 0x00, 0x12, 0x34}

		if !bytes.Equal(slice, expected) {
			t.Errorf("Expected %v, got %v", expected, slice)
		}

		// Verify zero-copy (same underlying array)
		if len(slice) > 0 && &slice[0] != &data[0] {
			t.Error("GetSlice should return zero-copy slice")
		}
	})

	t.Run("GetSlice after decoding", func(t *testing.T) {
		decoder.Reset(data)

		// Decode first value to advance position
		val1, err := decoder.DecodeUint32()
		if err != nil {
			t.Fatalf("DecodeUint32 failed: %v", err)
		}
		if val1 != 0x1234 {
			t.Errorf("Expected 0x1234, got 0x%x", val1)
		}

		// Get slice from current position to end
		pos := decoder.Position()
		slice := decoder.GetSlice(pos, len(data))
		expected := []byte{0x00, 0x00, 0x56, 0x78}

		if !bytes.Equal(slice, expected) {
			t.Errorf("Expected %v, got %v", expected, slice)
		}
	})

	t.Run("GetSlice with invalid ranges", func(t *testing.T) {
		decoder.Reset(data)

		// Test negative start
		slice := decoder.GetSlice(-1, 4)
		if slice != nil {
			t.Error("Expected nil for negative start")
		}

		// Test end beyond buffer
		slice = decoder.GetSlice(0, len(data)+1)
		if slice != nil {
			t.Error("Expected nil for end beyond buffer")
		}

		// Test start > end
		slice = decoder.GetSlice(4, 2)
		if slice != nil {
			t.Error("Expected nil for start > end")
		}
	})
}

func TestDecoderMethods(t *testing.T) {
	data := []byte{0x00, 0x00, 0x12, 0x34, 0x00, 0x00, 0x56, 0x78}
	decoder := NewDecoder(data)

	t.Run("Position and Remaining", func(t *testing.T) {
		if decoder.Position() != 0 {
			t.Errorf("Expected position 0, got %d", decoder.Position())
		}

		if decoder.Remaining() != len(data) {
			t.Errorf("Expected remaining %d, got %d", len(data), decoder.Remaining())
		}

		// Decode one uint32
		_, err := decoder.DecodeUint32()
		if err != nil {
			t.Fatalf("DecodeUint32 failed: %v", err)
		}

		if decoder.Position() != 4 {
			t.Errorf("Expected position 4, got %d", decoder.Position())
		}

		if decoder.Remaining() != len(data)-4 {
			t.Errorf("Expected remaining %d, got %d", len(data)-4, decoder.Remaining())
		}
	})

	t.Run("Reset", func(t *testing.T) {
		newData := []byte{0x11, 0x22, 0x33, 0x44}
		decoder.Reset(newData)

		if decoder.Position() != 0 {
			t.Errorf("Expected position 0 after reset, got %d", decoder.Position())
		}

		if decoder.Remaining() != len(newData) {
			t.Errorf("Expected remaining %d after reset, got %d", len(newData), decoder.Remaining())
		}

		val, err := decoder.DecodeUint32()
		if err != nil {
			t.Fatalf("DecodeUint32 failed: %v", err)
		}

		if val != 0x11223344 {
			t.Errorf("Expected 0x11223344, got 0x%x", val)
		}
	})
}

func TestEncoderMethods(t *testing.T) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	t.Run("Len and Bytes", func(t *testing.T) {
		if encoder.Len() != 0 {
			t.Errorf("Expected length 0, got %d", encoder.Len())
		}

		if len(encoder.Bytes()) != 0 {
			t.Errorf("Expected bytes length 0, got %d", len(encoder.Bytes()))
		}

		// Encode one uint32
		err := encoder.EncodeUint32(0x12345678)
		if err != nil {
			t.Fatalf("EncodeUint32 failed: %v", err)
		}

		if encoder.Len() != 4 {
			t.Errorf("Expected length 4, got %d", encoder.Len())
		}

		if len(encoder.Bytes()) != 4 {
			t.Errorf("Expected bytes length 4, got %d", len(encoder.Bytes()))
		}
	})

	t.Run("Reset", func(t *testing.T) {
		newBuf := make([]byte, 512)
		encoder.Reset(newBuf)

		if encoder.Len() != 0 {
			t.Errorf("Expected length 0 after reset, got %d", encoder.Len())
		}

		err := encoder.EncodeUint32(0x87654321)
		if err != nil {
			t.Fatalf("EncodeUint32 failed: %v", err)
		}

		expected := []byte{0x87, 0x65, 0x43, 0x21}
		if !bytes.Equal(encoder.Bytes(), expected) {
			t.Errorf("Expected %v, got %v", expected, encoder.Bytes())
		}
	})
}

func TestPadding(t *testing.T) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	testCases := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "No padding needed",
			input:    []byte{0x01, 0x02, 0x03, 0x04},
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "One byte padding",
			input:    []byte{0x01, 0x02, 0x03},
			expected: []byte{0x01, 0x02, 0x03, 0x00},
		},
		{
			name:     "Two bytes padding",
			input:    []byte{0x01, 0x02},
			expected: []byte{0x01, 0x02, 0x00, 0x00},
		},
		{
			name:     "Three bytes padding",
			input:    []byte{0x01},
			expected: []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:     "Empty input",
			input:    []byte{},
			expected: []byte{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoder.Reset(buf)
			err := encoder.EncodeFixedBytes(tc.input)
			if err != nil {
				t.Fatalf("EncodeFixedBytes failed: %v", err)
			}

			result := encoder.Bytes()
			if !bytes.Equal(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}

			// Test decoding
			decoder := NewDecoder(result)
			decoded, err := decoder.DecodeFixedBytes(len(tc.input))
			if err != nil {
				t.Fatalf("DecodeFixedBytes failed: %v", err)
			}

			if !bytes.Equal(decoded, tc.input) {
				t.Errorf("Round-trip failed: expected %v, got %v", tc.input, decoded)
			}
		})
	}
}

func BenchmarkEncoder(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	b.Run("EncodeUint32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeUint32(0x12345678)
		}
	})

	b.Run("EncodeString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeString("hello world")
		}
	})

	b.Run("EncodeBytes", func(b *testing.B) {
		data := make([]byte, 100)
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeBytes(data)
		}
	})
}

func BenchmarkDecoder(b *testing.B) {
	// Pre-encoded data
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)
	encoder.EncodeUint32(0x12345678)
	encoder.EncodeString("hello world")
	encoder.EncodeBytes(make([]byte, 100))
	data := encoder.Bytes()

	b.Run("DecodeUint32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data)
			decoder.DecodeUint32()
		}
	})

	b.Run("DecodeString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data[4:]) // Skip the uint32
			decoder.DecodeString()
		}
	})

	b.Run("DecodeBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data[20:]) // Skip uint32 and string
			decoder.DecodeBytes()
		}
	})
}

func TestWriter(t *testing.T) {
	t.Run("WriteUint32", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		err := writer.WriteUint32(0x12345678)
		if err != nil {
			t.Fatalf("WriteUint32 failed: %v", err)
		}

		expected := []byte{0x12, 0x34, 0x56, 0x78}
		if !bytes.Equal(buf.Bytes(), expected) {
			t.Errorf("Expected %v, got %v", expected, buf.Bytes())
		}
	})

	t.Run("WriteBytes", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		testData := []byte("hello")
		err := writer.WriteBytes(testData)
		if err != nil {
			t.Fatalf("WriteBytes failed: %v", err)
		}

		// Expected: length (4 bytes) + data (5 bytes) + padding (3 bytes)
		expected := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		if !bytes.Equal(buf.Bytes(), expected) {
			t.Errorf("Expected %v, got %v", expected, buf.Bytes())
		}
	})

	t.Run("WriteBytes_NoPadding", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		testData := []byte("test") // 4 bytes, no padding needed
		err := writer.WriteBytes(testData)
		if err != nil {
			t.Fatalf("WriteBytes failed: %v", err)
		}

		// Expected: length (4 bytes) + data (4 bytes)
		expected := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		if !bytes.Equal(buf.Bytes(), expected) {
			t.Errorf("Expected %v, got %v", expected, buf.Bytes())
		}
	})

	t.Run("WriteBytes_ErrorPaths", func(t *testing.T) {
		t.Run("WriteUint32_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 0})
			err := writer.WriteBytes([]byte("test"))
			if err == nil {
				t.Error("Expected error from WriteUint32, got nil")
			}
		})

		t.Run("Write_Data_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 1})
			err := writer.WriteBytes([]byte("test"))
			if err == nil {
				t.Error("Expected error from Write data, got nil")
			}
		})

		t.Run("Write_Padding_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 2})
			err := writer.WriteBytes([]byte("hello")) // needs padding
			if err == nil {
				t.Error("Expected error from Write padding, got nil")
			}
		})
	})
}

func TestReader(t *testing.T) {
	t.Run("ReadUint32", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadUint32()
		if err != nil {
			t.Fatalf("ReadUint32 failed: %v", err)
		}

		expected := uint32(0x12345678)
		if result != expected {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("ReadBytes", func(t *testing.T) {
		// Data: length (4 bytes) + data (5 bytes) + padding (3 bytes)
		data := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes failed: %v", err)
		}

		expected := []byte("hello")
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("ReadBytes_NoPadding", func(t *testing.T) {
		// Data: length (4 bytes) + data (4 bytes)
		data := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadBytes()
		if err != nil {
			t.Fatalf("ReadBytes failed: %v", err)
		}

		expected := []byte("test")
		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("ReadBytes_InvalidLength", func(t *testing.T) {
		// Data with length > MaxInt32
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		reader := NewReader(bytes.NewReader(data))

		_, err := reader.ReadBytes()
		if err != ErrInvalidData {
			t.Errorf("Expected ErrInvalidData, got %v", err)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		t.Run("ReadUint32_Fails", func(t *testing.T) {
			reader := NewReader(&failingReader{failAfter: 0})
			_, err := reader.ReadUint32()
			if err == nil {
				t.Error("Expected error from ReadUint32, got nil")
			}
		})

		t.Run("ReadBytes_Length_Fails", func(t *testing.T) {
			reader := NewReader(&failingReader{failAfter: 0})
			_, err := reader.ReadBytes()
			if err == nil {
				t.Error("Expected error from ReadBytes length, got nil")
			}
		})

		t.Run("ReadBytes_Data_Fails", func(t *testing.T) {
			// Valid length but read fails on data
			data := []byte{0x00, 0x00, 0x00, 0x04}
			reader := NewReader(&failingReader{
				data:      data,
				failAfter: 1, // Fail after reading length
			})
			_, err := reader.ReadBytes()
			if err == nil {
				t.Error("Expected error from ReadBytes data, got nil")
			}
		})
	})
}
