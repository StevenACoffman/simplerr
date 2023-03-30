package errors

import (
	"fmt"
	"go/build"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

const unknown string = "unknown"

// ReportableStackTrace holds information about the frames of the stack.
type ReportableStackTrace struct {
	Frames        []Frame `json:"frames,omitempty"`
	FramesOmitted []uint  `json:"frames_omitted,omitempty"`
}

// NewSentryStacktrace creates a stacktrace using runtime.Callers.
func NewSentryStacktrace(pcs []uintptr) *ReportableStackTrace {
	n := runtime.Callers(1, pcs)

	if n == 0 {
		return nil
	}

	frames := extractFrames(pcs[:n])
	frames = filterFrames(frames)

	stacktrace := ReportableStackTrace{
		Frames: frames,
	}

	return &stacktrace
}

// TODO: Make it configurable so that anyone can provide their own implementation?
// Use of reflection allows us to not have a hard dependency on any given
// package, so we don't have to import it.

// ExtractSentryStacktrace creates a new ReportableStackTrace based on the given error.
func ExtractSentryStacktrace(err error) *ReportableStackTrace {
	if st, ok := err.(*withStack); ok {
		pcs := ([]uintptr)(*st.Stack)
		return NewSentryStacktrace(pcs)
	}
	if st, ok := err.(StackTraceProvider); ok {
		return convertPkgStack(st.StackTrace())
	}

	method := extractReflectedSentryStacktraceMethod(err)

	var pcs []uintptr

	if method.IsValid() {
		pcs = extractPcs(method)
	} else {
		pcs = extractXErrorsPC(err)
	}

	if len(pcs) == 0 {
		return nil
	}

	frames := extractFrames(pcs)
	frames = filterFrames(frames)

	stacktrace := ReportableStackTrace{
		Frames: frames,
	}

	return &stacktrace
}

// convertPkgStack converts a StackTrace from github.com/pkg/errors
// to a Stacktrace in github.com/getsentry/sentry-go.
func convertPkgStack(st StackTrace) *ReportableStackTrace {
	stackTraceString := fmt.Sprintf("%+v", st)
	if stackTraceString == "" {
		return nil
	}
	// Note: the stack trace logic changed between go 1.11 and 1.12.
	// Trying to analyze the frame PCs point-wise will cause
	// the output to change between the go versions.
	return parsePrintedStack(stackTraceString)
}

// parsePrintedStack reverse-engineers a reportable stack trace from
// the result of printing a github.com/pkg/errors stack trace with format %+v.
func parsePrintedStack(st string) *ReportableStackTrace {
	// A printed stack trace looks like a repetition of either:
	// "unknown"
	// or
	// <result of fn.Name()>
	// <tab><file>:<linenum>
	// It's also likely to contain a heading newline character(s).
	var frames []Frame
	lines := strings.Split(strings.TrimSpace(st), "\n")
	for i := 0; i < len(lines); i++ {
		nextI, file, line, fnName := parsePrintedStackEntry(lines, i)
		i = nextI

		// Compose the frame.
		frame := Frame{
			AbsPath:  file,
			Filename: trimPath(file),
			Lineno:   line,
			InApp:    true,
			Module:   "unknown",
			Function: fnName,
		}
		if fnName != "unknown" {
			// Extract the function/module details.
			frame.Module, frame.Function = functionName(fnName)
		}
		frames = append(frames, frame)
	}

	if frames == nil {
		return nil
	}

	// Sentry wants the frames with the oldest first, so reverse them.
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}

	return &ReportableStackTrace{Frames: frames}
}

// trimPath is a copy of the same function in package sentry-go.
func trimPath(filename string) string {
	for _, prefix := range trimPaths {
		if trimmed := strings.TrimPrefix(filename, prefix); len(trimmed) < len(filename) {
			return trimmed
		}
	}

	return filename
}

var trimPaths []string

// init is a copy of the same function in package sentry-go.
func init() {
	// Collect all source directories, and make sure they
	// end in a trailing "separator"
	for _, prefix := range build.Default.SrcDirs() {
		if prefix[len(prefix)-1] != filepath.Separator {
			prefix += string(filepath.Separator)
		}
		trimPaths = append(trimPaths, prefix)
	}
}

// functionName is an adapted copy of the same function in package sentry-go.
func functionName(fnName string) (pack string, name string) {
	name = fnName
	// We get this:
	//	runtime/debug.*T·ptrmethod
	// and want this:
	//  pack = runtime/debug
	//	name = *T.ptrmethod
	if idx := strings.LastIndex(name, "."); idx != -1 {
		pack = name[:idx]
		name = name[idx+1:]
	}
	name = strings.ReplaceAll(name, "·", ".")

	return
}

