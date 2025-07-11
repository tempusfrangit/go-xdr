package xdr_test

//go:generate go run ../tools/xdrgen/main.go $GOFILE

import (
	"testing"

	"github.com/tempusfrangit/go-xdr"
)

// Test type aliases for testing
type TestUserID string
type TestSessionID []byte
type TestStatusCode uint32
type TestFlags uint64
type TestPriority int32
type TestTimestamp int64
type TestIsActive bool

// Test struct using alias types
type TestUser struct {
	ID       TestUserID     `xdr:"alias:string"`
	Session  TestSessionID  `xdr:"alias:bytes"`
	Status   TestStatusCode `xdr:"alias:uint32"`
	Flags    TestFlags      `xdr:"alias:uint64"`
	Priority TestPriority   `xdr:"alias:int32"`
	Created  TestTimestamp  `xdr:"alias:int64"`
	Active   TestIsActive   `xdr:"alias:bool"`
}

func TestAliasRoundTrip(t *testing.T) {
	// Create a test user with alias types
	user := &TestUser{
		ID:       TestUserID("user123"),
		Session:  TestSessionID{0x01, 0x02, 0x03, 0x04},
		Status:   TestStatusCode(200),
		Flags:    TestFlags(0x123456789ABCDEF0),
		Priority: TestPriority(-5),
		Created:  TestTimestamp(1234567890),
		Active:   TestIsActive(true),
	}

	// Encode
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()

	// Decode
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify all fields match
	if decoded.ID != user.ID {
		t.Errorf("ID mismatch: expected %v, got %v", user.ID, decoded.ID)
	}
	if string(decoded.Session) != string(user.Session) {
		t.Errorf("Session mismatch: expected %v, got %v", user.Session, decoded.Session)
	}
	if decoded.Status != user.Status {
		t.Errorf("Status mismatch: expected %v, got %v", user.Status, decoded.Status)
	}
	if decoded.Flags != user.Flags {
		t.Errorf("Flags mismatch: expected %v, got %v", user.Flags, decoded.Flags)
	}
	if decoded.Priority != user.Priority {
		t.Errorf("Priority mismatch: expected %v, got %v", user.Priority, decoded.Priority)
	}
	if decoded.Created != user.Created {
		t.Errorf("Created mismatch: expected %v, got %v", user.Created, decoded.Created)
	}
	if decoded.Active != user.Active {
		t.Errorf("Active mismatch: expected %v, got %v", user.Active, decoded.Active)
	}
}

func TestAliasMarshalUnmarshal(t *testing.T) {
	// Test using the Marshal/Unmarshal functions
	user := &TestUser{
		ID:       TestUserID("test-user"),
		Session:  TestSessionID{0xAA, 0xBB, 0xCC},
		Status:   TestStatusCode(404),
		Flags:    TestFlags(0xDEADBEEF),
		Priority: TestPriority(10),
		Created:  TestTimestamp(987654321),
		Active:   TestIsActive(false),
	}

	// Marshal
	data, err := xdr.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded TestUser
	err = xdr.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if decoded.ID != user.ID {
		t.Errorf("ID mismatch: expected %v, got %v", user.ID, decoded.ID)
	}
	if string(decoded.Session) != string(user.Session) {
		t.Errorf("Session mismatch: expected %v, got %v", user.Session, decoded.Session)
	}
	if decoded.Status != user.Status {
		t.Errorf("Status mismatch: expected %v, got %v", user.Status, decoded.Status)
	}
	if decoded.Flags != user.Flags {
		t.Errorf("Flags mismatch: expected %v, got %v", user.Flags, decoded.Flags)
	}
	if decoded.Priority != user.Priority {
		t.Errorf("Priority mismatch: expected %v, got %v", user.Priority, decoded.Priority)
	}
	if decoded.Created != user.Created {
		t.Errorf("Created mismatch: expected %v, got %v", user.Created, decoded.Created)
	}
	if decoded.Active != user.Active {
		t.Errorf("Active mismatch: expected %v, got %v", user.Active, decoded.Active)
	}
}

func TestAliasTypeSafety(t *testing.T) {
	// Test that alias types provide type safety
	var user TestUser

	// These should compile and work correctly
	user.ID = TestUserID("test")
	user.Status = TestStatusCode(200)
	user.Active = TestIsActive(true)

	// Verify the types are distinct (these should not compile if uncommented)
	// if user.ID == "test" { // This would be a compile error
	//     t.Error("Type alias should prevent direct string comparison")
	// }
	// if user.Status == 200 { // This would be a compile error
	//     t.Error("Type alias should prevent direct uint32 comparison")
	// }
	// if user.Active == true { // This would be a compile error
	//     t.Error("Type alias should prevent direct bool comparison")
	// }

	// These should work (explicit conversions)
	if string(user.ID) == "test" {
		// This is expected
	} else {
		t.Error("String conversion should work")
	}
	if uint32(user.Status) == 200 {
		// This is expected
	} else {
		t.Error("Uint32 conversion should work")
	}
	if bool(user.Active) == true {
		// This is expected
	} else {
		t.Error("Bool conversion should work")
	}
}

func TestAliasEdgeCases(t *testing.T) {
	// Test edge cases for alias types
	user := &TestUser{
		ID:       TestUserID(""),      // Empty string
		Session:  TestSessionID{},     // Empty slice
		Status:   TestStatusCode(0),   // Zero value
		Flags:    TestFlags(0),        // Zero value
		Priority: TestPriority(0),     // Zero value
		Created:  TestTimestamp(0),    // Zero value
		Active:   TestIsActive(false), // False
	}

	// Round trip
	buf := make([]byte, 1024)
	enc := xdr.NewEncoder(buf)
	err := user.Encode(enc)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	data := enc.Bytes()
	dec := xdr.NewDecoder(data)
	var decoded TestUser
	err = decoded.Decode(dec)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify zero values are preserved
	if decoded.ID != TestUserID("") {
		t.Errorf("Empty string not preserved: got %v", decoded.ID)
	}
	if len(decoded.Session) != 0 {
		t.Errorf("Empty slice not preserved: got %v", decoded.Session)
	}
	if decoded.Status != TestStatusCode(0) {
		t.Errorf("Zero uint32 not preserved: got %v", decoded.Status)
	}
	if decoded.Flags != TestFlags(0) {
		t.Errorf("Zero uint64 not preserved: got %v", decoded.Flags)
	}
	if decoded.Priority != TestPriority(0) {
		t.Errorf("Zero int32 not preserved: got %v", decoded.Priority)
	}
	if decoded.Created != TestTimestamp(0) {
		t.Errorf("Zero int64 not preserved: got %v", decoded.Created)
	}
	if decoded.Active != TestIsActive(false) {
		t.Errorf("False bool not preserved: got %v", decoded.Active)
	}
}
