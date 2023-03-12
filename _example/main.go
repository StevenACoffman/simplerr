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
	withErr := errors.With(foo(), errors.New("Another bad thing happened"))
	return errors.WithStack(withErr)
}

func baz() error {
	withErr := errors.With(foo(), errors.New("Yet another bad thing happened"))
	return errors.WithStack(withErr)
}

func main() {
	myErr := bar()
	fmt.Println("Doing something")
	err := errors.WithStack(myErr)

	myOtherErr := errors.WithStack(baz())

	joinErr := errors.Join(err, myOtherErr)
	fmt.Println("\n\n---withStackBar")
	fmt.Printf("%+v\n", err)
	fmt.Println("\n\n---baz")
	fmt.Printf("%+v\n", myOtherErr)
	fmt.Println("\n\n---joinErr")
	fmt.Printf("%+v\n", joinErr)
}
