package errors

import "fmt"

// Wrap wraps an error with a message prefix.
// A stack trace is retained.
func Wrap(err error, msg string) error {
	return With(err, New(msg))
}

// Wrapf wraps an error with a formatted message prefix. A stack
// trace is also retained. If the format is empty, no prefix is added,
// but the extra arguments are still processed for reportable strings.
func Wrapf(err error, format string, args ...interface{}) error {
	return With(err, New(fmt.Sprintf(format, args...)))
}
