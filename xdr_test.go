package xdr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err, "EncodeUint32 failed")

		expected := []byte{0x12, 0x34, 0x56, 0x78}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeUint64", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeUint64(0x123456789ABCDEF0)
		require.NoError(t, err, "EncodeUint64 failed")

		expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeInt32", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeInt32(-1)
		require.NoError(t, err, "EncodeInt32 failed")

		expected := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeInt64", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeInt64(-1)
		require.NoError(t, err, "EncodeInt64 failed")

		expected := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeBool", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeBool(true)
		require.NoError(t, err, "EncodeBool failed")

		expected := []byte{0x00, 0x00, 0x00, 0x01}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)

		encoder.Reset(buf)
		err = encoder.EncodeBool(false)
		require.NoError(t, err, "EncodeBool failed")

		expected = []byte{0x00, 0x00, 0x00, 0x00}
		result = encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeString", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeString("test")
		require.NoError(t, err, "EncodeString failed")

		// Length (4) + "test" + padding (0)
		expected := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeStringWithPadding", func(t *testing.T) {
		encoder.Reset(buf)
		err := encoder.EncodeString("hello")
		require.NoError(t, err, "EncodeString failed")

		// Length (5) + "hello" + padding (3 bytes to align to 4)
		expected := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeBytes", func(t *testing.T) {
		encoder.Reset(buf)
		data := []byte{0x01, 0x02, 0x03}
		err := encoder.EncodeBytes(data)
		require.NoError(t, err, "EncodeBytes failed")

		// Length (3) + data + padding (1 byte)
		expected := []byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03, 0x00}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("EncodeFixedBytes", func(t *testing.T) {
		encoder.Reset(buf)
		data := []byte{0x01, 0x02, 0x03}
		err := encoder.EncodeFixedBytes(data)
		require.NoError(t, err, "EncodeFixedBytes failed")

		// Data + padding (1 byte) - no length prefix
		expected := []byte{0x01, 0x02, 0x03, 0x00}
		result := encoder.Bytes()
		assert.Equal(t, expected, result)
	})

	t.Run("BufferTooSmall", func(t *testing.T) {
		t.Run("EncodeUint32", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeUint32(0x12345678)
			assert.ErrorIs(t, err, ErrBufferTooSmall, "Expected ErrBufferTooSmall")
		})

		t.Run("EncodeUint64", func(t *testing.T) {
			smallBuf := make([]byte, 4)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeUint64(0x123456789ABCDEF0)
			require.ErrorIs(t, err, ErrBufferTooSmall, "Expected ErrBufferTooSmall")
		})

		t.Run("EncodeBytes", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeBytes([]byte("test"))
			require.ErrorIs(t, err, ErrBufferTooSmall, "Expected ErrBufferTooSmall")
		})

		t.Run("EncodeString", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeString("test")
			require.ErrorIs(t, err, ErrBufferTooSmall, "Expected ErrBufferTooSmall")
		})

		t.Run("EncodeFixedBytes", func(t *testing.T) {
			smallBuf := make([]byte, 2)
			encoder := NewEncoder(smallBuf)
			err := encoder.EncodeFixedBytes([]byte("test"))
			require.ErrorIs(t, err, ErrBufferTooSmall, "Expected ErrBufferTooSmall")
		})
	})
}

