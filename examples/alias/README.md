# XDR Alias Types Example

This example demonstrates how to use type aliases with XDR encoding/decoding.

## What are Alias Types?

Type aliases in Go provide type safety while maintaining the same underlying representation. For example:

```go
type UserID string
type StatusCode uint32
type IsActive bool
```

These aliases are useful for:
- Preventing accidental mixing of different string types (e.g., UserID vs SessionID)
- Making code more readable and self-documenting
- Providing compile-time type safety

## How XDR Alias Support Works

The XDR library supports aliases through the `alias:TYPE` tag:

```go
type User struct {
    ID       UserID     `xdr:"alias:string"`
    Status   StatusCode `xdr:"alias:uint32"`
    Active   IsActive   `xdr:"alias:bool"`
}
```

## Generated Code

The code generator creates efficient encode/decode methods:

**Encode:**
```go
if err := enc.EncodeString(v.ID); err != nil { ... }
if err := enc.EncodeUint32(v.Status); err != nil { ... }
if err := enc.EncodeBool(v.Active); err != nil { ... }
```

**Decode (with type conversion):**
```go
tmp, err := dec.DecodeString()
if err != nil {
    return fmt.Errorf("failed to decode ID: %w", err)
}
v.ID = UserID(tmp)

tmp, err := dec.DecodeUint32()
if err != nil {
    return fmt.Errorf("failed to decode Status: %w", err)
}
v.Status = StatusCode(tmp)

tmp, err := dec.DecodeBool()
if err != nil {
    return fmt.Errorf("failed to decode Active: %w", err)
}
v.Active = IsActive(tmp)
```

## Benefits

- **Zero allocations**: Type conversions are free for primitive types
- **Type safety**: Compile-time prevention of type mixing
- **Performance**: No runtime reflection or switching
- **Clean code**: Idiomatic Go with clear intent

## Supported Types

All primitive XDR types support aliasing:
- `string` → `type MyString string`
- `[]byte` → `type MyBytes []byte`
- `uint32` → `type MyUint32 uint32`
- `uint64` → `type MyUint64 uint64`
- `int32` → `type MyInt32 int32`
- `int64` → `type MyInt64 int64`
- `bool` → `type MyBool bool`

## Running the Example

```bash
cd examples/alias
go generate
go run main.go
```

This will demonstrate round-trip encoding/decoding of all alias types.