// parsePrintedStackEntry extracts the stack entry information
// in lines at position i. It returns the new value of i if more than
// one line was read.
func parsePrintedStackEntry(
	lines []string, i int,
) (newI int, file string, line int, fnName string) {
	// The function name is on the first line.
	fnName = lines[i]

	// The file:line pair may be on the line after that.
	if i < len(lines)-1 && strings.HasPrefix(lines[i+1], "\t") {
		fileLine := strings.TrimSpace(lines[i+1])
		// Separate file path and line number.
		lineSep := strings.LastIndexByte(fileLine, ':')
		if lineSep == -1 {
			file = fileLine
		} else {
			file = fileLine[:lineSep]
			lineStr := fileLine[lineSep+1:]
			line, _ = strconv.Atoi(lineStr)
		}
		i++
	}

	return i, file, line, fnName
}

func extractReflectedSentryStacktraceMethod(err error) reflect.Value {
	var method reflect.Value

	// https://github.com/pingcap/errors
	methodGetStackTracer := reflect.ValueOf(err).MethodByName("GetStackTracer")
	// https://github.com/pkg/errors
	methodStackTrace := reflect.ValueOf(err).MethodByName("StackTrace")
	// https://github.com/go-errors/errors
	methodStackFrames := reflect.ValueOf(err).MethodByName("StackFrames")

	if methodGetStackTracer.IsValid() {
		stacktracer := methodGetStackTracer.Call(make([]reflect.Value, 0))[0]
		stacktracerStackTrace := reflect.ValueOf(stacktracer).MethodByName("StackTrace")

		if stacktracerStackTrace.IsValid() {
			method = stacktracerStackTrace
		}
	}

	if methodStackTrace.IsValid() {
		method = methodStackTrace
	}

	if methodStackFrames.IsValid() {
		method = methodStackFrames
	}

	return method
}

func extractPcs(method reflect.Value) []uintptr {
	var pcs []uintptr

	stacktrace := method.Call(make([]reflect.Value, 0))[0]

	if stacktrace.Kind() != reflect.Slice {
		return nil
	}

	for i := 0; i < stacktrace.Len(); i++ {
		pc := stacktrace.Index(i)

		switch pc.Kind() {
		case reflect.Uintptr:
			pcs = append(pcs, uintptr(pc.Uint()))
		case reflect.Struct:
			for _, fieldName := range []string{"ProgramCounter", "PC"} {
				field := pc.FieldByName(fieldName)
				if !field.IsValid() {
					continue
				}
				if field.Kind() == reflect.Uintptr {
					pcs = append(pcs, uintptr(field.Uint()))
					break
				}
			}
		}
	}

	return pcs
}

// extractXErrorsPC extracts program counters from error values compatible with
// the error types from golang.org/x/xerrors.
//
// It returns nil if err is not compatible with errors from that package or if
// no program counters are stored in err.
func extractXErrorsPC(err error) []uintptr {
	// This implementation uses the reflect package to avoid a hard dependency
	// on third-party packages.

	// We don't know if err matches the expected type. For simplicity, instead
	// of trying to account for all possible ways things can go wrong, some
	// assumptions are made and if they are violated the code will panic. We
	// recover from any panic and ignore it, returning nil.
	//nolint: errcheck
	defer func() { recover() }()

	field := reflect.ValueOf(err).Elem().FieldByName("frame") // type Frame struct{ frames [3]uintptr }
	field = field.FieldByName("frames")
	field = field.Slice(1, field.Len()) // drop first pc pointing to xerrors.New
	pc := make([]uintptr, field.Len())
	for i := 0; i < field.Len(); i++ {
		pc[i] = uintptr(field.Index(i).Uint())
	}
	return pc
}