func TestDecoder(t *testing.T) {
	t.Run("DecodeUint32", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeUint32()
		require.NoError(t, err, "DecodeUint32 failed")

		expected := uint32(0x12345678)
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeUint64", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeUint64()
		require.NoError(t, err, "DecodeUint64 failed")

		expected := uint64(0x123456789ABCDEF0)
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeInt32", func(t *testing.T) {
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeInt32()
		require.NoError(t, err, "DecodeInt32 failed")

		expected := int32(-1)
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeInt64", func(t *testing.T) {
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeInt64()
		require.NoError(t, err, "DecodeInt64 failed")

		expected := int64(-1)
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeBool", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x01}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeBool()
		require.NoError(t, err, "DecodeBool failed")

		assert.True(t, result)

		decoder.Reset([]byte{0x00, 0x00, 0x00, 0x00})
		result, err = decoder.DecodeBool()
		require.NoError(t, err, "DecodeBool failed")

		assert.False(t, result)
	})

	t.Run("DecodeString", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeString()
		require.NoError(t, err, "DecodeString failed")

		expected := "test"
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeStringWithPadding", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeString()
		require.NoError(t, err, "DecodeString failed")

		expected := "hello"
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeBytes", func(t *testing.T) {
		data := []byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeBytes()
		require.NoError(t, err, "DecodeBytes failed")

		expected := []byte{0x01, 0x02, 0x03}
		assert.Equal(t, expected, result)
	})

	t.Run("DecodeFixedBytes", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x00}
		decoder := NewDecoder(data)

		result, err := decoder.DecodeFixedBytes(3)
		require.NoError(t, err, "DecodeFixedBytes failed")

		expected := []byte{0x01, 0x02, 0x03}
		assert.Equal(t, expected, result)
	})

	t.Run("UnexpectedEOF", func(t *testing.T) {
		t.Run("DecodeUint32", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeUint32()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeUint64", func(t *testing.T) {
			data := []byte{0x01, 0x02, 0x03, 0x04}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeUint64()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeBool", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBool()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeBytes_Length", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBytes()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeBytes_Data", func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x08, 0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeBytes()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeString_Length", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeString()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeString_Data", func(t *testing.T) {
			data := []byte{0x00, 0x00, 0x00, 0x08, 0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeString()
			require.ErrorIs(t, err, ErrUnexpectedEOF)
		})

		t.Run("DecodeFixedBytes", func(t *testing.T) {
			data := []byte{0x01, 0x02}
			decoder := NewDecoder(data)

			_, err := decoder.DecodeFixedBytes(8)
			require.ErrorIs(t, err, ErrUnexpectedEOF)
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
		require.ErrorIs(t, err, ErrInvalidData)
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
			_ = encoder.EncodeUint32(v)
		case uint64:
			_ = encoder.EncodeUint64(v)
		case int32:
			_ = encoder.EncodeInt32(v)
		case int64:
			_ = encoder.EncodeInt64(v)
		case bool:
			_ = encoder.EncodeBool(v)
		case string:
			_ = encoder.EncodeString(v)
		case []byte:
			_ = encoder.EncodeBytes(v)
		}
	}

	// Decode and verify
	decoder := NewDecoder(encoder.Bytes())

	for i, expected := range values {
		switch exp := expected.(type) {
		case uint32:
			result, err := decoder.DecodeUint32()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case uint64:
			result, err := decoder.DecodeUint64()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case int32:
			result, err := decoder.DecodeInt32()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case int64:
			result, err := decoder.DecodeInt64()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case bool:
			result, err := decoder.DecodeBool()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case string:
			result, err := decoder.DecodeString()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
		case []byte:
			result, err := decoder.DecodeBytes()
			require.NoError(t, err, "Value %d decode failed", i)
			assert.Equal(t, exp, result)
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

		assert.Equal(t, expected, slice)

		// Verify zero-copy (same underlying array)
		assert.True(t, len(slice) > 0 && &slice[0] == &data[0], "GetSlice should return zero-copy slice")
	})

	t.Run("GetSlice after decoding", func(t *testing.T) {
		decoder.Reset(data)

		// Decode first value to advance position
		val1, err := decoder.DecodeUint32()
		require.NoError(t, err, "DecodeUint32 failed")
		assert.Equal(t, uint32(0x1234), val1)

		// Get slice from current position to end
		pos := decoder.Position()
		slice := decoder.GetSlice(pos, len(data))
		expected := []byte{0x00, 0x00, 0x56, 0x78}

		assert.Equal(t, expected, slice)
	})

	t.Run("GetSlice with invalid ranges", func(t *testing.T) {
		decoder.Reset(data)

		// Test negative start
		slice := decoder.GetSlice(-1, 4)
		assert.Nil(t, slice, "Expected nil for negative start")

		// Test end beyond buffer
		slice = decoder.GetSlice(0, len(data)+1)
		assert.Nil(t, slice, "Expected nil for end beyond buffer")

		// Test start > end
		slice = decoder.GetSlice(4, 2)
		assert.Nil(t, slice, "Expected nil for start > end")
	})
}

func TestDecoderMethods(t *testing.T) {
	data := []byte{0x00, 0x00, 0x12, 0x34, 0x00, 0x00, 0x56, 0x78}
	decoder := NewDecoder(data)

	t.Run("Position and Remaining", func(t *testing.T) {
		assert.Equal(t, 0, decoder.Position(), "Expected position 0")

		assert.Equal(t, len(data), decoder.Remaining(), "Expected remaining %d", len(data))

		// Decode one uint32
		_, err := decoder.DecodeUint32()
		require.NoError(t, err, "DecodeUint32 failed")

		assert.Equal(t, 4, decoder.Position(), "Expected position 4")

		assert.Equal(t, len(data)-4, decoder.Remaining(), "Expected remaining %d", len(data)-4)
	})

	t.Run("Reset", func(t *testing.T) {
		newData := []byte{0x11, 0x22, 0x33, 0x44}
		decoder.Reset(newData)

		assert.Equal(t, 0, decoder.Position(), "Expected position 0 after reset")
		assert.Equal(t, len(newData), decoder.Remaining(), "Expected remaining %d after reset", len(newData))

		val, err := decoder.DecodeUint32()
		require.NoError(t, err, "DecodeUint32 failed")

		assert.Equal(t, uint32(0x11223344), val)
	})
}

func TestEncoderMethods(t *testing.T) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	t.Run("Len and Bytes", func(t *testing.T) {
		assert.Equal(t, 0, encoder.Len(), "Expected length 0")

		assert.Empty(t, encoder.Bytes(), "Expected bytes length 0")

		// Encode one uint32
		err := encoder.EncodeUint32(0x12345678)
		require.NoError(t, err, "EncodeUint32 failed")

		assert.Equal(t, 4, encoder.Len(), "Expected length 4")

		assert.Len(t, encoder.Bytes(), 4, "Expected bytes length 4")
	})

	t.Run("Reset", func(t *testing.T) {
		newBuf := make([]byte, 512)
		encoder.Reset(newBuf)

		assert.Equal(t, 0, encoder.Len(), "Expected length 0 after reset")

		err := encoder.EncodeUint32(0x87654321)
		require.NoError(t, err, "EncodeUint32 failed")

		expected := []byte{0x87, 0x65, 0x43, 0x21}
		assert.Equal(t, expected, encoder.Bytes())
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
			require.NoError(t, err, "EncodeFixedBytes failed")

			result := encoder.Bytes()
			assert.Equal(t, tc.expected, result)

			// Test decoding
			decoder := NewDecoder(result)
			decoded, err := decoder.DecodeFixedBytes(len(tc.input))
			require.NoError(t, err, "DecodeFixedBytes failed")
			assert.Equal(t, tc.input, decoded)
		})
	}
}

