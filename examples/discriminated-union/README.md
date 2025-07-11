# Discriminated Union XDR Example

This example demonstrates XDR discriminated unions (variant types) where the presence and type of data depends on a discriminant field.

## What it shows

- Discriminated unions with different payload types
- Conditional encoding based on discriminant values
- Void cases (no data) in union types
- Complex union structures with nested types

## Key concepts

- **Discriminant**: Field that determines which union case is active
- **Union Cases**: Different data types or void based on discriminant value
- **Void Case**: Union case with no associated data
- **Type Safety**: Compile-time validation of union specifications

## XDR Union Tags

- `xdr:"union,discriminant:FieldName,case:value=Type"` - Union with specific case mapping
- `xdr:"bytes,discriminant:Status,case:0"` - Conditional field present only when discriminant equals 0
- `case:1=TextPayload,case:2=BinaryPayload,case:3=void` - Multiple union cases with different types

## Union Types

### Simple Discriminated Union
```go
type OperationResult struct {
    Status uint32 `xdr:"uint32"`
    Data   []byte `xdr:"bytes,discriminant:Status,case:0"`
}
```

### Multi-Type Union
```go
type NetworkMessage struct {
    Type    uint32      `xdr:"uint32"`
    Payload interface{} `xdr:"union,discriminant:Type,case:1=TextPayload,case:2=BinaryPayload,case:3=void"`
}
```

## Running the example

```bash
# Generate XDR methods
go generate

# Run the example
go run main.go
```

## Expected output

The example demonstrates encoding/decoding of discriminated unions with different cases, including void cases where no data is present.