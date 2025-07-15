package xdr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestType implements the Codec interface for testing
type TestType struct {
	ID   uint32
	Name string
}

func (t *TestType) Encode(enc *Encoder) error {
	if err := enc.EncodeUint32(t.ID); err != nil {
		return err
	}
	if err := enc.EncodeString(t.Name); err != nil {
		return err
	}
	return nil
}

func (t *TestType) Decode(dec *Decoder) error {
	id, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	t.ID = id

	name, err := dec.DecodeString()
	if err != nil {
		return err
	}
	t.Name = name

	return nil
}

// Compile-time assertion that TestType implements Codec
var _ Codec = (*TestType)(nil)

func TestCodecInterface(t *testing.T) {
	original := &TestType{
		ID:   12345,
		Name: "test-codec",
	}

	// Test Marshal
	data, err := Marshal(original)
	require.NoError(t, err, "Marshal failed")
	assert.NotEmpty(t, data, "Marshal returned empty data")

	// Test Unmarshal
	var decoded TestType
	err = Unmarshal(data, &decoded)
	require.NoError(t, err, "Unmarshal failed")

	// Verify round-trip
	assert.Equal(t, original.ID, decoded.ID, "ID mismatch")
	assert.Equal(t, original.Name, decoded.Name, "Name mismatch")
}

func TestCodecMarshalError(t *testing.T) {
	// Test with a type that will cause encoding error
	badType := &TestType{
		Name: string(make([]byte, 1<<20)), // Very large string to cause buffer overflow
	}

	_, err := Marshal(badType)
	require.Error(t, err, "Expected error for oversized data")
	assert.Equal(t, "XDR encoding failed: buffer too small", err.Error(), "Unexpected error message")
}

func TestCodecUnmarshalError(t *testing.T) {
	// Test with malformed data
	badData := []byte{0x01, 0x02} // Too short

	var testType TestType
	err := Unmarshal(badData, &testType)
	require.Error(t, err, "Expected error for malformed data")
	assert.Equal(t, "XDR decoding failed: unexpected end of data", err.Error(), "Unexpected error message")
}

func TestMarshalRaw(t *testing.T) {
	// Create some test XDR data manually
	buf := make([]byte, 8)
	enc := NewEncoder(buf)
	err := enc.EncodeUint32(123)
	if err != nil {
		t.Fatalf("EncodeUint32 failed: %v", err)
	}

	originalData := enc.Bytes()

	// Test MarshalRaw
	wrappedData, err := MarshalRaw(originalData)
	require.NoError(t, err, "MarshalRaw failed")

	assert.Len(t, wrappedData, len(originalData), "Length mismatch")

	// Verify contents are identical
	assert.Equal(t, originalData, wrappedData, "Data contents should be identical")

	// Verify it's a copy, not the same slice
	if len(originalData) > 0 && len(wrappedData) > 0 {
		assert.NotSame(t, &originalData[0], &wrappedData[0], "MarshalRaw should return a copy, not the same slice")
	}
}

func TestMarshalRawNil(t *testing.T) {
	_, err := MarshalRaw(nil)
	require.Error(t, err, "Expected error for nil data")
	assert.Equal(t, "data cannot be nil", err.Error(), "Unexpected error message")
}

func TestMarshalRawSparseExample(t *testing.T) {
	// Simulate sparse encoding logic for exceptional cases
	mask := uint64((1 << 0) | (1 << 1)) // bits 0 and 1 set

	buf := make([]byte, 256)
	enc := NewEncoder(buf)

	// Encode mask first
	err := enc.EncodeUint64(mask)
	if err != nil {
		t.Fatalf("EncodeUint64 failed: %v", err)
	}

	// Encode values conditionally based on mask
	if mask&(1<<0) != 0 {
		err = enc.EncodeUint32(100)
		require.NoError(t, err, "EncodeUint32 failed")
	}
	if mask&(1<<1) != 0 {
		err = enc.EncodeUint64(200)
		require.NoError(t, err, "EncodeUint64 failed")
	}

	sparseData := make([]byte, len(enc.Bytes()))
	copy(sparseData, enc.Bytes())

	// Use MarshalRaw for sparse data
	result, err := MarshalRaw(sparseData)
	require.NoError(t, err, "MarshalRaw failed")

	// Decode and verify
	dec := NewDecoder(result)

	decodedMask, err := dec.DecodeUint64()
	require.NoError(t, err, "DecodeUint64 failed")
	assert.Equal(t, mask, decodedMask, "Mask mismatch")

	decodedValue1, err := dec.DecodeUint32()
	require.NoError(t, err, "DecodeUint32 failed")
	assert.Equal(t, uint32(100), decodedValue1, "Value1 mismatch")

	decodedValue2, err := dec.DecodeUint64()
	require.NoError(t, err, "DecodeUint64 failed")
	assert.Equal(t, uint64(200), decodedValue2, "Value2 mismatch")
}

// Test with nested structs
type NestedStruct struct {
	Inner TestType
	Count uint32
}

func (n *NestedStruct) Encode(enc *Encoder) error {
	if err := n.Inner.Encode(enc); err != nil {
		return err
	}
	if err := enc.EncodeUint32(n.Count); err != nil {
		return err
	}
	return nil
}

func (n *NestedStruct) Decode(dec *Decoder) error {
	if err := n.Inner.Decode(dec); err != nil {
		return err
	}
	count, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	n.Count = count
	return nil
}

var _ Codec = (*NestedStruct)(nil)

func TestNestedCodec(t *testing.T) {
	original := &NestedStruct{
		Inner: TestType{
			ID:   999,
			Name: "nested",
		},
		Count: 42,
	}

	// Test Marshal
	data, err := Marshal(original)
	require.NoError(t, err, "Marshal failed")

	// Test Unmarshal
	var decoded NestedStruct
	err = Unmarshal(data, &decoded)
	require.NoError(t, err, "Unmarshal failed")

	// Verify round-trip
	assert.Equal(t, original.Inner.ID, decoded.Inner.ID, "Inner.ID mismatch")
	assert.Equal(t, original.Inner.Name, decoded.Inner.Name, "Inner.Name mismatch")
	assert.Equal(t, original.Count, decoded.Count, "Count mismatch")
}

func BenchmarkCodec(b *testing.B) {
	testType := &TestType{
		ID:   12345,
		Name: "benchmark-test",
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Marshal(testType)
			require.NoError(b, err, "Marshal failed")
		}
	})

	data, err := Marshal(testType)
	require.NoError(b, err, "Marshal failed")

	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var decoded TestType
			err := Unmarshal(data, &decoded)
			require.NoError(b, err, "Unmarshal failed")
		}
	})
}
