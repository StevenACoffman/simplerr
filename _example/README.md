# Example of simplerr.errors
```
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