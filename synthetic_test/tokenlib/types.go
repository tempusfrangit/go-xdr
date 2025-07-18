package tokenlib

// Token represents an opaque authentication token
type Token []byte

// ID represents a unique identifier
type ID string

// Hash represents a cryptographic hash
type Hash [32]byte

// Some methods to make it look realistic
func (t Token) IsValid() bool {
	return len(t) > 0
}

func (id ID) String() string {
	return string(id)
}