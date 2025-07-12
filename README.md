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

### Code Generation

Install the xdrgen tool:

```bash
go install github.com/tempusfrangit/go-xdr/tools/xdrgen@latest
```

Alternatively, you can run it directly without installing:

```bash
go run github.com/tempusfrangit/go-xdr/tools/xdrgen@latest <args>
```

Then define your structs with XDR tags:

```go
package main

//go:generate xdrgen $GOFILE

import "github.com/tempusfrangit/go-xdr"

type Person struct {
    ID   uint32 `xdr:"uint32"`
    Name string `xdr:"string"`
    Age  uint32 `xdr:"uint32"`
}

func main() {
    p := &Person{ID: 1, Name: "Alice", Age: 30}
    
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

### Supported XDR Tags

- `xdr:"uint32"` - 32-bit unsigned integer
- `xdr:"uint64"` - 64-bit unsigned integer
- `xdr:"int64"` - 64-bit signed integer
- `xdr:"string"` - variable-length string
- `xdr:"bytes"` - variable-length byte array
- `xdr:"bool"` - boolean value
- `xdr:"struct"` - nested struct (must implement xdr.Codec)
- `xdr:"array"` - variable-length array
- `xdr:"fixed:N"` - fixed-size byte array (N bytes)
- `xdr:"alias:TYPE"` - type alias with custom encoding
- `xdr:"-"` - exclude field from encoding/decoding

### Discriminated Unions

Discriminated unions use a clean, intuitive syntax with automatic void case detection:

```go
type OperationResult struct {
    Status Status `xdr:"key"`
    Data   []byte `xdr:"union,default=nil"`
}

// OpSuccessResult for successful operations
//xdr:union=OperationResult,case=StatusSuccess
type OpSuccessResult struct {
    Message string `xdr:"string"`
}
```

#### Key Features

- **Clean Syntax**: `//xdr:union=<container>,case=<constant>` on payload structs
- **Automatic Void Cases**: Any constants not mapped to payloads are automatically void (0 bytes)
- **Type Safety**: Discriminant must be uint32 or uint32 alias, constants must be typed
- **XDR Compliant**: Void cases marshal to 4 bytes (discriminant only), payload cases include data

#### Example: Multi-Type Union

```go
type NetworkMessage struct {
    Type    MessageType `xdr:"key"`
    Payload []byte      `xdr:"union,default=nil"`
}

// Text payload for text messages
//xdr:union=NetworkMessage,case=MessageTypeText
type TextPayload struct {
    Content string `xdr:"string"`
    Sender  string `xdr:"string"`
}

// Binary payload for binary messages  
//xdr:union=NetworkMessage,case=MessageTypeBinary
type BinaryPayload struct {
    Data     []byte `xdr:"bytes"`
    Checksum uint32 `xdr:"uint32"`
}
```

#### Void Cases

Void cases are automatically inferred from constants not mapped to payload structs:

```go
const (
    StatusSuccess Status = 0  // mapped to OpSuccessResult
    StatusError   Status = 1  // void case (no payload)
    StatusPending Status = 2  // void case (no payload)
)
```

- `StatusError` and `StatusPending` are automatically void cases
- They marshal to 4 bytes total (discriminant only)
- No explicit configuration needed

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
- **[autogen/](examples/autogen/)** - Auto-generated XDR methods using struct tags
- **[discriminated-union/](examples/discriminated-union/)** - Discriminated unions with conditional encoding
- **[mixed-manual/](examples/mixed-manual/)** - Mixed auto-generated and manual XDR implementations

Each example includes a README with detailed explanations and can be run independently.

## License

This project is licensed under the Apache 2.0 License.