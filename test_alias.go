package xdr

// Test type aliases for testing
type TestUserID string
type TestSessionID []byte
type TestStatusCode uint32

// Test struct using alias types
type TestUser struct {
	ID      TestUserID     `xdr:"alias"`
	Session TestSessionID  `xdr:"alias"`
	Status  TestStatusCode `xdr:"alias"`
}
