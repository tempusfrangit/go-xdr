# go-xdr

**Minimum Go version: 1.20**

An allocation-efficient XDR (External Data Representation) library for Go with code generation support.

## Features

- **Allocation-efficient design** for performance-critical paths
- **Comprehensive XDR support** including all primitive types, arrays, and structures
- **Discriminated union support** with automatic pattern discovery
- **Code generation** via `xdrgen` tool for automatic Encode/Decode methods
- **Streaming support** for large data processing
- **Type-safe** with compile-time validation

## Installation

```bash
go get github.com/tempusfrangit/go-xdr
```

## Usage

### Manual Encoding/Decoding

```go
package main

import (
    "fmt"

    "github.com/tempusfrangit/go-xdr"
)

func main() {
    // Create encoder
    buf := make([]byte, 1024)
    enc := xdr.NewEncoder(buf)
    
    // Encode data
    enc.EncodeUint32(42)
    enc.EncodeString("hello")
    enc.EncodeBytes([]byte("world"))
    
    // Get encoded data
    data := enc.Bytes()
    
    // Create decoder
    dec := xdr.NewDecoder(data)
    
    // Decode data
    num, _ := dec.DecodeUint32()
    str, _ := dec.DecodeString()
    bytes, _ := dec.DecodeBytes()
    
    fmt.Printf("Decoded: %d, %s, %s\n", num, str, string(bytes))
}
```

### Code Generation (Ultra-Minimal)

Install the xdrgen tool:

```bash
go install github.com/tempusfrangit/go-xdr/tools/xdrgen@latest
```

Alternatively, you can run it directly without installing:

```bash
go run github.com/tempusfrangit/go-xdr/tools/xdrgen@latest <args>
```

Define your structs with **minimal tagging** - everything auto-detected:

```go
package main

//go:generate xdrgen $GOFILE

import "github.com/tempusfrangit/go-xdr"

// Type aliases (auto-resolved)
type UserID string
type Hash [32]byte

// +xdr:generate
type Person struct {
    ID     UserID  // auto-detected as string
    Name   string  // auto-detected as string
    Age    uint32  // auto-detected as uint32
    Hash   Hash    // auto-detected as bytes with v.Hash[:] conversion
    Active bool    // auto-detected as bool
    Secret string  `xdr:"-"` // excluded from encoding
}

func main() {
    p := &Person{
        ID: UserID("alice123"), 
        Name: "Alice", 
        Age: 30,
        Hash: Hash{0x01, 0x02, /* ... */},
        Active: true,
        Secret: "password", // not encoded
    }
    
    // Marshal to XDR
    data, err := xdr.Marshal(p)
    if err != nil {
        panic(err)
    }
    
    // Unmarshal from XDR
    var p2 Person
    err = xdr.Unmarshal(data, &p2)
    if err != nil {
        panic(err)
    }
}
```

Run code generation:

```bash
go generate
```

### Ultra-Minimal Directive System

#### Struct Opt-in
- `// +xdr:generate` - Mark regular struct for XDR code generation
- `// +xdr:union,key=FieldName[,default=ConstName]` - Mark union container struct
- `// +xdr:payload,union=UnionName,discriminant=ConstName` - Mark union payload struct

#### Only 1 Tag Needed!
- `xdr:"-"` - exclude field from encoding

#### Everything Else Auto-Detected
**Cross-package type handling**: Automatically detects and resolves type aliases across packages, with proper handling of types that have incompatible Encode/Decode methods.
**Basic types** (from Go syntax):
- `uint32`, `uint64`, `int32`, `int64` - integers
- `string` - strings
- `[]byte` - byte arrays  
- `bool` - booleans
- `[]Type`, `[N]Type` - arrays (element type auto-detected)
- `CustomStruct` - structs (must implement xdr.Codec)

**Type aliases** (auto-resolved with recursive unwrapping and cross-package support):
- `type UserID string` → string with casting
- `type Hash [16]byte` → bytes with `v.Hash[:]` conversion
- `type StatusCode uint32` → uint32 with casting
- `type Alias1 = Alias2; type Alias2 = string` → string (recursive)
- `type CrossPkgAlias = otherpkg.SomeType` → resolved to underlying type
- Smart detection of xdr.Codec interface compliance vs primitive encoding

