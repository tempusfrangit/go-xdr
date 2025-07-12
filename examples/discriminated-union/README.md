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

## XDR Union Tags (New Style)

- `xdr:"key"` - marks the discriminant field
- `xdr:"union,default=nil"` - marks the union payload field with a void default (optional)
- `xdr:"union,default=Type"` - marks the union payload field with a struct default (optional)
- `xdr:"union"` - marks the union payload field with no default case
- `//xdr:union=DiscriminantType,case=ConstantValue` - comment above the union payload struct to map discriminant values to payload types. Do not use `default` in the comment.

### Example
```go
type OperationResult struct {
    Status uint32 `xdr:"key"`
    Data   []byte `xdr:"union"`
}

//xdr:union=uint32,case=0=OpSuccessResult
// OpSuccessResult is the payload for StatusSuccess
// (add struct if you want a non-void payload)
```

### Multi-Type Union Example
```go
type NetworkMessage struct {
    Type    uint32 `xdr:"key"`
    Payload []byte `xdr:"union"`
}

//xdr:union=uint32,case=1=TextPayload,case=2=BinaryPayload
// No default case: only these discriminant values are valid
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