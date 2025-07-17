# Mixed Manual XDR Example

This example demonstrates manual XDR implementation alongside auto-generated code, showing advanced customization techniques.

## What it shows

- Custom `MarshalXDR` and `UnmarshalXDR` implementations
- Mixed auto-generated and manual encoding within the same struct
- Validation during encoding/decoding
- Custom data transformations (reversing arrays, checksums, etc.)

## Key concepts

- **Manual Override**: Implementing custom XDR methods to override auto-generation
- **Mixed Approaches**: Combining auto-generated and manual encoding in the same struct
- **Validation**: Adding custom validation logic during encoding/decoding
- **Data Transformation**: Modifying data during the XDR process

## Implementation Patterns

### Fully Manual Implementation
```go
func (c *CustomMessage) MarshalXDR(enc *xdr.Encoder) error {
    // Custom encoding logic
    return enc.EncodeUint32(c.Value)
}

func (c *CustomMessage) UnmarshalXDR(dec *xdr.Decoder) error {
    // Custom decoding logic
    var err error
    c.Value, err = dec.DecodeUint32()
    return err
}
```

### Mixed Auto/Manual
```go
// +xdr:generate
type MixedStruct struct {
    AutoField   string        // Auto-detected as string
    ManualField []uint32      // Manual implementation (implements xdr.Codec)
    Header      MessageHeader // Auto-detected as struct
}
```

### Validation During Encoding
```go
func (v *ValidatedData) MarshalXDR(enc *xdr.Encoder) error {
    // Calculate checksum before marshaling
    v.Checksum = calculateChecksum(v.Value)
    return enc.EncodeUint32(v.Value)
}
```

## Use Cases

- **Custom Protocols**: Implementing proprietary or legacy XDR formats
- **Data Validation**: Adding checksums, signatures, or other validation
- **Performance Optimization**: Custom encoding for specific data patterns
- **Backwards Compatibility**: Maintaining compatibility with existing systems

## Running the example

```bash
# Generate XDR methods for auto-generated fields
go generate

# Run the example
go run main.go
```

## Expected output

The example demonstrates various manual XDR implementations with custom logic, validation, and data transformations.