//go:generate xdrgen $GOFILE

package xdr_test

type VoidOpCode uint32

const (
	VoidOpNone VoidOpCode = 0
	VoidOpPing VoidOpCode = 1
)

// Test container with default=nil (void-only union)
type VoidOperation struct {
	OpCode VoidOpCode `xdr:"key"`   // Operation code
	Data   []byte     `xdr:"union,default=nil"` // No payload needed
}