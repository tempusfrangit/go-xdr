# Discriminated Union XDR Example

This example demonstrates XDR discriminated unions (variant types) where the presence and type of data depends on a discriminant field.

## What it shows

- Discriminated unions with different payload types
- Conditional encoding based on discriminant values
- Void cases (no data) in union types
- Complex union structures with nested types

## Key concepts

- **Discriminant**: Field that determines which union case is active (defined by `+xdr:union,key=FieldName`)
- **Union Cases**: Different data types or void based on discriminant value
- **Void Case**: Union case with no associated data
- **Type Safety**: Compile-time validation of union specifications

## XDR Union Directives (No Struct Tags!)

- `// +xdr:union,key=FieldName[,default=ConstName]` - marks union container struct
- `// +xdr:payload,union=UnionName,discriminant=ConstName` - maps payload struct to discriminant value
- Auto-detection of `[]byte` field as union payload (no tags needed)

### Example
```go
// +xdr:union,key=Status
type OperationResult struct {
    Status Status // discriminant
    Data   []byte // auto-detected as union payload
}

// OpSuccessResult for successful operations
// +xdr:payload,union=OperationResult,discriminant=StatusSuccess
type OpSuccessResult struct {
    Message string // auto-detected as string
}
```

### Multi-Type Union Example
```go
// +xdr:union,key=Type
type NetworkMessage struct {
    Type    MessageType // discriminant
    Payload []byte      // auto-detected as union payload
}

// Text payload for text messages
// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeText
type TextPayload struct {
    Content string // auto-detected as string
    Sender  string // auto-detected as string
}

// Binary payload for binary messages
// +xdr:payload,union=NetworkMessage,discriminant=MessageTypeBinary
type BinaryPayload struct {
    Data     []byte // auto-detected as bytes
    Checksum uint32 // auto-detected as uint32
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