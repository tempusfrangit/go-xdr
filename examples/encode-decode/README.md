# Basic XDR Encode/Decode Example

This example demonstrates the fundamental XDR encoding and decoding operations using the core library primitives.

## What it shows

- Basic XDR data types (uint32, uint64, string, bytes, bool)
- Direct use of Encoder/Decoder for fine-grained control
- Manual encoding/decoding without code generation
- Round-trip validation of encoded data

## Key concepts

- **Encoder**: Provides methods to encode Go values into XDR format
- **Decoder**: Provides methods to decode XDR data back into Go values
- **XDR Format**: Network byte order (big-endian) with 4-byte alignment
- **Round-trip**: Ensure data integrity through encode→decode→compare cycles

## Running the example

```bash
go run main.go
```

## Expected output

The example encodes various data types, shows their XDR byte representation, then decodes them back to verify correctness.