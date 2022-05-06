// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"

	"github.com/StevenACoffman/simplerr/errors"
)

// ErrMyError is an error that can be returned from a public API.
type ErrMyError struct {
	Msg string
}

func (e ErrMyError) Error() string {
	return e.Msg
}

func foo() error {
	// Attach stack trace to the sentinel error.
	return errors.WithStack(ErrMyError{Msg: "Something went wrong"})
}

func bar() error {
	return errors.WithStack(foo())
}

func main() {
	err := bar()
	fmt.Printf("%+v\n", err)
}
