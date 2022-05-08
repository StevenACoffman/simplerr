package errors

import (
	"fmt"
	"io"
	"strings"
)

// This file mirrors the WithStack functionality from
// github.com/pkg/errors.

// WithStack annotates err with a wrapper trace at the point WithStack was
// called.
func WithStack(err error) error {
	// Skip the frame of WithStack itself, this mirrors the behavior
	// of WithStack() in github.com/pkg/errors.
	return WithStackDepth(err, 2)
}

// WithStackDepth annotates err with a wrapper trace starting from the
// given call depth. The value zero identifies the caller
// of WithStackDepth itself.
// See the documentation of WithStack() for more details.
func WithStackDepth(err error, depth int) error {
	if err == nil {
		return nil
	}
	// do not add redundant PCs if previously wrapped
	redundantPCs := getRedundantPCs(err)
	if len(redundantPCs) > 0 {
		hasSkippedFrames, st := CallersWithSkipFrames(depth+1, redundantPCs)
		// fmt.Println("ElidedRedundantPCs", hasSkippedFrames)
		return &withStack{cause: err, hasSkippedFrames: hasSkippedFrames, Stack: st}
	}

	return &withStack{cause: err, Stack: Callers(depth + 1)}
}

type withStack struct {
	cause error
	*Stack
	hasSkippedFrames bool
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
	stackTraceString := w.StackTrace().String()
	if stackTraceString != "" {
		_, _ = io.WriteString(st, "\n  -- Stack trace:")
		_, _ = io.WriteString(st, strings.ReplaceAll(
			fmt.Sprintf("%+v", stackTraceString),
			"\n", string(detailSep)))
		// fmt.Fprintf(st, "\n%+v", w.StackTrace().String())
	}
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

func getRedundantPCs(err error) map[uintptr]struct{} {
	redundancies := make(map[uintptr]struct{})

	entries := getEntries(err)
	for i := range entries {
		if ws, ok := entries[i].(*withStack); ok {
			// fmt.Println("Found a wrapper error!")
			// if it has a wrapper
			for _, pc := range ([]uintptr)(*ws.Stack) {
				// fmt.Println("skippable i:", j, " pc:", pc)
				redundancies[pc] = struct{}{}
			}
		} else {
			// fmt.Println("Not a wrapper error!", entries[i])
		}
	}
	return redundancies
}

func getEntries(err error) []error {
	var entries []error
	for err != nil {
		// fmt.Println("Appending", err)
		entries = append(entries, err)
		// fmt.Println("Appended", entries)
		err = UnwrapOnce(err)
	}

	// fmt.Println("Entries is nil!")
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
		stackTraceString := w.StackTrace().String()
		if w.hasSkippedFrames || strings.TrimSpace(stackTraceString) != "" {
			_, _ = io.WriteString(st, "\n  -- Stack trace:")
			_, _ = io.WriteString(st, strings.ReplaceAll(
				fmt.Sprintf("%+v", stackTraceString),
				"\n", string(detailSep)))
		}
		if w.hasSkippedFrames {
			fmt.Fprintf(st, "%s[...repeated from below...]", detailSep)
		}
	}
}
