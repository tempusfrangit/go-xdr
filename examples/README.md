# XDR Examples

This directory contains comprehensive examples demonstrating different aspects of the XDR library.

## Example Categories

### 1. [encode-decode](encode-decode/) - Basic XDR Operations
- Direct use of Encoder/Decoder
- Manual encoding/decoding of primitive types
- Round-trip validation
- **Best for**: Understanding XDR fundamentals

### 2. [autogen](autogen/) - Auto-Generated XDR
- Struct tags for automatic code generation
- Complex nested structures
- Generated `MarshalXDR`/`UnmarshalXDR` methods
- **Best for**: Rapid development with minimal boilerplate

### 3. [discriminated-union](discriminated-union/) - Discriminated Unions
- Clean syntax with automatic void case detection
- Variant types based on discriminant fields
- Conditional encoding/decoding with proper XDR compliance
- **Best for**: Protocol implementations with optional fields and variant types

### 4. [mixed-manual](mixed-manual/) - Manual Override
- Custom XDR implementations
- Mixed auto-generated and manual code
- Validation and data transformation
- **Best for**: Advanced customization and legacy compatibility

### 5. [alias](alias/) - Type Aliases
- Type-safe aliases for primitive types
- Zero-allocation type conversions
- Compile-time type safety
- **Best for**: Preventing type mixing and improving code clarity

## Running Examples

Each example is self-contained with its own `go.mod` file. To run any example:

```bash
cd <example-directory>
go generate  # Generate XDR methods if needed
go run main.go
```

## Code Generation

Examples using auto-generation include a `go generate` comment:

```go
//go:generate ../../bin/xdrgen types.go
```

This integrates with Go's standard tooling and doesn't require custom build systems.

## Learning Path

1. **Start with encode-decode** - Learn XDR basics and manual operations
2. **Try autogen** - See how struct tags simplify development
3. **Explore discriminated-union** - Understand variant types for protocols
4. **Study mixed-manual** - Learn advanced customization techniques
5. **Learn alias** - Use type aliases for better type safety

## Common Patterns

- **Round-trip testing**: Always verify encode→decode→compare cycles
- **Error handling**: Check encoding/decoding errors in production code
- **Type safety**: Use struct tags and generated methods for validation
- **Performance**: Consider allocation patterns for high-throughput scenarios