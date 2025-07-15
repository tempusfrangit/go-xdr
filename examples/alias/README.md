# XDR Alias Types Example

This example demonstrates how type aliases work with XDR auto-detection.

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

The XDR library automatically detects and resolves type aliases:

```go
// +xdr:generate
type User struct {
    ID       UserID     // auto-detected as string
    Status   StatusCode // auto-detected as uint32
    Active   IsActive   // auto-detected as bool
}
```

## Generated Code

The code generator creates efficient encode/decode methods with automatic type casting:

**Encode:**
```go
if err := enc.EncodeString(string(v.ID)); err != nil { ... }
if err := enc.EncodeUint32(uint32(v.Status)); err != nil { ... }
if err := enc.EncodeBool(bool(v.Active)); err != nil { ... }
```

**Decode (with type conversion):**
```go
tempID, err := dec.DecodeString()
if err != nil {
    return fmt.Errorf("failed to decode ID: %w", err)
}
v.ID = UserID(tempID)

tempStatus, err := dec.DecodeUint32()
if err != nil {
    return fmt.Errorf("failed to decode Status: %w", err)
}
v.Status = StatusCode(tempStatus)

tempActive, err := dec.DecodeBool()
if err != nil {
    return fmt.Errorf("failed to decode Active: %w", err)
}
v.Active = IsActive(tempActive)
```

## Benefits

- **Zero allocations**: Type conversions are free for primitive types
- **Type safety**: Compile-time prevention of type mixing
- **Performance**: No runtime reflection or switching
- **Clean code**: Idiomatic Go with clear intent

## Supported Types

All primitive XDR types support automatic alias detection:
- `string` → `type MyString string`
- `[]byte` → `type MyBytes []byte`
- `[N]byte` → `type MyHash [N]byte`
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