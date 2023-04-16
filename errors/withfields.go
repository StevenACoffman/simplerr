package errors

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WithFields is our wrapper type.
type withFields struct {
	fields Fields
	cause  error
	*Stack
	hasSkippedFrames bool
}

type Fields map[string]interface{}

// WrapWithFields adds fields to an existing error.
func WrapWithFields(err error, fields Fields) error {
	return WrapWithFieldsAndDepth(err, fields, 1)
}

func WrapWithFieldsAndDepth(err error, fields Fields, depth int) error {
	if err == nil {
		return nil
	}
	// wrap with no fields does not wrap
	if fields == nil {
		return WithStackDepth(err, depth+1)
	}
	st := Callers(depth + 2)
	prevStack := getLastStack(err)
	var hasSkippedFrames bool
	st, hasSkippedFrames = ElideSharedStackSuffix(prevStack, st)

	return &withFields{cause: err, hasSkippedFrames: hasSkippedFrames, Stack: st, fields: fields}
}

var (
	_ error         = (*withFields)(nil)
	_ fmt.Formatter = (*withFields)(nil)
)

// it's an error.
func (w *withFields) Error() string { return w.formatFields() + " " + w.cause.Error() }

// Format implements the fmt.Formatter interface.
func (w *withFields) Format(st fmt.State, _ rune) {
	s := w.formatFields()
	// TODO(steve): this is maybe wrong?
	_, _ = fmt.Fprint(st, s)
	_, _ = fmt.Fprint(st, "cause: ", w.cause.Error(), "\n")

	w.formatEntries(st)
	stackTraceString := w.StackTrace().String()
	if stackTraceString != "" {
		_, _ = io.WriteString(st, "\n  -- Stack trace:")
		_, _ = io.WriteString(st, strings.ReplaceAll(
			fmt.Sprintf("%+v", stackTraceString),
			"\n", string(detailSep)))
	}
}

// formatEntries reads the entries from s.entries and produces a
// detailed rendering in s.finalBuf.
func (w *withFields) formatEntries(st fmt.State) {
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

func (w *withFields) getFields() Fields {
	result := Fields{}

	// getEntries returns prepended last error in, first out
	entries := getEntries(w)

	for _, err := range entries {
		var tmpErr withFields
		if As(err, &tmpErr) {
			for k, v := range tmpErr.fields {
				result[k] = v
			}
		}
	}
	for k, v := range w.fields {
		result[k] = v
	}
	return result
}

// GetFields retrieves the Fields from a stack of causes,
// combines them such that only the last key value pair wins.
func GetFields(err error) Fields {
	var tmpErr withFields
	if As(err, &tmpErr) {
		return tmpErr.getFields()
	}
	// TODO(steve): Hmm... nil? Not sure which is preferable
	return Fields{}
}

func (w *withFields) formatFields() string {
	var sb strings.Builder
	if w.fields != nil && len(w.fields) != 0 {
		var empty string
		_, _ = sb.WriteString("fields: [")

		keys := make([]string, 0, len(w.fields))
		for k := range w.fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			v := w.fields[k]
			eq := empty
			var val interface{} = empty
			if i > 0 {
				_, _ = sb.WriteString(",")
			}
			if v != nil {
				if len(k) > 1 {
					eq = ":"
				}
				val = v
			}

			_, _ = sb.WriteString(fmt.Sprintf("%s%s%v", k, eq, val))
		}

		_, _ = sb.WriteString("], ")
	}
	return sb.String()
}
