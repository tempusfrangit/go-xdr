# Discriminated Union XDR Example

This example demonstrates XDR discriminated unions (variant types) where the presence and type of data depends on a discriminant field.

## What it shows

- Discriminated unions with different payload types
- Conditional encoding based on discriminant values
- Void cases (no data) in union types
- Complex union structures with nested types

## Key concepts

- **Discriminant**: Field that determines which union case is active (tagged with `xdr:"key"`)
- **Union Cases**: Different data types or void based on discriminant value (tagged with `xdr:"union,default=Type"` or `xdr:"union,default=nil"` for void)
- **Void Case**: Union case with no associated data
- **Type Safety**: Compile-time validation of union specifications

## XDR Union Tags (Clean Syntax)

- `xdr:"key"` - marks the discriminant field (must be uint32 or uint32 alias)
- `xdr:"union,default=nil"` - marks the union payload field with void default
- `//xdr:union=<container>,case=<constant>` - comment on payload struct to map discriminant values

### Example
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

### Multi-Type Union Example
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

### Automatic Void Cases

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

## Running the example

```bash
# Generate XDR methods
go generate

# Run the example
go run main.go
```

## Expected output

The example demonstrates encoding/decoding of discriminated unions with different cases, including void cases where no data is present.