func BenchmarkEncoder(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)

	b.Run("EncodeUint32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			_ = encoder.EncodeUint32(0x12345678)
		}
	})

	b.Run("EncodeString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			_ = encoder.EncodeString("hello world")
		}
	})

	b.Run("EncodeBytes", func(b *testing.B) {
		data := make([]byte, 100)
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			_ = encoder.EncodeBytes(data)
		}
	})
}

func BenchmarkDecoder(b *testing.B) {
	// Pre-encoded data
	buf := make([]byte, 1024)
	encoder := NewEncoder(buf)
	_ = encoder.EncodeUint32(0x12345678)
	_ = encoder.EncodeString("hello world")
	_ = encoder.EncodeBytes(make([]byte, 100))
	data := encoder.Bytes()

	b.Run("DecodeUint32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data)
			_, _ = decoder.DecodeUint32()
		}
	})

	b.Run("DecodeString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data[4:]) // Skip the uint32
			_, _ = decoder.DecodeString()
		}
	})

	b.Run("DecodeBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder(data[20:]) // Skip uint32 and string
			_, _ = decoder.DecodeBytes()
		}
	})
}

func TestWriter(t *testing.T) {
	t.Run("WriteUint32", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		err := writer.WriteUint32(0x12345678)
		require.NoError(t, err, "WriteUint32 failed")

		expected := []byte{0x12, 0x34, 0x56, 0x78}
		assert.Equal(t, expected, buf.Bytes())
	})

	t.Run("WriteBytes", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		testData := []byte("hello")
		err := writer.WriteBytes(testData)
		require.NoError(t, err, "WriteBytes failed")

		// Expected: length (4 bytes) + data (5 bytes) + padding (3 bytes)
		expected := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		assert.Equal(t, expected, buf.Bytes())
	})

	t.Run("WriteBytes_NoPadding", func(t *testing.T) {
		var buf bytes.Buffer
		writer := NewWriter(&buf)

		testData := []byte("test") // 4 bytes, no padding needed
		err := writer.WriteBytes(testData)
		require.NoError(t, err, "WriteBytes failed")

		// Expected: length (4 bytes) + data (4 bytes)
		expected := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		assert.Equal(t, expected, buf.Bytes())
	})

	t.Run("WriteBytes_ErrorPaths", func(t *testing.T) {
		t.Run("WriteUint32_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 0})
			err := writer.WriteBytes([]byte("test"))
			require.Error(t, err, "Expected error from WriteUint32, got nil")
		})

		t.Run("Write_Data_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 1})
			err := writer.WriteBytes([]byte("test"))
			require.Error(t, err, "Expected error from Write data, got nil")
		})

		t.Run("Write_Padding_Fails", func(t *testing.T) {
			writer := NewWriter(&failingWriter{failAfter: 2})
			err := writer.WriteBytes([]byte("hello")) // needs padding
			require.Error(t, err, "Expected error from Write padding, got nil")
		})
	})
}

