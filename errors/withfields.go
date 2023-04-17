package errors

import (
	"fmt"
	"io"
	reflectlite "reflect"
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

type Fields map[string]any

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

// compiler enforced interface conformance checks
var (
	_ error         = (*withFields)(nil)
	_ fmt.Formatter = (*withFields)(nil)
	_ Iser          = (*withFields)(nil)
	_ Aser          = (*withFields)(nil)
	_ Unwrapper     = (*withFields)(nil)
)

// Error conforms to the error interface by returning a string representation
// The returned value is top level error's fields concatenated with cause.Error()
func (w *withFields) Error() string { return w.formatFields() + w.cause.Error() }

// Unwrap returns the underlying cause
func (w *withFields) Unwrap() error { return w.cause }
func (w *withFields) Cause() error  { return w.cause }

// Format implements the fmt.Formatter interface.
func (w *withFields) Format(st fmt.State, _ rune) {
	s := w.formatAllFields()
	_, _ = fmt.Fprint(st, s)
	_, _ = fmt.Fprint(st, "cause:", w.cause.Error(), "\nWraps: ")

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

// getFields returns the fields of this error and any wrapped error
// for key collisions, the outermost error's field wins
func (w *withFields) getFields() Fields {
	result := Fields{}
	// getEntries returns prepended last error in, first out
	entries := getEntries(w)
	for _, err := range entries {
		var tmpErr *withFields
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

// GetFields retrieves any Fields from a stack of causes,
// combines them such that the last key value pair wins.
func GetFields(err error) Fields {
	var tmpErr *withFields
	if As(err, &tmpErr) {
		return tmpErr.getFields()
	}
	// TODO(steve): Hmm... nil? Not sure which is preferable
	return Fields{}
}

func (w *withFields) formatFields() string {
	return formatFields(w.fields)
}

func formatFields(fields Fields) string {
	if len(fields) == 0 {
		return ""
	}
	var sb strings.Builder
	var empty string
	_, _ = sb.WriteString("fields:[")

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		v := fields[k]
		eq := empty
		var val any = empty
		if i > 0 {
			_, _ = sb.WriteString(",")
		}
		if v != nil {
			eq = ":"
			val = v
		}

		_, _ = sb.WriteString(fmt.Sprintf("%s%s%v", k, eq, val))
	}

	_, _ = sb.WriteString("],")

	return sb.String()
}

func (w *withFields) formatAllFields() string {
	return formatFields(w.getFields())
}

// Is implements the interface needed for errors.Is. It checks s.front first, and
// then s.back.
func (w *withFields) Is(target error) bool {
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
func (w *withFields) As(target any) bool {
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
	if x, ok := w.cause.(interface{ As(any) bool }); ok && x.As(target) {
		return true
	}
	return false
}
