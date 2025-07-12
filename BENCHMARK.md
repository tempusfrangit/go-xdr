# XDR Performance Benchmarks

This document contains up-to-date performance benchmarks for the XDR library, demonstrating allocation-efficient design and high-throughput capabilities.

## System Information

- **CPU**: Apple M2 Max (12 cores)
- **Architecture**: darwin/arm64
- **Go Version**: 1.24
- **Test Date**: 2025-07-11

## Benchmark Results

### Template-Generated XDR Code Performance

#### Person (simple struct)
| Operation   | ns/op   | B/op | allocs/op |
|-------------|---------|------|-----------|
| Encode      | 11.63   | 0    | 0         |
| Decode      | 53.18   | 64   | 4         |
| Marshal     | 115.8   | 592  | 3         |
| Unmarshal   | 72.13   | 96   | 5         |

#### Company (nested struct)
| Operation   | ns/op   | B/op | allocs/op |
|-------------|---------|------|-----------|
| Encode      | 62.87   | 0    | 0         |
| Decode      | 295.8   | 464  | 19        |
| Marshal     | 175.1   | 768  | 3         |
| Unmarshal   | 317.7   | 496  | 20        |

#### Config (mixed types)
| Operation   | ns/op   | B/op | allocs/op |
|-------------|---------|------|-----------|
| Encode      | 39.85   | 0    | 0         |
| Decode      | 166.6   | 184  | 12        |
| Marshal     | 135.5   | 656  | 3         |
| Unmarshal   | 184.0   | 216  | 13        |

#### Discriminated Unions
| Case         | Encode ns/op | Decode ns/op | B/op | allocs/op |
|--------------|-------------|--------------|------|-----------|
| Result/Success | 6.30      | 18.23        | 32   | 1         |
| Result/Error   | 2.22      | 2.15         | 0    | 0         |
| Message/Text   | 7.53      | 15.32        | 16   | 1         |
| Message/Void   | 2.34      | 2.26         | 0    | 0         |
| Operation/Read | 7.44      | 17.51        | 24   | 1         |
| Operation/Write| 2.26      | 2.17         | 0    | 0         |

#### Marshal/Unmarshal (Round-Trip)
| Type     | Marshal ns/op | Unmarshal ns/op | B/op | allocs/op |
|----------|---------------|-----------------|------|-----------|
| Person   | 117.2         | 70.82           | 592  | 3         |
| Company  | 182.8         | 322.5           | 768  | 3         |
| Config   | 139.0         | 185.4           | 656  | 3         |
| Union/Success | 114.6    | 33.95           | 592  | 3         |
| Union/Void    | 105.0    | 17.45           | 548  | 3         |

### Encoder/Decoder Primitives
| Operation         | ns/op  | B/op | allocs/op |
|-------------------|--------|------|-----------|
| EncodeUint32      | 0.91   | 0    | 0         |
| EncodeString      | 4.89   | 0    | 0         |
| EncodeBytes       | 5.40   | 0    | 0         |
| DecodeUint32      | 0.92   | 0    | 0         |
| DecodeString      | 15.79  | 16   | 1         |
| DecodeBytes       | 23.64  | 112  | 1         |

### Summary
- All encode paths for generated types are zero-allocation.
- Decoding incurs minimal allocations, only for output buffers/fields.
- Discriminated union codegen is as fast as hand-written code.
- Encoder/decoder primitives are sub-nanosecond for fixed types, and a few nanoseconds for strings/bytes.
- No legacy or NFS/architectural code remains in the benchmarks.

## Running Benchmarks

```bash
go test -bench=. -benchmem -tags=bench
```