func TestReader(t *testing.T) {
	t.Run("ReadUint32", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadUint32()
		require.NoError(t, err, "ReadUint32 failed")

		expected := uint32(0x12345678)
		assert.Equal(t, expected, result)
	})

	t.Run("ReadBytes", func(t *testing.T) {
		// Data: length (4 bytes) + data (5 bytes) + padding (3 bytes)
		data := []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadBytes()
		require.NoError(t, err, "ReadBytes failed")

		expected := []byte("hello")
		assert.Equal(t, expected, result)
	})

	t.Run("ReadBytes_NoPadding", func(t *testing.T) {
		// Data: length (4 bytes) + data (4 bytes)
		data := []byte{0x00, 0x00, 0x00, 0x04, 't', 'e', 's', 't'}
		reader := NewReader(bytes.NewReader(data))

		result, err := reader.ReadBytes()
		require.NoError(t, err, "ReadBytes failed")

		expected := []byte("test")
		assert.Equal(t, expected, result)
	})

	t.Run("ReadBytes_InvalidLength", func(t *testing.T) {
		// Data with length > MaxInt32
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		reader := NewReader(bytes.NewReader(data))

		_, err := reader.ReadBytes()
		require.ErrorIs(t, err, ErrInvalidData, "Expected ErrInvalidData, got %v", err)
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		t.Run("ReadUint32_Fails", func(t *testing.T) {
			reader := NewReader(&failingReader{failAfter: 0})
			_, err := reader.ReadUint32()
			require.Error(t, err, "Expected error from ReadUint32, got nil")
		})

		t.Run("ReadBytes_Length_Fails", func(t *testing.T) {
			reader := NewReader(&failingReader{failAfter: 0})
			_, err := reader.ReadBytes()
			require.Error(t, err, "Expected error from ReadBytes length, got nil")
		})

		t.Run("ReadBytes_Data_Fails", func(t *testing.T) {
			// Valid length but read fails on data
			data := []byte{0x00, 0x00, 0x00, 0x04}
			reader := NewReader(&failingReader{
				data:      data,
				failAfter: 1, // Fail after reading length
			})
			_, err := reader.ReadBytes()
			require.Error(t, err, "Expected error from ReadBytes data, got nil")
		})
	})
}

func TestDecodeFixedBytesInto(t *testing.T) {
	// Test data with padding
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x00, 0x00, 0x00} // 5 bytes + 3 padding
	decoder := NewDecoder(testData)

	// Test decoding into a fixed-size array
	var result [5]byte
	err := decoder.DecodeFixedBytesInto(result[:])
	require.NoError(t, err, "DecodeFixedBytesInto failed")

	expected := [5]byte{0x01, 0x02, 0x03, 0x04, 0x05}
	assert.Equal(t, expected, result)

	// Test that decoder position advanced correctly (data + padding)
	assert.Equal(t, 8, decoder.Position(), "Expected position 8")
}

func TestDecodeFixedBytesIntoErrors(t *testing.T) {
	// Test with insufficient data
	shortData := []byte{0x01, 0x02, 0x03} // Only 3 bytes
	decoder := NewDecoder(shortData)

	var result [5]byte
	err := decoder.DecodeFixedBytesInto(result[:])
	require.ErrorIs(t, err, ErrUnexpectedEOF, "Expected ErrUnexpectedEOF, got %v", err)
}

func TestDecodeFixedBytesIntoZeroAllocation(t *testing.T) {
	// Test that DecodeFixedBytesInto doesn't allocate
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x00, 0x00, 0x00, 0x00} // 8 bytes + 0 padding
	decoder := NewDecoder(testData)

	var result [8]byte
	err := decoder.DecodeFixedBytesInto(result[:])
	require.NoError(t, err, "DecodeFixedBytesInto failed")

	expected := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	assert.Equal(t, expected, result)
}

