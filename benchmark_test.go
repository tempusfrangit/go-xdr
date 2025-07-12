//go:build bench
// +build bench

package xdr_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/tempusfrangit/go-xdr"
)

// BenchmarkSuite provides comprehensive performance benchmarks for XDR operations

func BenchmarkEncodePrimitives(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	b.Run("Uint32", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeUint32(0x12345678)
		}
		b.SetBytes(4)
	})

	b.Run("Uint64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeUint64(0x123456789ABCDEF0)
		}
		b.SetBytes(8)
	})

	b.Run("Int32", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeInt32(-12345)
		}
		b.SetBytes(4)
	})

	b.Run("Int64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeInt64(-123456789)
		}
		b.SetBytes(8)
	})

	b.Run("Bool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeBool(true)
		}
		b.SetBytes(4)
	})
}

func BenchmarkDecodePrimitives(b *testing.B) {
	// Pre-encode test data
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)
	encoder.EncodeUint32(0x12345678)
	uint32Data := encoder.Bytes()

	encoder.Reset(buf)
	encoder.EncodeUint64(0x123456789ABCDEF0)
	uint64Data := encoder.Bytes()

	encoder.Reset(buf)
	encoder.EncodeInt32(-12345)
	int32Data := encoder.Bytes()

	encoder.Reset(buf)
	encoder.EncodeInt64(-123456789)
	int64Data := encoder.Bytes()

	encoder.Reset(buf)
	encoder.EncodeBool(true)
	boolData := encoder.Bytes()

	b.Run("Uint32", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(uint32Data)
			decoder.DecodeUint32()
		}
		b.SetBytes(4)
	})

	b.Run("Uint64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(uint64Data)
			decoder.DecodeUint64()
		}
		b.SetBytes(8)
	})

	b.Run("Int32", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(int32Data)
			decoder.DecodeInt32()
		}
		b.SetBytes(4)
	})

	b.Run("Int64", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(int64Data)
			decoder.DecodeInt64()
		}
		b.SetBytes(8)
	})

	b.Run("Bool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(boolData)
			decoder.DecodeBool()
		}
		b.SetBytes(4)
	})
}

func BenchmarkEncodeStrings(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	testStrings := []string{
		"",
		"x",
		"hello",
		"hello world",
		"this is a longer string for testing XDR encoding performance",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
	}

	for _, str := range testStrings {
		b.Run(str+"_len_"+string(rune(len(str))), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encoder.Reset(buf)
				encoder.EncodeString(str)
			}
			b.SetBytes(int64(len(str) + 4)) // String length + length prefix
		})
	}
}

func BenchmarkDecodeStrings(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	testStrings := []string{
		"",
		"x",
		"hello",
		"hello world",
		"this is a longer string for testing XDR decoding performance",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
	}

	for _, str := range testStrings {
		encoder.Reset(buf)
		encoder.EncodeString(str)
		data := make([]byte, len(encoder.Bytes()))
		copy(data, encoder.Bytes())

		b.Run(str+"_len_"+string(rune(len(str))), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoder := xdr.NewDecoder(data)
				decoder.DecodeString()
			}
			b.SetBytes(int64(len(str) + 4))
		})
	}
}

func BenchmarkEncodeBytes(b *testing.B) {
	buf := make([]byte, 64*1024) // 64KB buffer
	encoder := xdr.NewEncoder(buf)

	sizes := []int{0, 1, 4, 16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(string(rune(size))+"_bytes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encoder.Reset(buf)
				encoder.EncodeBytes(data)
			}
			b.SetBytes(int64(size + 4)) // Data length + length prefix
		})
	}
}

func BenchmarkDecodeBytes(b *testing.B) {
	buf := make([]byte, 64*1024) // 64KB buffer
	encoder := xdr.NewEncoder(buf)

	sizes := []int{0, 1, 4, 16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		encoder.Reset(buf)
		encoder.EncodeBytes(data)
		encodedData := make([]byte, len(encoder.Bytes()))
		copy(encodedData, encoder.Bytes())

		b.Run(string(rune(size))+"_bytes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoder := xdr.NewDecoder(encodedData)
				decoder.DecodeBytes()
			}
			b.SetBytes(int64(size + 4))
		})
	}
}

func BenchmarkEncodeFixedBytes(b *testing.B) {
	buf := make([]byte, 64*1024) // 64KB buffer
	encoder := xdr.NewEncoder(buf)

	sizes := []int{1, 4, 16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(string(rune(size))+"_bytes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encoder.Reset(buf)
				encoder.EncodeFixedBytes(data)
			}
			b.SetBytes(int64(size))
		})
	}
}

