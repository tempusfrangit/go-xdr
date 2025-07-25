//go:build ignore

package codegen_test

//go:generate ../bin/xdrgen cross_package_alias_test.go

import (
	"github.com/tempusfrangit/go-xdr"
	altpkg "github.com/tempusfrangit/go-xdr/codegen_test/alt_pkg"
	altpkg2 "github.com/tempusfrangit/go-xdr/codegen_test/alt_pkg2"
)

// Test aliases to types in another package that have non-conforming Encode/Decode methods
type AliasToMyBytes altpkg.MyBytes     // Type definition (no =)
type AliasToMyString altpkg.MyString   // Type definition (no =)
type TrueAliasBytes = altpkg.MyBytes   // True type alias (with =)
type TrueAliasString = altpkg.MyString // True type alias (with =)

// +xdr:generate
type CrossPackageTest struct {
	ID              uint32
	Data            AliasToMyBytes                // Type definition: should resolve to []byte
	Name            AliasToMyString               // Type definition: should resolve to string
	TrueBytes       TrueAliasBytes                // True alias: should resolve to []byte
	TrueString      TrueAliasString               // True alias: should resolve to string
	Direct          altpkg.MyBytes                // Direct use of cross-package type
	Another         altpkg.AnotherString          // Should resolve to string, this does not a non-conforming Encode/Decode method
	MyInterface     MyInterface                   // Should use interface methods (has conforming Encode/Decode)
	MultiDepth      altpkg.MultiDepthAlias        // Should warn + struct (multi-depth alias)
	ExportedPrivate altpkg2.ExportedPrivateString // Should warn + struct (unknown package)
}

type MyInterface []byte

func (m *MyInterface) Encode(enc *xdr.Encoder) error {
	return nil
}

func (m *MyInterface) Decode(dec *xdr.Decoder) error {
	return nil
}

// Ensure we implement xdr.Codec
var _ xdr.Codec = (*CrossPackageTest)(nil)
var _ xdr.Codec = (*MyInterface)(nil)
