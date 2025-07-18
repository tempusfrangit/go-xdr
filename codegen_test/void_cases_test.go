//go:build ignore

//go:generate ../bin/xdrgen $GOFILE

package codegen_test

type StatusCode uint32

const (
    SUCCESS StatusCode = 1
    FAILURE StatusCode = 2
    PENDING StatusCode = 3
)

// Test 1: All void cases - auto-detected, default=nil optional
// +xdr:generate
type AllVoidUnion struct {
    Status StatusCode `` // discriminant
    Data   []byte                 // auto-detected union field
    // All StatusCode constants without payload structs -> void cases
}