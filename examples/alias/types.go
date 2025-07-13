package main

//go:generate ../../tools/xdrgen/xdrgen $GOFILE

import (
	"github.com/tempusfrangit/go-xdr"
)

// Type aliases for demonstration
type UserID string
type SessionID []byte
type StatusCode uint32
type Flags uint64
type Priority int32
type Timestamp int64
type IsActive bool

// +xdr:generate
// Example struct using alias types - auto-detected with minimal tagging
type User struct {
	ID       UserID     // UserID is string alias - auto-detected
	Session  SessionID  // SessionID is []byte alias - auto-detected
	Status   StatusCode // StatusCode is uint32 alias - auto-detected
	Flags    Flags      // Flags is uint64 alias - auto-detected
	Priority Priority   // Priority is int32 alias - auto-detected
	Created  Timestamp  // Timestamp is int64 alias - auto-detected
	Active   IsActive   // IsActive is bool alias - auto-detected
}

// Ensure User implements Codec interface
var _ xdr.Codec = (*User)(nil)