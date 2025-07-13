//go:generate ../bin/xdrgen $GOFILE

package codegen_test

// +xdr:generate
type AutoInferenceTest struct {
    ID       uint32   // auto-detected as uint32
    Name     string   // auto-detected as string  
    Data     []byte   // auto-detected as bytes
    Count    int64    // auto-detected as int64
    Active   bool     // auto-detected as bool
    Values   []uint32 // auto-detected as uint32 array
    Hash     [16]byte // auto-detected as bytes with v.Hash[:]
    Internal string   `xdr:"-"` // excluded from encoding
}