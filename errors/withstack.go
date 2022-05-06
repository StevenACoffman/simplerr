package errors

import (
	"fmt"
	"io"
	"strings"
)

// This file mirrors the WithStack functionality from
// github.com/pkg/errors.

// WithStack annotates err with a stack trace at the point WithStack was
// called.
func WithStack(err error) error {
	// Skip the frame of WithStack itself, this mirrors the behavior
	// of WithStack() in github.com/pkg/errors.
	return WithStackDepth(err, 1)
}

// WithStackDepth annotates err with a stack trace starting from the
// given call depth. The value zero identifies the caller
// of WithStackDepth itself.
// See the documentation of WithStack() for more details.
func WithStackDepth(err error, depth int) error {
	if err == nil {
		return nil
	}
	// do not re-wrap
	if _, ok := err.(*withStack); ok {
		return err
	}

	return &withStack{cause: err, Stack: Callers(depth + 1)}
}

type withStack struct {
	cause error
	*Stack
}

var (
	_ error         = (*withStack)(nil)
	_ fmt.Formatter = (*withStack)(nil)
)

func (w *withStack) Error() string { return w.cause.Error() }
func (w *withStack) Cause() error  { return w.cause }
func (w *withStack) Unwrap() error { return w.cause }

// Format implements the fmt.Formatter interface.
func (w *withStack) Format(st fmt.State, _ rune) {
	w.formatEntries(st)

	_, _ = io.WriteString(st, "\n  -- Stack trace:")
	_, _ = io.WriteString(st, strings.ReplaceAll(
		fmt.Sprintf("%+v", w.StackTrace().String()),
		"\n", string(detailSep)))
	// fmt.Fprintf(st, "\n%+v", w.StackTrace().String())
}

// formatEntries reads the entries from s.entries and produces a
// detailed rendering in s.finalBuf.
func (w *withStack) formatEntries(st fmt.State) {
	entries := getEntries(w.cause)
	if len(entries) == 0 {
		return
	}
	// The first entry at the top is special. We format it as follows:
	//
	//   (1) <details>

	_, _ = io.WriteString(st, "(1)")

	printEntry(st, entries[len(entries)-1])

	// All the entries that follow are printed as follows:
	//
	// Wraps: (N) <details>
	//
	for i, j := len(entries)-2, 2; i >= 0; i, j = i-1, j+1 {
		_, _ = fmt.Fprintf(st, "\nWraps: (%d)", j)
		entry := entries[i]
		printEntry(st, entry)
	}

	// At the end, we link all the (N) references to the Go type of the
	// error.
	_, _ = io.WriteString(st, "\nError types:")
	for i, j := len(entries)-1, 1; i >= 0; i, j = i-1, j+1 {
		_, _ = fmt.Fprintf(st, " (%d) %T", j, entries[i])
	}
}

func getEntries(err error) []error {
	var entries []error
	for err != nil {
		entries = append(entries, err)
		err = UnwrapOnce(err)
	}
	return entries
}

func printEntry(st fmt.State, entry error) {
	errString := entry.Error()
	if len(errString) > 0 {
		if !strings.HasPrefix(errString, "\n") {
			_, _ = io.WriteString(st, " ")
		}
		if len(errString) > 0 {
			_, _ = io.WriteString(st, errString)
		}
	}
	if w, ok := entry.(*withStack); ok {
		_, _ = io.WriteString(st, "\n  -- Stack trace:")
		_, _ = io.WriteString(st, strings.ReplaceAll(
			fmt.Sprintf("%+v", w.StackTrace().String()),
			"\n", string(detailSep)))
	}
}