func TestFixedBytesCompatibility(t *testing.T) {
	// Test that both DecodeFixedBytes and DecodeFixedBytesInto produce identical results
	testCases := []struct {
		name string
		data []byte
		size int
	}{
		{
			name: "1 byte with 3 padding",
			data: []byte{0xAA, 0x00, 0x00, 0x00},
			size: 1,
		},
		{
			name: "2 bytes with 2 padding",
			data: []byte{0xAA, 0xBB, 0x00, 0x00},
			size: 2,
		},
		{
			name: "3 bytes with 1 padding",
			data: []byte{0xAA, 0xBB, 0xCC, 0x00},
			size: 3,
		},
		{
			name: "4 bytes with 0 padding",
			data: []byte{0xAA, 0xBB, 0xCC, 0xDD},
			size: 4,
		},
		{
			name: "8 bytes with 0 padding",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			size: 8,
		},
		{
			name: "16 bytes with 0 padding",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			size: 16,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test DecodeFixedBytes (allocating path)
			decoder1 := NewDecoder(tc.data)
			result1, err := decoder1.DecodeFixedBytes(tc.size)
			require.NoError(t, err, "DecodeFixedBytes failed")

			// Test DecodeFixedBytesInto (zero-allocation path)
			decoder2 := NewDecoder(tc.data)
			result2 := make([]byte, tc.size)
			err = decoder2.DecodeFixedBytesInto(result2)
			require.NoError(t, err, "DecodeFixedBytesInto failed")

			// Results should be identical
			assert.Equal(t, result1, result2)

			// Both decoders should advance by same amount
			assert.Equal(t, decoder1.Position(), decoder2.Position())

			// Verify expected data (excluding padding)
			expected := tc.data[:tc.size]
			assert.Equal(t, expected, result1)
		})
	}
}

func TestFixedBytesRoundTripCompatibility(t *testing.T) {
	// Test that encoding followed by both decode methods produces identical results
	testSizes := []int{1, 2, 3, 4, 5, 8, 12, 16, 32}

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Create test data
			original := make([]byte, size)
			for i := range original {
				original[i] = byte(i + 1) // 1, 2, 3, ...
			}

			// Encode
			buf := make([]byte, 1024)
			encoder := NewEncoder(buf)
			err := encoder.EncodeFixedBytes(original)
			require.NoError(t, err, "EncodeFixedBytes failed")
			encoded := encoder.Bytes()

			// Decode using DecodeFixedBytes
			decoder1 := NewDecoder(encoded)
			result1, err := decoder1.DecodeFixedBytes(size)
			require.NoError(t, err, "DecodeFixedBytes failed")

			// Decode using DecodeFixedBytesInto
			decoder2 := NewDecoder(encoded)
			result2 := make([]byte, size)
			err = decoder2.DecodeFixedBytesInto(result2)
			require.NoError(t, err, "DecodeFixedBytesInto failed")

			// All should match original
			assert.Equal(t, original, result1)
			assert.Equal(t, original, result2)
			assert.Equal(t, result1, result2)
		})
	}
}

func TestFixedBytesErrorHandling(t *testing.T) {
	// Test that both methods handle errors consistently
	t.Run("InsufficientData", func(t *testing.T) {
		shortData := []byte{0x01, 0x02, 0x03} // Only 3 bytes, need 8

		decoder1 := NewDecoder(shortData)
		_, err1 := decoder1.DecodeFixedBytes(8)

		decoder2 := NewDecoder(shortData)
		result2 := make([]byte, 8)
		err2 := decoder2.DecodeFixedBytesInto(result2)

		// Both should return ErrUnexpectedEOF
		require.ErrorIs(t, err1, ErrUnexpectedEOF, "DecodeFixedBytes: expected ErrUnexpectedEOF, got %v", err1)
		require.ErrorIs(t, err2, ErrUnexpectedEOF, "DecodeFixedBytesInto: expected ErrUnexpectedEOF, got %v", err2)
	})

	t.Run("ZeroLength", func(t *testing.T) {
		data := []byte{}

		decoder1 := NewDecoder(data)
		result1, err1 := decoder1.DecodeFixedBytes(0)

		decoder2 := NewDecoder(data)
		result2 := make([]byte, 0)
		err2 := decoder2.DecodeFixedBytesInto(result2)

		// Both should succeed with empty result
		require.NoError(t, err1, "DecodeFixedBytes: expected no error, got %v", err1)
		require.NoError(t, err2, "DecodeFixedBytesInto: expected no error, got %v", err2)
		assert.Empty(t, result1, "DecodeFixedBytes: expected empty result")
		assert.Empty(t, result2, "DecodeFixedBytesInto: expected empty result")
	})
}
