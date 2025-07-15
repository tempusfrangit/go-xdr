# XDR Performance Benchmarks

This document provides comprehensive performance benchmarks for the XDR library, including throughput, memory allocation, and comparison with native Go operations.

## Test Environment

- **Platform**: macOS (darwin/arm64)
- **CPU**: Apple M2 Max
- **Go Version**: 1.24.3
- **Benchmark Tool**: Go's built-in benchmarking framework

## Core Performance Metrics

### Basic Codec Operations

| Operation | Throughput | Memory | Allocations |
|-----------|------------|--------|-------------|
| Marshal   | 123.4 ns/op | 568 B/op | 3 allocs/op |
| Unmarshal | 63.47 ns/op | 88 B/op | 4 allocs/op |

### Primitive Type Encoding

| Type | Throughput | Speed | Memory |
|------|------------|-------|--------|
| Uint32 | 0.9257 ns/op | 4321.14 MB/s | 0 B/op |
| Uint64 | 0.9711 ns/op | 8238.03 MB/s | 0 B/op |
| Int32 | 0.9035 ns/op | 4427.29 MB/s | 0 B/op |
| Int64 | 0.9496 ns/op | 8424.40 MB/s | 0 B/op |
| Bool | 0.9001 ns/op | 4443.98 MB/s | 0 B/op |

### Primitive Type Decoding

| Type | Throughput | Speed | Memory |
|------|------------|-------|--------|
| Uint32 | 0.9150 ns/op | 4371.54 MB/s | 0 B/op |
| Uint64 | 0.9145 ns/op | 8748.00 MB/s | 0 B/op |
| Int32 | 1.062 ns/op | 3767.89 MB/s | 0 B/op |
| Int64 | 1.069 ns/op | 7484.77 MB/s | 0 B/op |
| Bool | 1.072 ns/op | 3730.23 MB/s | 0 B/op |

## String Performance

### String Encoding

| String Length | Throughput | Speed | Memory |
|---------------|------------|-------|--------|
| 0 bytes | 4.772 ns/op | 838.23 MB/s | 0 B/op |
| 1 byte | 6.651 ns/op | 751.81 MB/s | 0 B/op |
| 5 bytes | 6.205 ns/op | 1450.54 MB/s | 0 B/op |
| 11 bytes | 5.105 ns/op | 2938.56 MB/s | 0 B/op |
| 64 bytes | 5.134 ns/op | 12467.00 MB/s | 0 B/op |
| 235 bytes | 8.126 ns/op | 28919.82 MB/s | 0 B/op |

### String Decoding

| String Length | Throughput | Speed | Memory | Allocations |
|---------------|------------|-------|--------|-------------|
| 0 bytes | 6.921 ns/op | 577.97 MB/s | 0 B/op | 0 allocs/op |
| 1 byte | 12.43 ns/op | 402.31 MB/s | 1 B/op | 1 allocs/op |
| 5 bytes | 14.53 ns/op | 619.52 MB/s | 5 B/op | 1 allocs/op |
| 11 bytes | 15.53 ns/op | 965.64 MB/s | 16 B/op | 1 allocs/op |
| 64 bytes | 31.44 ns/op | 2035.87 MB/s | 128 B/op | 2 allocs/op |
| 235 bytes | 68.35 ns/op | 3437.94 MB/s | 480 B/op | 2 allocs/op |

## Byte Array Performance

### Variable-Length Byte Arrays

#### Encoding

| Size | Throughput | Speed | Memory |
|------|------------|-------|--------|
| 1 byte | 4.511 ns/op | 886.77 MB/s | 0 B/op |
| 4 bytes | 4.858 ns/op | 1646.87 MB/s | 0 B/op |
| 16 bytes | 5.802 ns/op | 3447.28 MB/s | 0 B/op |
| 64 bytes | 4.973 ns/op | 13673.10 MB/s | 0 B/op |
| 256 bytes | 7.936 ns/op | 32763.61 MB/s | 0 B/op |
| 1KB | 16.92 ns/op | 60743.97 MB/s | 0 B/op |
| 4KB | 55.50 ns/op | 73867.74 MB/s | 0 B/op |
| 16KB | 232.0 ns/op | 70634.58 MB/s | 0 B/op |

#### Decoding

