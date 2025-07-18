# Synthetic Test Module

This is a separate Go module used to test cross-module type resolution in the XDR code generator.

## Structure

- `tokenlib/` - External package defining types (`Token`, `ID`, `Hash`)  
- `consumer/` - Package that imports and uses the external types

## Purpose

Tests that the XDR generator can properly resolve types from external modules:

- `tokenlib.Token` ([]byte) → should generate bytes encoding
- `tokenlib.ID` (string) → should generate string encoding  
- `tokenlib.Hash` ([32]byte) → should generate fixed bytes encoding

This simulates the real-world scenario where downstream projects use the XDR generator with types from external dependencies.