func BenchmarkDecodeFixedBytes(b *testing.B) {
	buf := make([]byte, 64*1024) // 64KB buffer
	encoder := xdr.NewEncoder(buf)

	sizes := []int{1, 4, 16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		encoder.Reset(buf)
		encoder.EncodeFixedBytes(data)
		encodedData := make([]byte, len(encoder.Bytes()))
		copy(encodedData, encoder.Bytes())

		b.Run(string(rune(size))+"_bytes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoder := xdr.NewDecoder(encodedData)
				decoder.DecodeFixedBytes(size)
			}
			b.SetBytes(int64(size))
		})
	}
}

func BenchmarkDecodeFixedBytesInto(b *testing.B) {
	buf := make([]byte, 64*1024) // 64KB buffer
	encoder := xdr.NewEncoder(buf)

	sizes := []int{1, 4, 16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		encoder.Reset(buf)
		encoder.EncodeFixedBytes(data)
		encodedData := make([]byte, len(encoder.Bytes()))
		copy(encodedData, encoder.Bytes())

		// Pre-allocate destination buffer once
		dst := make([]byte, size)

		b.Run(string(rune(size))+"_bytes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoder := xdr.NewDecoder(encodedData)
				decoder.DecodeFixedBytesInto(dst)
			}
			b.SetBytes(int64(size))
		})
	}
}

func BenchmarkDecodeFixedBytesComparison(b *testing.B) {
	// Direct comparison between allocation vs zero-allocation approaches
	// Using a realistic 16-byte hash size
	testData := make([]byte, 16)
	rand.Read(testData)

	buf := make([]byte, 64)
	encoder := xdr.NewEncoder(buf)
	encoder.EncodeFixedBytes(testData)
	encodedData := make([]byte, len(encoder.Bytes()))
	copy(encodedData, encoder.Bytes())

	b.Run("DecodeFixedBytes_allocating", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(encodedData)
			_, err := decoder.DecodeFixedBytes(16)
			if err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(16)
	})

	// Pre-allocate destination buffer for zero-allocation path
	dst := make([]byte, 16)
	b.Run("DecodeFixedBytesInto_zero_alloc", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(encodedData)
			err := decoder.DecodeFixedBytesInto(dst)
			if err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(16)
	})

	// Realistic scenario: decode into a struct field (simulating generated code)
	type TestStruct struct {
		Hash [16]byte
	}
	var testStruct TestStruct

	b.Run("DecodeFixedBytesInto_struct_field", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(encodedData)
			err := decoder.DecodeFixedBytesInto(testStruct.Hash[:])
			if err != nil {
				b.Fatal(err)
			}
		}
		b.SetBytes(16)
	})
}

func BenchmarkRoundTrip(b *testing.B) {
	buf := make([]byte, 1024)
	encoder := xdr.NewEncoder(buf)

	b.Run("Uint32", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeUint32(0x12345678)
			data := encoder.Bytes()
			decoder := xdr.NewDecoder(data)
			decoder.DecodeUint32()
		}
		b.SetBytes(4)
	})

	b.Run("String", func(b *testing.B) {
		testStr := "hello world"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeString(testStr)
			data := encoder.Bytes()
			decoder := xdr.NewDecoder(data)
			decoder.DecodeString()
		}
		b.SetBytes(int64(len(testStr) + 4))
	})

	b.Run("Bytes_1KB", func(b *testing.B) {
		testData := make([]byte, 1024)
		rand.Read(testData)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeBytes(testData)
			data := encoder.Bytes()
			decoder := xdr.NewDecoder(data)
			decoder.DecodeBytes()
		}
		b.SetBytes(1024 + 4)
	})
}

// BenchStruct implements Codec for benchmarking
type BenchStruct struct {
	ID    uint32
	Name  string
	Data  []byte
	Count uint64
}

func (bs *BenchStruct) Encode(enc *xdr.Encoder) error {
	if err := enc.EncodeUint32(bs.ID); err != nil {
		return err
	}
	if err := enc.EncodeString(bs.Name); err != nil {
		return err
	}
	if err := enc.EncodeBytes(bs.Data); err != nil {
		return err
	}
	if err := enc.EncodeUint64(bs.Count); err != nil {
		return err
	}
	return nil
}

func (bs *BenchStruct) Decode(dec *xdr.Decoder) error {
	id, err := dec.DecodeUint32()
	if err != nil {
		return err
	}
	bs.ID = id

	name, err := dec.DecodeString()
	if err != nil {
		return err
	}
	bs.Name = name

	data, err := dec.DecodeBytes()
	if err != nil {
		return err
	}
	bs.Data = data

	count, err := dec.DecodeUint64()
	if err != nil {
		return err
	}
	bs.Count = count

	return nil
}

// Compile-time assertion that BenchStruct implements Codec
var _ xdr.Codec = (*BenchStruct)(nil)

