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
	var st Stack = captureStacktrace(skip)

	return &st
}

func captureStacktrace(skip int) []uintptr {
	// Unlike other "skip"-based APIs, skip=0 identifies runtime.Callers
	// itself. +2 to skip captureStacktrace and runtime.Callers.
	selfSkip := 2
	var numFrames = 64
	pcs := make([]uintptr, numFrames)
	numFrames = runtime.Callers(skip+selfSkip, pcs)
	// runtime.Callers will truncate the recorded stacktrace if there is no
	// room in the provided slice. For the full wrapper trace, keep expanding
	// storage until there are fewer frames than there is room.
	for numFrames == len(pcs) {
		pcs = make([]uintptr, len(pcs)*2)
		numFrames = runtime.Callers(skip+selfSkip, pcs)
	}
	pcs = pcs[:numFrames]

	var newPCs []uintptr
	for i := range pcs[0:numFrames] {
		newPCs = append(newPCs, pcs[i])
	}
	return newPCs
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

// StackTraceProvider is a provider of StackTraces.
// This is, intentionally, defined to be implemented by pkg/errors.stack.
type StackTraceProvider interface {
	StackTrace() StackTrace
}

// StackTrace is Stack of Frames from innermost (newest) to outermost (oldest).
type StackTrace runtime.Frames

// Next returns the next frame in the wrapper trace,
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

// stackFormatter formats a wrapper trace into a readable string representation.
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

// ElideSharedStackSuffix removes the suffix of newStack that's already
// present in prevStack. The function returns true if some entries
// were elided.
// type StackTrace []Frame -> type StackTrace runtime.Frames
// callers []uintptr
func ElideSharedStackSuffix(prevStack, newStack *Stack) (*Stack, bool) {
	if newStack == nil || prevStack == nil {
		return newStack, false
	}
	newSt := *newStack
	prevSt := *prevStack

	if len(prevSt) == 0 {
		return newStack, false
	}
	if len(newSt) == 0 {
		return newStack, false
	}

	// Skip over the common suffix.
	var i, j int
	for i, j = len(newSt)-1, len(prevSt)-1; i > 0 && j > 0; i, j = i-1, j-1 {
		if (newSt)[i] != (prevSt)[j] {
			break
		}
	}
	if i == 0 {
		// Keep at least one entry.
		i = 1
	}
	elidedStack := newSt[:i]
	return &(elidedStack), i < len((newSt))-1
}
