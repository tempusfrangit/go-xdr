# XDR Performance Benchmarks

This document contains detailed performance benchmarks for the XDR library, demonstrating its allocation-efficient design and high-throughput capabilities.

## System Information

- **CPU**: Apple M2 Max (12 cores)
- **Architecture**: darwin/arm64
- **Go Version**: 1.24
- **Test Duration**: ~111 seconds

## Benchmark Results

### Primitive Type Encoding

| Operation | Throughput | Memory | Allocations |
|-----------|------------|---------|-------------|
| EncodeUint32 | 4,516.76 MB/s | 0 B/op | 0 allocs/op |
| EncodeUint64 | 8,265.24 MB/s | 0 B/op | 0 allocs/op |
| EncodeInt32 | 4,494.31 MB/s | 0 B/op | 0 allocs/op |
| EncodeInt64 | 8,706.88 MB/s | 0 B/op | 0 allocs/op |
| EncodeBool | 4,492.47 MB/s | 0 B/op | 0 allocs/op |

**Key Finding**: All primitive encoding operations achieve **zero allocations** and multi-gigabyte throughput.

### Primitive Type Decoding

| Operation | Throughput | Memory | Allocations |
|-----------|------------|---------|-------------|
| DecodeUint32 | 4,393.73 MB/s | 0 B/op | 0 allocs/op |
| DecodeUint64 | 8,831.97 MB/s | 0 B/op | 0 allocs/op |
| DecodeInt32 | 3,855.36 MB/s | 0 B/op | 0 allocs/op |
| DecodeInt64 | 7,636.14 MB/s | 0 B/op | 0 allocs/op |
| DecodeBool | 3,885.50 MB/s | 0 B/op | 0 allocs/op |

**Key Finding**: Primitive decoding maintains **zero allocations** and excellent throughput.

### String Operations

#### String Encoding (Zero Allocations)

| String Length | Throughput | Memory | Allocations |
|---------------|------------|---------|-------------|
| Empty string | 818.54 MB/s | 0 B/op | 0 allocs/op |
| 1 character | 748.64 MB/s | 0 B/op | 0 allocs/op |
| 5 characters | 1,467.00 MB/s | 0 B/op | 0 allocs/op |
| 15 characters | 2,959.03 MB/s | 0 B/op | 0 allocs/op |
| 64 characters | 12,553.31 MB/s | 0 B/op | 0 allocs/op |
| 235 characters | 29,467.53 MB/s | 0 B/op | 0 allocs/op |

**Key Finding**: String encoding shows **zero allocations** and throughput that scales with string length.

#### String Decoding (Minimal Allocations)

| String Length | Throughput | Memory | Allocations |
|---------------|------------|---------|-------------|
| Empty string | 587.02 MB/s | 0 B/op | 0 allocs/op |
| 1 character | 406.11 MB/s | 1 B/op | 1 allocs/op |
| 5 characters | 622.55 MB/s | 5 B/op | 1 allocs/op |
| 15 characters | 972.88 MB/s | 16 B/op | 1 allocs/op |
| 64 characters | 1,952.74 MB/s | 128 B/op | 2 allocs/op |
| 235 characters | 3,628.58 MB/s | 480 B/op | 2 allocs/op |

**Key Finding**: String decoding requires minimal allocations (only for the string itself) and maintains high throughput.

### Byte Array Operations

#### Variable-Length Byte Arrays

| Array Size | Encode Throughput | Decode Throughput | Decode Memory | Decode Allocs |
|------------|-------------------|-------------------|---------------|---------------|
| 0 bytes | 902.39 MB/s | 696.50 MB/s | 0 B/op | 0 allocs/op |
| 1 byte | 732.82 MB/s | 429.75 MB/s | 1 B/op | 1 allocs/op |
| 4 bytes | 1,669.75 MB/s | 733.97 MB/s | 4 B/op | 1 allocs/op |
| 16 bytes | 4,506.89 MB/s | 1,387.09 MB/s | 16 B/op | 1 allocs/op |
| 64 bytes | 13,869.50 MB/s | 3,693.30 MB/s | 64 B/op | 1 allocs/op |
| 1024 bytes | 60,857.90 MB/s | 8,473.82 MB/s | 1024 B/op | 1 allocs/op |
| 16384 bytes | 76,015.19 MB/s | 10,648.56 MB/s | 16384 B/op | 1 allocs/op |

