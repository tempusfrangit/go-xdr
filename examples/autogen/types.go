package main

//go:generate ../../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// Type aliases for domain clarity (all auto-resolved)
type PersonID uint32
type EmailAddress string

// +xdr:generate
// Person represents a person - everything auto-detected!
type Person struct {
	ID     PersonID     // auto-detected as uint32 with casting
	Name   string       // auto-detected as string
	Age    uint32       // auto-detected as uint32
	Email  EmailAddress // auto-detected as string with casting
	Secret string       `xdr:"-"` // excluded from encoding
}

// +xdr:generate
// Company represents a company with nested structures and arrays
type Company struct {
	Name      string   // auto-detected as string
	Founded   uint32   // auto-detected as uint32
	CEO       Person   // auto-detected as struct
	Employees []Person // auto-detected as struct array
}

// More type aliases
type Port uint32
type LogLevel string

// +xdr:generate
// ServerConfig shows comprehensive auto-detection
type ServerConfig struct {
	Host       string    // auto-detected as string
	Port       Port      // auto-detected as uint32 with casting
	EnableTLS  bool      // auto-detected as bool
	MaxClients uint32    // auto-detected as uint32
	Timeout    uint64    // auto-detected as uint64
	LogLevel   LogLevel  // auto-detected as string with casting
	Features   []string  // auto-detected as string array
	Metadata   []byte    // auto-detected as bytes
	Internal   string    `xdr:"-"` // excluded
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*Person)(nil)
var _ xdr.Codec = (*Company)(nil)
var _ xdr.Codec = (*ServerConfig)(nil)