// Frame represents a function call and it's metadata. Frames are associated
// with a ReportableStackTrace.
type Frame struct {
	Function string `json:"function,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	// Module is, despite the name, the Sentry protocol equivalent of a Go
	// package's import path.
	Module string `json:"module,omitempty"`
	// Package is not used for Go stack trace frames. In other platforms it
	// refers to a container where the Module can be found. For example, a
	// Java JAR, a .NET Assembly, or a native dynamic library.
	// It exists for completeness, allowing the construction and reporting
	// of custom event payloads.
	Package     string                 `json:"package,omitempty"`
	Filename    string                 `json:"filename,omitempty"`
	AbsPath     string                 `json:"abs_path,omitempty"`
	Lineno      int                    `json:"lineno,omitempty"`
	Colno       int                    `json:"colno,omitempty"`
	PreContext  []string               `json:"pre_context,omitempty"`
	ContextLine string                 `json:"context_line,omitempty"`
	PostContext []string               `json:"post_context,omitempty"`
	InApp       bool                   `json:"in_app,omitempty"`
	Vars        map[string]interface{} `json:"vars,omitempty"`
}

// NewFrame assembles a stacktrace frame out of runtime.Frame.
func NewFrame(f runtime.Frame) Frame {
	var abspath, relpath string
	// NOTE: f.File paths historically use forward slash as path separator even
	// on Windows, though this is not yet documented, see
	// https://golang.org/issues/3335. In any case, filepath.IsAbs can work with
	// paths with either slash or backslash on Windows.
	switch {
	case f.File == "":
		relpath = unknown
		// Leave abspath as the empty string to be omitted when serializing
		// event as JSON.
		abspath = ""
	case filepath.IsAbs(f.File):
		abspath = f.File
		// TODO: in the general case, it is not trivial to come up with a
		// "project relative" path with the data we have in run time.
		// We shall not use filepath.Base because it creates ambiguous paths and
		// affects the "Suspect Commits" feature.
		// For now, leave relpath empty to be omitted when serializing the event
		// as JSON. Improve this later.
		relpath = ""
	default:
		// f.File is a relative path. This may happen when the binary is built
		// with the -trimpath flag.
		relpath = f.File
		// Omit abspath when serializing the event as JSON.
		abspath = ""
	}

	function := f.Function
	var pkg string

	if function != "" {
		pkg, function = splitQualifiedFunctionName(function)
	}

	frame := Frame{
		AbsPath:  abspath,
		Filename: relpath,
		Lineno:   f.Line,
		Module:   pkg,
		Function: function,
	}

	frame.InApp = isInAppFrame(frame)

	return frame
}

// splitQualifiedFunctionName splits a package path-qualified function name into
// package name and function name. Such qualified names are found in
// runtime.Frame.Function values.
func splitQualifiedFunctionName(name string) (pkg string, fun string) {
	pkg = packageName(name)
	fun = strings.TrimPrefix(name, pkg+".")
	return
}

func extractFrames(pcs []uintptr) []Frame {
	var frames []Frame
	callersFrames := runtime.CallersFrames(pcs)

	for {
		callerFrame, more := callersFrames.Next()

		frames = append([]Frame{
			NewFrame(callerFrame),
		}, frames...)

		if !more {
			break
		}
	}

	return frames
}

// filterFrames filters out stack frames that are not meant to be reported to
// Sentry. Those are frames internal to the SDK or Go.
func filterFrames(frames []Frame) []Frame {
	if len(frames) == 0 {
		return nil
	}

	filteredFrames := make([]Frame, 0, len(frames))

	for _, frame := range frames {
		// Skip Go internal frames.
		if frame.Module == "runtime" || frame.Module == "testing" {
			continue
		}
		// Skip Sentry internal frames, except for frames in _test packages (for
		// testing).
		if strings.HasPrefix(frame.Module, "github.com/getsentry/sentry-go") &&
			!strings.HasSuffix(frame.Module, "_test") {
			continue
		}
		filteredFrames = append(filteredFrames, frame)
	}

	return filteredFrames
}

func isInAppFrame(frame Frame) bool {
	if strings.HasPrefix(frame.AbsPath, build.Default.GOROOT) ||
		strings.Contains(frame.Module, "vendor") ||
		strings.Contains(frame.Module, "third_party") {
		return false
	}

	return true
}

func callerFunctionName() string {
	pcs := make([]uintptr, 1)
	runtime.Callers(3, pcs)
	callersFrames := runtime.CallersFrames(pcs)
	callerFrame, _ := callersFrames.Next()
	return baseName(callerFrame.Function)
}

// packageName returns the package part of the symbol name, or the empty string
// if there is none.
// It replicates https://golang.org/pkg/debug/gosym/#Sym.PackageName, avoiding a
// dependency on debug/gosym.
func packageName(name string) string {
	// A prefix of "type." and "go." is a compiler-generated symbol that doesn't belong to any package.
	// See variable reservedimports in cmd/compile/internal/gc/subr.go
	if strings.HasPrefix(name, "go.") || strings.HasPrefix(name, "type.") {
		return ""
	}

	pathend := strings.LastIndex(name, "/")
	if pathend < 0 {
		pathend = 0
	}

	if i := strings.Index(name[pathend:], "."); i != -1 {
		return name[:pathend+i]
	}
	return ""
}

// baseName returns the symbol name without the package or receiver name.
// It replicates https://golang.org/pkg/debug/gosym/#Sym.BaseName, avoiding a
// dependency on debug/gosym.
func baseName(name string) string {
	if i := strings.LastIndex(name, "."); i != -1 {
		return name[i+1:]
	}
	return name
}