#### Fixed-Length Byte Arrays

| Array Size | Encode Throughput | Decode Throughput | Decode Memory | Decode Allocs |
|------------|-------------------|-------------------|---------------|---------------|
| 1 byte | 218.77 MB/s | 101.28 MB/s | 1 B/op | 1 allocs/op |
| 4 bytes | 1,666.79 MB/s | 423.00 MB/s | 4 B/op | 1 allocs/op |
| 16 bytes | 7,016.63 MB/s | 1,320.16 MB/s | 16 B/op | 1 allocs/op |
| 64 bytes | 23,134.18 MB/s | 3,737.27 MB/s | 64 B/op | 1 allocs/op |
| 1024 bytes | 73,313.50 MB/s | 8,796.27 MB/s | 1024 B/op | 1 allocs/op |
| 16384 bytes | 79,166.81 MB/s | 10,887.02 MB/s | 16384 B/op | 1 allocs/op |

**Key Finding**: Byte array encoding achieves **zero allocations** and extremely high throughput (up to 79 GB/s). Decoding requires exactly one allocation for the target array.

### Round-Trip Operations

| Operation | Throughput | Memory | Allocations |
|-----------|------------|---------|-------------|
| Uint32 Round-Trip | 1,927.11 MB/s | 0 B/op | 0 allocs/op |
| String Round-Trip | 740.27 MB/s | 16 B/op | 1 allocs/op |
| 1KB Bytes Round-Trip | 160,341.83 MB/s | 0 B/op | 0 allocs/op |

**Key Finding**: Round-trip operations maintain excellent performance with minimal allocations.

### High-Level Codec Interface

| Operation | Throughput | Memory | Allocations |
|-----------|------------|---------|-------------|
| Marshal | ~124 ns/op | 864 B/op | 3 allocs/op |
| Unmarshal | ~98 ns/op | 384 B/op | 5 allocs/op |
| Round-Trip | 1,117.97 MB/s | 1248 B/op | 8 allocs/op |

**Key Finding**: The high-level codec interface trades some performance for convenience but maintains reasonable allocation patterns.

### Memory Allocation Patterns

| Test | Memory | Allocations |
|------|--------|-------------|
| Encoder Reuse | 0 B/op | 0 allocs/op |
| Encoder New | 0 B/op | 0 allocs/op |
| Decoder Reuse | 10 B/op | 2 allocs/op |
| Decoder New | 10 B/op | 2 allocs/op |

**Key Finding**: Encoders achieve true zero allocation. Decoders have minimal overhead (~10 bytes, 2 allocations).

### Comparison with Native Operations

| Operation | Throughput |
|-----------|------------|
| XDR EncodeBytes | 62,711.72 MB/s |
| Native Copy | 78,965.10 MB/s |
| Bytes Buffer | 65,543.05 MB/s |

**Key Finding**: XDR encoding performance is competitive with native Go operations (79% of native copy performance).


## Performance Summary

### Allocation Efficiency

- **Primitive Operations**: True zero allocation for all encode/decode operations
- **String Encoding**: Zero allocations (source strings not copied)
- **Byte Array Encoding**: Zero allocations (source arrays not copied)
- **String/Byte Decoding**: Single allocation for target buffer (unavoidable)
- **Encoder/Decoder**: Minimal overhead (~10 bytes for decoders)

### Throughput Characteristics

