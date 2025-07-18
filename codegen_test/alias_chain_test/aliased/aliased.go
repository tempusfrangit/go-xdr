//go:build ignore

package aliased

type Aliased []byte

func (a Aliased) Something() error {
	return nil
}
