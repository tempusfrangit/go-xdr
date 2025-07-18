//go:build ignore

package referencer

import "github.com/tempusfrangit/go-xdr/codegen_test/alias_chain_test/aliased"

//go:generate ../../../bin/xdrgen $GOFILE

type Referenced = aliased.Aliased

// +xdr:generate
type Referencer struct {
	Referenced Referenced
}
