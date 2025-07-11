package main

//go:generate ../../bin/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// Person represents a person with auto-generated XDR methods
type Person struct {
	ID    uint32 `xdr:"uint32"`
	Name  string `xdr:"string"`
	Age   uint32 `xdr:"uint32"`
	Email string `xdr:"string"`
}

// Company represents a company with nested structures
type Company struct {
	Name      string   `xdr:"string"`
	Founded   uint32   `xdr:"uint32"`
	CEO       Person   `xdr:"struct"`
	Employees []Person `xdr:"array"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host       string   `xdr:"string"`
	Port       uint32   `xdr:"uint32"`
	EnableTLS  bool     `xdr:"bool"`
	MaxClients uint32   `xdr:"uint32"`
	Timeout    uint64   `xdr:"uint64"`
	LogLevel   string   `xdr:"string"`
	Features   []string `xdr:"array"`
	Metadata   []byte   `xdr:"bytes"`
}

// Ensure all types implement the Codec interface
var _ xdr.Codec = (*Person)(nil)
var _ xdr.Codec = (*Company)(nil)
var _ xdr.Codec = (*ServerConfig)(nil)
