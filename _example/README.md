# Example of simplerr.errors
```
$ go run -trimpath main.go
Doing something
----
(1) Another bad thing happened: Something went wrong
  -- Stack trace:
  | [...repeated from below...]
Wraps: (2) Another bad thing happened: Something went wrong
Wraps: (3) Something went wrong
  -- Stack trace:main.foo
  | 	command-line-arguments/main.go:22
  | main.bar
  | 	command-line-arguments/main.go:26
  | main.main
  | 	command-line-arguments/main.go:31
  | runtime.main
  | 	runtime/proc.go:225
Wraps: (4) Something went wrong
Error types: (1) *errors.withStack (2) errors.wrapper (3) *errors.withStack (4) main.ErrMyError
  -- Stack trace:main.main
  | 	command-line-arguments/main.go:33
```