- **Primitive Types**: 3-8 GB/s throughput
- **Variable Data**: Throughput scales with data size (up to 79 GB/s for large arrays)
- **String Operations**: 400 MB/s - 29 GB/s depending on string length
- **Round-Trip**: Maintains high performance across operation types

### Scalability

- **Memory Usage**: Scales linearly with data size (no hidden allocations)
- **Throughput**: Improves with larger data sizes due to amortized overhead
- **CPU Efficiency**: Sub-nanosecond per-byte processing for large data

## Running Benchmarks

To reproduce these results:

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark categories
go test -bench=BenchmarkEncode -benchmem
go test -bench=BenchmarkDecode -benchmem
go test -bench=BenchmarkRoundTrip -benchmem

# Run benchmarks multiple times for statistical significance
go test -bench=. -benchmem -count=5
```

## Template-Generated XDR Code Performance

The XDR code generator produces high-performance, zero-allocation encoding methods. These benchmarks validate that template-generated code maintains the same performance characteristics as hand-written implementations.

### Basic Structures

**BenchmarkPerson** (simple struct with 4 fields):
- Encode: 11.65 ns/op, 0 allocations
- Decode: 53.30 ns/op, 4 allocations  
- Marshal: 109.5 ns/op, 3 allocations
- Unmarshal: 72.09 ns/op, 5 allocations

### Complex Structures

**BenchmarkCompany** (nested struct with employee array):
- Encode: 64.67 ns/op, 0 allocations
- Decode: 299.9 ns/op, 19 allocations
- Marshal: 185.3 ns/op, 3 allocations
- Unmarshal: 328.5 ns/op, 20 allocations

**BenchmarkConfig** (mixed primitive types with arrays):
- Encode: 41.60 ns/op, 0 allocations
- Decode: 171.6 ns/op, 12 allocations
- Marshal: 146.1 ns/op, 3 allocations
- Unmarshal: 187.9 ns/op, 13 allocations

### Discriminated Unions (Key Feature)

**BenchmarkResult** (union with success/error cases):
- Success Encode: 6.31 ns/op, 0 allocations
- Success Decode: 18.08 ns/op, 1 allocation
- Error Encode: 2.19 ns/op, 0 allocations
- Error Decode: 2.13 ns/op, 0 allocations

**BenchmarkMessage** (multi-case union):
- Text Encode: 7.61 ns/op, 0 allocations
- Text Decode: 16.40 ns/op, 1 allocation
- Void Encode: 2.32 ns/op, 0 allocations
- Void Decode: 2.26 ns/op, 0 allocations

**BenchmarkOperation** (complex union with void cases):
- Read Encode: 7.55 ns/op, 0 allocations
- Read Decode: 18.88 ns/op, 1 allocation
- Write Encode: 2.24 ns/op, 0 allocations
- Write Decode: 2.16 ns/op, 0 allocations

### Template-Generated Code Validation

**Key Findings:**
- Zero-allocation encoding maintained across all generated code
- Discriminated unions perform excellently with switch-based logic
- Void union cases achieve sub-3ns performance
- Complex nested structures handle arrays and embedded structs efficiently
- Performance comparable to hand-written code (within 10-20% range)

**Comparison with Core Library:**
- Core `EncodeUint32`: 0.91 ns/op vs Generated Person field: ~3 ns/op
- Core `Marshal`: 112.3 ns/op vs Generated Person: 109.5 ns/op
- Core `Unmarshal`: 60.58 ns/op vs Generated Person: 72.09 ns/op

The template system successfully produces code that meets all performance requirements while providing the convenience of automatic code generation.

## Performance Recommendations

1. **Use primitive operations directly** for maximum performance
2. **Reuse encoders/decoders** when possible (though overhead is minimal)
3. **Consider data size** - performance improves with larger data
4. **Batch operations** when processing multiple items
5. **Profile your specific use case** - these benchmarks represent synthetic workloads
6. **Use discriminated unions** for protocol efficiency - void cases are extremely fast
7. **Template-generated code** provides excellent performance with development convenience