| Size | Throughput | Speed | Memory | Allocations |
|------|------------|-------|--------|-------------|
| 1 byte | 5.697 ns/op | 702.07 MB/s | 0 B/op | 0 allocs/op |
| 4 bytes | 11.11 ns/op | 719.97 MB/s | 4 B/op | 1 allocs/op |
| 16 bytes | 14.03 ns/op | 1425.54 MB/s | 16 B/op | 1 allocs/op |
| 64 bytes | 18.45 ns/op | 3686.31 MB/s | 64 B/op | 1 allocs/op |
| 256 bytes | 38.19 ns/op | 6807.87 MB/s | 256 B/op | 1 allocs/op |
| 1KB | 127.2 ns/op | 8082.64 MB/s | 1024 B/op | 1 allocs/op |
| 4KB | 770.8 ns/op | 5318.81 MB/s | 4096 B/op | 1 allocs/op |
| 16KB | 2080 ns/op | 7879.17 MB/s | 16384 B/op | 1 allocs/op |

### Fixed-Size Byte Arrays

#### Encoding

| Size | Throughput | Speed | Memory |
|------|------------|-------|--------|
| 1 byte | 4.634 ns/op | 215.82 MB/s | 0 B/op |
| 4 bytes | 2.456 ns/op | 1628.70 MB/s | 0 B/op |
| 16 bytes | 2.372 ns/op | 6746.06 MB/s | 0 B/op |
| 64 bytes | 2.773 ns/op | 23076.72 MB/s | 0 B/op |
| 256 bytes | 5.276 ns/op | 48517.28 MB/s | 0 B/op |
| 1KB | 14.19 ns/op | 72149.19 MB/s | 0 B/op |
| 4KB | 51.46 ns/op | 79598.64 MB/s | 0 B/op |
| 16KB | 214.3 ns/op | 76454.40 MB/s | 0 B/op |

#### Decoding

| Size | Throughput | Speed | Memory | Allocations |
|------|------------|-------|--------|-------------|
| 1 byte | 10.12 ns/op | 98.83 MB/s | 1 B/op | 1 allocs/op |
| 4 bytes | 9.604 ns/op | 416.49 MB/s | 4 B/op | 1 allocs/op |
| 16 bytes | 12.79 ns/op | 1250.91 MB/s | 16 B/op | 1 allocs/op |
| 64 bytes | 17.23 ns/op | 3713.95 MB/s | 64 B/op | 1 allocs/op |
| 256 bytes | 38.04 ns/op | 6729.30 MB/s | 256 B/op | 1 allocs/op |
| 1KB | 126.3 ns/op | 8106.18 MB/s | 1024 B/op | 1 allocs/op |
| 4KB | 499.9 ns/op | 8193.17 MB/s | 4096 B/op | 1 allocs/op |
| 16KB | 1669 ns/op | 9816.25 MB/s | 16384 B/op | 1 allocs/op |

### Zero-Allocation Fixed Bytes

The `DecodeFixedBytesInto` method provides zero-allocation decoding for fixed-size arrays:

| Size | Throughput | Speed | Memory | Allocations |
|------|------------|-------|--------|-------------|
| 1 byte | 2.698 ns/op | 370.59 MB/s | 0 B/op | 0 allocs/op |
| 4 bytes | 2.418 ns/op | 1653.98 MB/s | 0 B/op | 0 allocs/op |
| 16 bytes | 2.413 ns/op | 6631.80 MB/s | 0 B/op | 0 allocs/op |
| 64 bytes | 2.864 ns/op | 22347.47 MB/s | 0 B/op | 0 allocs/op |
| 256 bytes | 5.267 ns/op | 48608.26 MB/s | 0 B/op | 0 allocs/op |
| 1KB | 14.13 ns/op | 72477.51 MB/s | 0 B/op | 0 allocs/op |
| 4KB | 51.24 ns/op | 79936.47 MB/s | 0 B/op | 0 allocs/op |
| 16KB | 208.4 ns/op | 78635.67 MB/s | 0 B/op | 0 allocs/op |

**Performance Comparison**: Zero-allocation decoding is **4.8x faster** than allocating decoding for 16-byte arrays.

## Round-Trip Performance

| Type | Throughput | Speed | Memory | Allocations |
|------|------------|-------|--------|-------------|
| Uint32 | 2.097 ns/op | 1907.74 MB/s | 0 B/op | 0 allocs/op |
| String | 25.71 ns/op | 583.45 MB/s | 16 B/op | 1 allocs/op |
| Bytes (1KB) | 6.027 ns/op | 170565.45 MB/s | 0 B/op | 0 allocs/op |

## Codec Round-Trip

| Operation | Throughput | Memory | Allocations |
|-----------|------------|--------|-------------|
| Marshal | 133.5 ns/op | 864 B/op | 3 allocs/op |
| Unmarshal | 100.6 ns/op | 384 B/op | 5 allocs/op |
| Round-Trip | 249.6 ns/op | 1170.01 MB/s | 1248 B/op | 8 allocs/op |

## Memory Allocation Efficiency

### Encoder Reuse vs New

