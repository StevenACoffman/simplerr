# simpleErr/errors (aka simpler Errors with StackTraces)

`github.com/StevenACoffman/simplerr/errors` will just work as a drop in replacement to `golang.org/x/xerrors`, and post-1.13 `errors`, and provides `WithStack` as in
`pkg/errors`, but updated for the modern Go world.

### Wait, why do I want this?

If you want to add a stacktrace to an error in Go, you *might*
use the pre-Go 1.13 library `pkg/errors` and it's [WithStack](https://pkg.go.dev/github.com/pkg/errors#WithStack).

However, `pkg/errors` has not been updated since then and has a few compatibility problems with modern Go errors. Also `pkg/errors` still uses `runtime.FuncForPC` and emits file paths in the stack trace derived from the raw file name of the source file.
This method is outdated, as the on-disk file name of a given package's source may be different from the package path (due to go mod versioning, reproducible build sandboxes, etc).
The "modern" equivalent is `runtime.CallersFrame`, which populates runtime.Frame structs with package path-qualified function names.

## Example of simplerr/errors
[Checkout this example of usage](./_example):
```
$ cd _example
$ go run main.go
(1) Something went wrong
Error types: (1) main.ErrMyError
-- Stack trace:main.foo
| 	/Users/steve/Documents/git/simplerr/_example/main.go:22
| main.bar
| 	/Users/steve/Documents/git/simplerr/_example/main.go:26
| main.main
| 	/Users/steve/Documents/git/simplerr/_example/main.go:30
| runtime.main
| 	/Users/steve/.asdf/installs/golang/1.16.12/go/src/runtime/proc.go:225
```

Snazzy output, huh?