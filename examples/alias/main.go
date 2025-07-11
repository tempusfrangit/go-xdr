package main

//go:generate go run ../../tools/xdrgen/main.go $GOFILE

import (
	"fmt"

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

// Example struct using alias types
type User struct {
	ID       UserID     `xdr:"alias:string"`
	Session  SessionID  `xdr:"alias:bytes"`
	Status   StatusCode `xdr:"alias:uint32"`
	Flags    Flags      `xdr:"alias:uint64"`
	Priority Priority   `xdr:"alias:int32"`
	Created  Timestamp  `xdr:"alias:int64"`
	Active   IsActive   `xdr:"alias:bool"`
}

// Ensure User implements Codec interface
var _ xdr.Codec = (*User)(nil)

func main() {
	// Create a user with alias types
	user := &User{
		ID:       UserID("user123"),
		Session:  SessionID{0x01, 0x02, 0x03, 0x04},
		Status:   StatusCode(200),
		Flags:    Flags(0x123456789ABCDEF0),
		Priority: Priority(-5),
		Created:  Timestamp(1234567890),
		Active:   IsActive(true),
	}

	fmt.Printf("Original user: %+v\n", user)

	// Marshal to XDR
	data, err := xdr.Marshal(user)
	if err != nil {
		panic(fmt.Sprintf("Marshal failed: %v", err))
	}

	fmt.Printf("XDR data length: %d bytes\n", len(data))

	// Unmarshal from XDR
	var decoded User
	err = xdr.Unmarshal(data, &decoded)
	if err != nil {
		panic(fmt.Sprintf("Unmarshal failed: %v", err))
	}

	fmt.Printf("Decoded user: %+v\n", decoded)

	// Verify all fields match
	if decoded.ID != user.ID {
		fmt.Printf("ID mismatch: expected %v, got %v\n", user.ID, decoded.ID)
	} else {
		fmt.Println("ID matches")
	}
	if string(decoded.Session) == string(user.Session) {
		fmt.Println("Session matches")
	} else {
		fmt.Printf("Session mismatch: expected %v, got %v\n", user.Session, decoded.Session)
	}
	if decoded.Status == user.Status {
		fmt.Println("Status matches")
	} else {
		fmt.Printf("Status mismatch: expected %v, got %v\n", user.Status, decoded.Status)
	}
	if decoded.Flags == user.Flags {
		fmt.Println("Flags match")
	} else {
		fmt.Printf("Flags mismatch: expected %v, got %v\n", user.Flags, decoded.Flags)
	}
	if decoded.Priority == user.Priority {
		fmt.Println("Priority matches")
	} else {
		fmt.Printf("Priority mismatch: expected %v, got %v\n", user.Priority, decoded.Priority)
	}
	if decoded.Created == user.Created {
		fmt.Println("Created matches")
	} else {
		fmt.Printf("Created mismatch: expected %v, got %v\n", user.Created, decoded.Created)
	}
	if decoded.Active == user.Active {
		fmt.Println("Active matches")
	} else {
		fmt.Printf("Active mismatch: expected %v, got %v\n", user.Active, decoded.Active)
	}

	fmt.Println("\nAll alias types working correctly!")
}
