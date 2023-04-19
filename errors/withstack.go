package errors

import (
	"fmt"
	"io"
	reflectlite "reflect"
	"strings"
)

// This file mirrors the WithStack functionality from
// github.com/pkg/errors.

// WithStack annotates err with a wrapper trace at the point WithStack was
// called.
func WithStack(err error) error {
	// Skip the frame of WithStack itself, this mirrors the behavior
	// of WithStack() in github.com/pkg/errors.
	return WithStackDepth(err, 1)
}

// WithStackDepth annotates err with a wrapper trace starting from the
// given call depth. The value zero identifies the caller
// of WithStackDepth itself.
// See the documentation of WithStack() for more details.
func WithStackDepth(err error, depth int) error {
	if err == nil {
		return nil
	}
	st := Callers(depth + 2)
	prevStack := getLastStack(err)
	var hasSkippedFrames bool
	st, hasSkippedFrames = ElideSharedStackSuffix(prevStack, st)
	return &withStack{cause: err, hasSkippedFrames: hasSkippedFrames, Stack: st}
}

type withStack struct {
	cause error
	*Stack
	hasSkippedFrames bool
}

// compiler enforced interface conformance checks
var (
	_ error         = (*withStack)(nil)
	_ fmt.Formatter = (*withStack)(nil)
	_ Iser          = (*withStack)(nil)
	_ Aser          = (*withStack)(nil)
	_ Unwrapper     = (*withStack)(nil)
)

func (w *withStack) Error() string { return w.cause.Error() }
func (w *withStack) Cause() error  { return w.cause }
func (w *withStack) Unwrap() error { return w.cause }

// Format implements the fmt.Formatter interface.
func (w *withStack) Format(st fmt.State, _ rune) {
	w.formatEntries(st)
	outputStackTrace(st, false, w.StackTrace().String())
}

// Is implements the interface needed for errors.Is. It checks s.front first, and
// then s.back.
func (w *withStack) Is(target error) bool {
	// This code copied exactly from errors.Is, minus the code to unwrap if the
	// check fails. Thus, it is effectively like calling errors.Is(w.front,
	// target).
	//
	// Note, if w.front doesn't match the target, errors.Is will call this
	// type'w Unwrap, which will iterate through the wrapped errors.

	if target == nil {
		return false
	}

	isComparable := reflectlite.TypeOf(target).Comparable()
	if isComparable && w.cause == target {
		return true
	}
	if x, ok := w.cause.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}

	return false
}

// As implements the interface needed for errors.As. It checks s.front first, and
// then s.back.
func (w *withStack) As(target interface{}) bool {
	// This code copied exactly from errors.As, minus the code to unwrap if the
	// check fails. Thus, it is effectively like calling errors.As(w.front,
	// target).
	//
	// Note, if w.front doesn't match the target, errors.As will call this types
	// Unwrap, which will iterate through the wrapped errors.

	if target == nil {
		panic("errors: target cannot be nil")
	}
	val := reflectlite.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflectlite.Ptr || val.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}
	targetType := typ.Elem()
	if targetType.Kind() != reflectlite.Interface && !targetType.Implements(errorType) {
		panic("errors: *target must be interface or implement error")
	}
	if reflectlite.TypeOf(w.cause).AssignableTo(targetType) {
		val.Elem().Set(reflectlite.ValueOf(w.cause))
		return true
	}
	if x, ok := w.cause.(interface{ As(interface{}) bool }); ok && x.As(target) {
		return true
	}
	return false
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

func getLastStack(err error) *Stack {
	for err != nil {
		if ws, ok := err.(*withStack); ok {
			return ws.Stack
		}
		if wf, ok := err.(*withFields); ok {
			return wf.Stack
		}
		err = UnwrapOnce(err)
	}

	return nil
}

// getEntries prepended last error in, first out
func getEntries(err error) []error {
	var entries []error
	for err != nil {
		// prepend because we want the stack last in, first out
		entries = append([]error{err}, entries...)
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
		outputStackTrace(st, w.hasSkippedFrames, w.StackTrace().String())
	}
	if w, ok := entry.(*withFields); ok {
		outputStackTrace(st, w.hasSkippedFrames, w.StackTrace().String())
	}
}

func outputStackTrace(st fmt.State, hasSkippedFrames bool, stackTraceString string) {
	if hasSkippedFrames || strings.TrimSpace(stackTraceString) != "" {
		_, _ = io.WriteString(st, "\n  -- Stack trace:")
		_, _ = io.WriteString(st, strings.ReplaceAll(
			fmt.Sprintf("%+v", stackTraceString),
			"\n", string(detailSep)))
	}
	if hasSkippedFrames {
		_, _ = fmt.Fprintf(st, "%s[...repeated from below...]", detailSep)
	}
}