### Discriminated Unions (Auto-Detected)

Unions are auto-detected from directive comments and `key + []byte` field patterns:

```go
// All-void union (default optional, auto-inferred)
// +xdr:union,key=Status
type StatusOnly struct {
    Status StatusCode // discriminant
    Data   []byte     // auto-detected as union payload
    // All StatusCode constants without payload structs → void cases
}

// Mixed union with void default
// +xdr:union,key=OpCode,default=nil
type MixedOperation struct {
    OpCode OpCode // discriminant
    Result []byte // auto-detected as union payload
}

// Payload structs use separate directive
// +xdr:payload,union=MixedOperation,discriminant=OpSuccess
type SuccessPayload struct {
    Message string // auto-detected as string
    Code    uint32 // auto-detected as uint32
}
```

#### Key Features

- **Directive-based**: `// +xdr:union,key=FieldName` instead of struct tags
- **Auto-detection**: `[]byte` field immediately following discriminant = union payload
- **Separate payload directives**: `// +xdr:payload,union=UnionName,discriminant=ConstName`
- **All-void unions**: `default=` optional (automatically inferred when no payloads exist)
- **Mixed unions**: `default=nil` for void default or `default=StructName` for struct default
- **Alias resolution**: Discriminant can be any uint32 alias, automatically resolved
- **Type safety**: Compile-time validation with interface assertions

#### Example: Real-World Usage

```go
// Type aliases for domain clarity
type UserID string
type SessionToken [16]byte
type RequestID uint64

// +xdr:generate
type User struct {
    ID      UserID       // auto-detected as string
    Token   SessionToken // auto-detected as bytes with v.Token[:]
    LastReq RequestID    // auto-detected as uint64
    Active  bool         // auto-detected as bool
    Internal string      `xdr:"-"` // excluded
}

// +xdr:union,key=Code,default=APIUnknown
type APIResponse struct {
    Code APICode // discriminant
    Data []byte  // auto-detected union payload
}

// Payload structs (no tags needed!)
// +xdr:payload,union=APIResponse,discriminant=APISuccess
type SuccessPayload struct {
    UserID   UserID // auto-detected as string
    UserData []byte // auto-detected as bytes
}

// +xdr:payload,union=APIResponse,discriminant=APIError
type ErrorPayload struct {
    Message string // auto-detected as string
    Code    uint32 // auto-detected as uint32
}
```

#### Union Semantics

**All-void unions** (no payload structs exist):
```go
const (
    StatusSuccess Status = 0  // void case
    StatusError   Status = 1  // void case  
    StatusPending Status = 2  // void case
)
// All constants → void, marshal to 4 bytes (discriminant only)
// default=nil is optional (auto-inferred)
```

**Mixed unions** (some constants have payload structs):
```go
const (
    OpSuccess OpCode = 0  // has SuccessPayload struct → non-void
    OpError   OpCode = 1  // has ErrorPayload struct → non-void
    OpPing    OpCode = 2  // no struct → void case
)
// default=nil or default=StructName REQUIRED for mixed unions
```

### Building

```bash
# Build xdrgen tool
make build

# Install xdrgen globally
make install

# Run tests
make test

# Run all checks
make check
```

## Performance

XDR-Go is designed with allocation-efficient encoding/decoding patterns for optimal performance. See [BENCHMARK.md](BENCHMARK.md) for detailed performance analysis and benchmark results.

## Examples

Comprehensive examples are available in the [examples/](examples/) directory:

- **[encode-decode/](examples/encode-decode/)** - Basic XDR encoding/decoding operations
- **[autogen/](examples/autogen/)** - Ultra-minimal auto-generated XDR methods
- **[discriminated-union/](examples/discriminated-union/)** - Auto-detected discriminated unions
- **[alias/](examples/alias/)** - Type alias resolution and conversion
- **[mixed-manual/](examples/mixed-manual/)** - Mixed auto-generated and manual XDR implementations

Each example includes a README with detailed explanations and can be run independently.

## License

This project is licensed under the Apache 2.0 License.