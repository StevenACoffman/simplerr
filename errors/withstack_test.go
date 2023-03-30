package errors_test

import (
	"fmt"
	"github.com/StevenACoffman/simplerr/errors"
	"github.com/StevenACoffman/simplerr/errors/internal"
	"github.com/StevenACoffman/simplerr/errors/testutils"
	"strings"
	"testing"

	"github.com/kr/pretty"
	pkgErr "github.com/pkg/errors"
)

func TestReportableStackTrace(t *testing.T) {
	baseErr := errors.New("hello")

	t.Run("pkgErr", func(t *testing.T) {
		err := internal.Run(func() error { return pkgErr.WithStack(baseErr) })
		t.Run("local", func(t *testing.T) {
			checkStackTrace(t, err, 0)
		})
	})

	t.Run("pkgFundamental", func(t *testing.T) {
		err := internal.Run(func() error { return pkgErr.New("hello") })
		t.Run("local", func(t *testing.T) {
			checkStackTrace(t, err, 0)
		})
	})

	t.Run("withStack", func(t *testing.T) {
		err := internal.Run(func() error { return errors.WithStack(baseErr) })
		t.Run("local", func(t *testing.T) {
			checkStackTrace(t, err, 0)
		})
	})

	t.Run("withStack depth", func(t *testing.T) {
		err := internal.Run(makeErr)
		checkStackTrace(t, err, 1)
	})
	t.Run("withStack nontrival depth", func(t *testing.T) {
		err := internal.Run(makeErr3)
		checkStackTrace(t, err, 0)
	})
}

func makeErr() error  { return makeErr2() }
func makeErr2() error { return errors.WithStack(errors.New("")) }

func makeErr3() error { return makeErr4() }
func makeErr4() error { return errors.WithStackDepth(errors.New(""), 1) }

func checkStackTrace(t *testing.T, err error, expectedDepth int) {
	tt := testutils.T{T: t}

	t.Logf("looking at err %# v", pretty.Formatter(err))

	r := errors.ExtractSentryStacktrace(err)
	tt.AssertWithf(r != nil, "ExtractSentryStacktrace returned nil")

	// We're expecting the Run() functions in second position.
	tt.Assert(len(r.Frames) >= expectedDepth+2)

	for i, f := range r.Frames {
		t.Logf("frame %d:", i)
		t.Logf("absolute path: %s", f.AbsPath)
		t.Logf("file: %s", f.Filename)
		t.Logf("line: %d", f.Lineno)
		t.Logf("module: %s", f.Module)
		t.Logf("function: %s", f.Function)
	}

	// The reportable frames are in reversed order. For the test,
	// we want to look at them in the "good" order.
	for i, j := 0, len(r.Frames)-1; i < j; i, j = i+1, j-1 {
		r.Frames[i], r.Frames[j] = r.Frames[j], r.Frames[i]
	}

	for i := expectedDepth; i < expectedDepth+2; i++ {
		f := r.Frames[i]
		tt.Check(strings.Contains(f.Filename, "/errors/") ||
			strings.Contains(f.Filename, "/errors@") ||
			strings.Contains(f.Filename, "/errors_test/"),
			fmt.Sprintf("Filename contains errors: %v %+v", f.Filename, f))

		tt.Check(strings.HasSuffix(f.AbsPath, f.Filename), fmt.Sprintln("HasSuffix abspath:", f.AbsPath, "fileName:", f.Filename))

		switch i {
		case expectedDepth:
			tt.Check(strings.HasSuffix(f.Filename, "withstack_test.go"),
				fmt.Sprintln("HasSuffix", f.Filename, "withstack_test.go"))

		case expectedDepth + 1, expectedDepth + 2:
			tt.Check(strings.HasSuffix(f.Filename, "internal/run.go"),
				fmt.Sprintln("HasSuffix", f.Filename, "internal/run.go"))

			tt.Check(strings.HasSuffix(f.Module, "errors/internal"),
				fmt.Sprintln("HasSuffix", f.Module, "errors/internal"))

			tt.Check(strings.HasPrefix(f.Function, "Run"),
				fmt.Sprintln("HasSuffix", f.Function, "Run"))
		}
	}

	// Check that Run2() is after Run() in the source code.
	tt.Check(r.Frames[expectedDepth+1].Lineno != 0 &&
		r.Frames[expectedDepth+2].Lineno != 0 &&
		(r.Frames[expectedDepth+1].Lineno > r.Frames[expectedDepth+2].Lineno),
		fmt.Sprintln("expectedDepth", r.Frames[expectedDepth+1].Lineno, r.Frames[expectedDepth+2].Lineno))
}
