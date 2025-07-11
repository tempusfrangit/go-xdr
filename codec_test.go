package xdr

import (
	"testing"
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
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Marshal returned empty data")
	}

	// Test Unmarshal
	var decoded TestType
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify round-trip
	if original.ID != decoded.ID {
		t.Errorf("ID mismatch: expected %d, got %d", original.ID, decoded.ID)
	}
	if original.Name != decoded.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, decoded.Name)
	}
}

func TestCodecMarshalError(t *testing.T) {
	// Test with a type that will cause encoding error
	badType := &TestType{
		Name: string(make([]byte, 1<<20)), // Very large string to cause buffer overflow
	}

	_, err := Marshal(badType)
	if err == nil {
		t.Error("Expected error for oversized data, got nil")
	}
	if err.Error() != "XDR encoding failed: buffer too small" {
		t.Errorf("Expected 'XDR encoding failed: buffer too small', got %s", err.Error())
	}
}

func TestCodecUnmarshalError(t *testing.T) {
	// Test with malformed data
	badData := []byte{0x01, 0x02} // Too short

	var testType TestType
	err := Unmarshal(badData, &testType)
	if err == nil {
		t.Error("Expected error for malformed data, got nil")
	}
	if err.Error() != "XDR decoding failed: unexpected end of data" {
		t.Errorf("Expected 'XDR decoding failed: unexpected end of data', got %s", err.Error())
	}
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
	if err != nil {
		t.Fatalf("MarshalRaw failed: %v", err)
	}

	if len(wrappedData) != len(originalData) {
		t.Errorf("Length mismatch: expected %d, got %d", len(originalData), len(wrappedData))
	}

	// Verify contents are identical
	for i := range originalData {
		if originalData[i] != wrappedData[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, originalData[i], wrappedData[i])
		}
	}

	// Verify it's a copy, not the same slice
	if len(originalData) > 0 && len(wrappedData) > 0 {
		if &originalData[0] == &wrappedData[0] {
			t.Error("MarshalRaw should return a copy, not the same slice")
		}
	}
}

func TestMarshalRawNil(t *testing.T) {
	_, err := MarshalRaw(nil)
	if err == nil {
		t.Error("Expected error for nil data, got nil")
	}
	if err.Error() != "data cannot be nil" {
		t.Errorf("Expected 'data cannot be nil', got %s", err.Error())
	}
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
		if err != nil {
			t.Fatalf("EncodeUint32 failed: %v", err)
		}
	}
	if mask&(1<<1) != 0 {
		err = enc.EncodeUint64(200)
		if err != nil {
			t.Fatalf("EncodeUint64 failed: %v", err)
		}
	}

	sparseData := make([]byte, len(enc.Bytes()))
	copy(sparseData, enc.Bytes())

	// Use MarshalRaw for sparse data
	result, err := MarshalRaw(sparseData)
	if err != nil {
		t.Fatalf("MarshalRaw failed: %v", err)
	}

	// Decode and verify
	dec := NewDecoder(result)

	decodedMask, err := dec.DecodeUint64()
	if err != nil {
		t.Fatalf("DecodeUint64 failed: %v", err)
	}
	if decodedMask != mask {
		t.Errorf("Mask mismatch: expected %d, got %d", mask, decodedMask)
	}

	decodedValue1, err := dec.DecodeUint32()
	if err != nil {
		t.Fatalf("DecodeUint32 failed: %v", err)
	}
	if decodedValue1 != 100 {
		t.Errorf("Value1 mismatch: expected 100, got %d", decodedValue1)
	}

	decodedValue2, err := dec.DecodeUint64()
	if err != nil {
		t.Fatalf("DecodeUint64 failed: %v", err)
	}
	if decodedValue2 != 200 {
		t.Errorf("Value2 mismatch: expected 200, got %d", decodedValue2)
	}
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
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Test Unmarshal
	var decoded NestedStruct
	err = Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify round-trip
	if original.Inner.ID != decoded.Inner.ID {
		t.Errorf("Inner.ID mismatch: expected %d, got %d", original.Inner.ID, decoded.Inner.ID)
	}
	if original.Inner.Name != decoded.Inner.Name {
		t.Errorf("Inner.Name mismatch: expected %s, got %s", original.Inner.Name, decoded.Inner.Name)
	}
	if original.Count != decoded.Count {
		t.Errorf("Count mismatch: expected %d, got %d", original.Count, decoded.Count)
	}
}

func BenchmarkCodec(b *testing.B) {
	testType := &TestType{
		ID:   12345,
		Name: "benchmark-test",
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Marshal(testType)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
		}
	})

	data, err := Marshal(testType)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}

	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var decoded TestType
			err := Unmarshal(data, &decoded)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
	})
}
