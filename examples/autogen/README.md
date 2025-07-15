# Auto-Generated XDR Example

This example demonstrates automatic XDR method generation using the `xdrgen` tool with auto-detection.

## What it shows

- Auto-detection of XDR types from Go types
- Generated `Encode` and `Decode` methods
- Nested struct encoding with arrays and slices
- Complex data structures with zero boilerplate

## Key concepts

- **`// +xdr:generate` directive**: Mark structs for XDR code generation
- **Auto-Detection**: XDR types automatically inferred from Go types
- **Code Generation**: `go generate` integration with `xdrgen`
- **Codec Interface**: Auto-generated types implement `xdr.Codec`
- **Type Safety**: Compile-time validation of structures

## Auto-Detected Types

- `uint32`, `uint64`, `int32`, `int64` - Integer types
- `string` - Variable-length string
- `[]byte` - Variable-length byte array
- `bool` - Boolean value
- `[]Type`, `[N]Type` - Arrays and slices
- `CustomStruct` - Nested structures

## Running the example

```bash
# Generate XDR methods
go generate

# Run the example
go run main.go
```

## Generated files

After running `go generate`, you'll see `types_xdr.go` containing the auto-generated XDR methods for all tagged structures.