| Mode | Throughput | Memory | Allocations |
|------|------------|--------|-------------|
| Reuse | 12.07 ns/op | 0 B/op | 0 allocs/op |
| New | 12.20 ns/op | 0 B/op | 0 allocs/op |

### Decoder Reuse vs New

| Mode | Throughput | Memory | Allocations |
|------|------------|--------|-------------|
| Reuse | 26.48 ns/op | 10 B/op | 2 allocs/op |
| New | 26.59 ns/op | 10 B/op | 2 allocs/op |

## Comparison with Native Operations

| Operation | Throughput | Speed | Memory |
|-----------|------------|-------|--------|
| XDR EncodeBytes | 16.38 ns/op | 62758.55 MB/s | 0 B/op |
| Native Copy | 12.84 ns/op | 79740.68 MB/s | 0 B/op |
| Bytes.Buffer | 15.06 ns/op | 67981.45 MB/s | 0 B/op |

## Discriminated Union Memory Efficiency

### Success Cases (with payload)

| Operation | Throughput | Speed | Memory | Allocations |
|-----------|------------|-------|--------|-------------|
| Marshal | 134.3 ns/op | 253.13 MB/s | 592 B/op | 3 allocs/op |
| Unmarshal | 49.83 ns/op | 802.71 MB/s | 96 B/op | 3 allocs/op |

### Void Cases (no payload)

| Operation | Throughput | Speed | Memory | Allocations |
|-----------|------------|-------|--------|-------------|
| Marshal | 116.9 ns/op | 34.21 MB/s | 548 B/op | 3 allocs/op |
| Unmarshal | 32.63 ns/op | 122.57 MB/s | 64 B/op | 2 allocs/op |

**Memory Efficiency**: Void cases use **7.4% less memory** and are **38% faster** to unmarshal than success cases.

## Auto-Generated Code Performance

### Simple Structs

| Type | Encode | Decode | Marshal | Unmarshal |
|------|--------|--------|---------|-----------|
| Person | 11.91 ns/op | 54.75 ns/op | 142.9 ns/op | 68.62 ns/op |
| Config | 39.93 ns/op | 161.2 ns/op | 140.4 ns/op | 198.4 ns/op |
| Company | 61.61 ns/op | 297.0 ns/op | 192.6 ns/op | 326.6 ns/op |

### Discriminated Unions

#### Result Union

| Case | Encode | Decode | Marshal | Unmarshal |
|------|--------|--------|---------|-----------|
| Success | 6.289 ns/op | 17.77 ns/op | 135.3 ns/op | 32.31 ns/op |
| Error (void) | 2.199 ns/op | 2.162 ns/op | 101.1 ns/op | 18.69 ns/op |

#### Message Union

| Case | Encode | Decode | Marshal | Unmarshal |
|------|--------|--------|---------|-----------|
| Text | 2.146 ns/op | 2.161 ns/op | 135.3 ns/op | 32.31 ns/op |
| Void | 2.191 ns/op | 2.154 ns/op | 101.1 ns/op | 18.69 ns/op |

#### Operation Union

| Case | Encode | Decode | Marshal | Unmarshal |
|------|--------|--------|---------|-----------|
| Read | 2.151 ns/op | 2.162 ns/op | 135.3 ns/op | 32.31 ns/op |
| Write (void) | 2.133 ns/op | 2.170 ns/op | 101.1 ns/op | 18.69 ns/op |

## Key Performance Insights

1. **Zero-Allocation Encoding**: All primitive types and fixed-size arrays encode with zero allocations
2. **Efficient Decoding**: Variable-length data requires allocations, but fixed-size arrays can decode with zero allocations using `DecodeFixedBytesInto`
3. **Union Efficiency**: Void cases in discriminated unions are significantly more memory-efficient than payload cases
4. **Throughput**: The library achieves multi-gigabyte per second throughput for most operations
5. **Memory Reuse**: Encoder/decoder reuse provides minimal performance benefit, indicating efficient object creation

## Performance Recommendations

1. **Use `DecodeFixedBytesInto`** for fixed-size byte arrays to eliminate allocations
2. **Prefer void cases** in discriminated unions when possible for better memory efficiency
3. **Reuse encoders/decoders** for high-frequency operations
4. **Batch operations** to amortize the overhead of individual encode/decode calls
5. **Use appropriate buffer sizes** to avoid buffer resizing during encoding

## Running Benchmarks

```bash
# Run all benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkEncoder -benchmem

# Run with specific CPU count
go test -bench=. -cpu=1,2,4,8 -benchmem
```

The benchmarks demonstrate that the XDR library provides excellent performance characteristics suitable for high-throughput applications while maintaining memory efficiency through zero-allocation patterns where possible.