package errors

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
)

// Callers mirrors the code in github.com/pkg/errors,
// but makes the skip depth customizable.
func Callers(skip int) *Stack {
	const numFrames = 64
	var pcs [numFrames]uintptr
	n := runtime.Callers(2+skip, pcs[:])
	var st Stack = pcs[0:n]

	return &st
}

// Stack represents a Stack of program counters. This mirrors the
// (non-exported) type of the same name in github.com/pkg/errors.
type Stack []uintptr

// Format mirrors the code in github.com/pkg/errors.
func (s *Stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = fmt.Fprintf(st, "\n%+v", s.StackTrace().String())
	}
}

// StackTrace mirrors the code in github.com/pkg/errors.
func (s *Stack) StackTrace() *StackTrace {
	pcs := []uintptr(*s)

	return (*StackTrace)(runtime.CallersFrames(pcs))
}

// StackTrace is Stack of Frames from innermost (newest) to outermost (oldest).
type StackTrace runtime.Frames

// Next returns the next frame in the stack trace,
// and a boolean indicating whether there are more after it.
func (st *StackTrace) Next() (_ runtime.Frame, more bool) {
	return (*runtime.Frames)(st).Next()
}

func (st *StackTrace) String() string {
	buffer := bytes.Buffer{}
	defer buffer.Reset()

	stackFmt := newStackTraceFormatter(&buffer)
	stackFmt.FormatStack(st)
	return buffer.String()
}

// stackFormatter formats a stack trace into a readable string representation.
type stackTraceFormatter struct {
	b        *bytes.Buffer
	nonEmpty bool // whether we've written at least one frame already
}

// newStackFormatter builds a new stackFormatter.
func newStackTraceFormatter(b *bytes.Buffer) stackTraceFormatter {
	return stackTraceFormatter{b: b}
}

// FormatStack formats all remaining frames in the provided stacktrace -- minus
// the final runtime.main/runtime.goexit frame.
func (sf *stackTraceFormatter) FormatStack(stack *StackTrace) {
	// Note: On the last iteration, frames.Next() returns false, with a valid
	// frame, but we ignore this frame. The last frame is a runtime frame which
	// adds noise, since it's only either runtime.main or runtime.goexit.
	for frame, more := stack.Next(); more; frame, more = stack.Next() {
		sf.FormatFrame(frame)
	}
}

var detailSep = []byte("\n  | ")

// FormatFrame formats the given frame.
func (sf *stackTraceFormatter) FormatFrame(frame runtime.Frame) {
	if sf.nonEmpty {
		sf.b.WriteRune('\n')
	}
	sf.nonEmpty = true
	sf.b.WriteString(frame.Function)
	sf.b.WriteRune('\n')
	sf.b.WriteRune('\t')
	sf.b.WriteString(frame.File)
	sf.b.WriteRune(':')
	sf.b.WriteString(strconv.Itoa(frame.Line))
}
