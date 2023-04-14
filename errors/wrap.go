package errors

import "fmt"

// Wrap wraps an error with a message prefix.
// A stack trace is retained.
func Wrap(err error, msg string) error {
	return With(err, New(msg))
}

func Wrapf(err error, format string, args ...interface{}) error {
	return With(err, New(fmt.Sprintf(format, args...)))
}