func BenchmarkCodecRoundTrip(b *testing.B) {
	impl := &BenchStruct{
		ID:    12345,
		Name:  "benchmark test",
		Data:  make([]byte, 256),
		Count: 9876543210,
	}
	rand.Read(impl.Data)

	b.Run("Marshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := xdr.Marshal(impl)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
		}
	})

	data, err := xdr.Marshal(impl)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}

	b.Run("Unmarshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var decoded BenchStruct
			err := xdr.Unmarshal(data, &decoded)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
	})

	b.Run("RoundTrip", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := xdr.Marshal(impl)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
			var decoded BenchStruct
			err = xdr.Unmarshal(data, &decoded)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
		b.SetBytes(int64(len(data)))
	})
}

func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("EncoderReuse", func(b *testing.B) {
		buf := make([]byte, 1024)
		encoder := xdr.NewEncoder(buf)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeUint32(0x12345678)
			encoder.EncodeString("hello")
			encoder.EncodeBytes([]byte("world"))
		}
	})

	b.Run("EncoderNew", func(b *testing.B) {
		buf := make([]byte, 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder := xdr.NewEncoder(buf)
			encoder.EncodeUint32(0x12345678)
			encoder.EncodeString("hello")
			encoder.EncodeBytes([]byte("world"))
		}
	})

	b.Run("DecoderReuse", func(b *testing.B) {
		buf := make([]byte, 1024)
		encoder := xdr.NewEncoder(buf)
		encoder.EncodeUint32(0x12345678)
		encoder.EncodeString("hello")
		encoder.EncodeBytes([]byte("world"))
		data := encoder.Bytes()

		decoder := xdr.NewDecoder(data)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder.Reset(data)
			decoder.DecodeUint32()
			decoder.DecodeString()
			decoder.DecodeBytes()
		}
	})

	b.Run("DecoderNew", func(b *testing.B) {
		buf := make([]byte, 1024)
		encoder := xdr.NewEncoder(buf)
		encoder.EncodeUint32(0x12345678)
		encoder.EncodeString("hello")
		encoder.EncodeBytes([]byte("world"))
		data := encoder.Bytes()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := xdr.NewDecoder(data)
			decoder.DecodeUint32()
			decoder.DecodeString()
			decoder.DecodeBytes()
		}
	})
}

func BenchmarkComparison(b *testing.B) {
	// Compare XDR encoding vs other methods
	data := make([]byte, 1024)
	rand.Read(data)

	b.Run("XDR_EncodeBytes", func(b *testing.B) {
		buf := make([]byte, 2048)
		encoder := xdr.NewEncoder(buf)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Reset(buf)
			encoder.EncodeBytes(data)
		}
		b.SetBytes(1024 + 4)
	})

	b.Run("Native_Copy", func(b *testing.B) {
		buf := make([]byte, 2048)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			copy(buf, data)
		}
		b.SetBytes(1024)
	})

	b.Run("Bytes_Buffer", func(b *testing.B) {
		var buf bytes.Buffer
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			buf.Write(data)
		}
		b.SetBytes(1024)
	})
}

// Discriminated union types for memory benchmarks
type MemBenchStatus uint32

const (
	MemBenchStatusSuccess MemBenchStatus = 0
	MemBenchStatusError   MemBenchStatus = 1
	MemBenchStatusPending MemBenchStatus = 2
)

type MemBenchResult struct {
	Status MemBenchStatus `xdr:"key"`
	Data   []byte         `xdr:"union,default=nil"`
}

//xdr:union=MemBenchResult,case=MemBenchStatusSuccess
type MemBenchSuccessResult struct {
	Message string `xdr:"string"`
	Details []byte `xdr:"bytes"`
}

func BenchmarkDiscriminatedUnionMemory(b *testing.B) {
	// Test memory allocation for discriminated unions
	successResult := &MemBenchResult{
		Status: MemBenchStatusSuccess,
		Data:   []byte("success payload with some data"),
	}

	errorResult := &MemBenchResult{
		Status: MemBenchStatusError,
		Data:   nil, // void case
	}

	b.Run("SuccessCase", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := xdr.Marshal(successResult)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
		}
		b.SetBytes(int64(len(successResult.Data) + 4)) // discriminant + payload
	})

	b.Run("VoidCase", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := xdr.Marshal(errorResult)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
		}
		b.SetBytes(4) // discriminant only
	})

	b.Run("SuccessUnmarshal", func(b *testing.B) {
		data, err := xdr.Marshal(successResult)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result MemBenchResult
			err := xdr.Unmarshal(data, &result)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
		b.SetBytes(int64(len(data)))
	})

	b.Run("VoidUnmarshal", func(b *testing.B) {
		data, err := xdr.Marshal(errorResult)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result MemBenchResult
			err := xdr.Unmarshal(data, &result)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
		b.SetBytes(int64(len(data)))
	})
}
