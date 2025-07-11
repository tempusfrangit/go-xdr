# Auto-Generated XDR Example

This example demonstrates automatic XDR method generation using struct tags and the `xdrgen` tool.

## What it shows

- XDR struct tags for automatic code generation
- Generated `MarshalXDR` and `UnmarshalXDR` methods
- Nested struct encoding with arrays and slices
- Complex data structures with minimal boilerplate

## Key concepts

- **XDR Tags**: Struct field tags that specify XDR encoding types
- **Code Generation**: `go generate` integration with `xdrgen`
- **Codec Interface**: Auto-generated types implement `xdr.Codec`
- **Type Safety**: Compile-time validation of XDR-tagged structures

## XDR Tags

- `xdr:"uint32"` - 32-bit unsigned integer
- `xdr:"uint64"` - 64-bit unsigned integer  
- `xdr:"string"` - Variable-length string
- `xdr:"bytes"` - Variable-length byte array
- `xdr:"bool"` - Boolean value
- `xdr:"array"` - Variable-length array/slice
- `xdr:"struct"` - Nested structure

## Running the example

```bash
# Generate XDR methods
go generate

# Run the example
go run main.go
```

## Generated files

After running `go generate`, you'll see `types_xdr.go` containing the auto-generated XDR methods for all tagged structures.