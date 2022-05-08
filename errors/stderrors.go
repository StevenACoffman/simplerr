package errors

// this file is for compatibility with both:
// + stdlib errors package after Go 1.13
// + pkg/errors
// + golang.org/x/xerrors
//
// this allows this package to be a drop in replacement
// for all three

import (
	"fmt"
	// reflectlite is a package internal to the stdlib, but its API is the same
	// as reflect. This rename keeps the code below identical to that in the
	// internals of the errors package.
	reflectlite "reflect"

	stderrs "errors"
)

// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
func New(text string) error {
	return stderrs.New(text)
}

// Cause aliases UnwrapAll() for compatibility with github.com/pkg/errors.
func Cause(err error) error { return UnwrapAll(err) }

// Unwrap aliases UnwrapOnce() for compatibility with xerrors.
func Unwrap(err error) error { return UnwrapOnce(err) }

// As finds the first error in err's chain that matches the type to which
// target points, and if so, sets the target to its value and returns true.
// An error matches a type if it is assignable to the target type, or if it
// has a method As(interface{}) bool such that As(target) returns true. As
// will panic if target is not a non-nil pointer to a type which implements
// error or is of interface type.
//
// The As method should set the target to its value and return true if err
// matches the type to which target points.
//
// Note: this implementation differs from that of xerrors as follows:
// - it also supports recursing through causes with Cause().
// - if it detects an API use error, its panic object is a valid error.

// As finds the first error in err's chain that matches the type to which
// target points, and if so, sets the target to its value and returns true.
// An error matches a type if it is assignable to the target type, or if it
// has a method As(interface{}) bool such that As(target) returns true. As
// will panic if target is not a non-nil pointer to a type which implements
// error or is of interface type.
//
// The As method should set the target to its value and return true if err
// matches the type to which target points.
//
// Note: this implementation differs from that of xerrors as follows:
// - it also supports recursing through causes with Cause().
// - if it detects an API use error, its panic object is a valid error.
func As(err error, target interface{}) bool {
	if target == nil {
		panic(fmt.Errorf("errors.As: target cannot be nil"))
	}

	// We use introspection for now, of course when/if Go gets generics
	// all this can go away.
	val := reflectlite.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflectlite.Ptr || val.IsNil() {
		panic(fmt.Errorf("errors.As: target must be a non-nil pointer, found %T", target))
	}
	if e := typ.Elem(); e.Kind() != reflectlite.Interface && !e.Implements(errorType) {
		panic(fmt.Errorf("errors.As: *target must be interface or implement error, found %T", target))
	}

	targetType := typ.Elem()
	for c := err; c != nil; c = UnwrapOnce(c) {
		if reflectlite.TypeOf(c).AssignableTo(targetType) {
			val.Elem().Set(reflectlite.ValueOf(c))

			return true
		}
		if x, ok := c.(interface{ As(interface{}) bool }); ok && x.As(target) {
			return true
		}
	}

	return false
}

var errorType = reflectlite.TypeOf((*error)(nil)).Elem()

// Is determines whether one of the causes of the given error or any
// of its causes is equivalent to some reference error.
//
// As in the Go standard library, an error is considered to match a
// reference error if it is equal to that target or if it implements a
// method Is(error) bool such that Is(reference) returns true.
//
// Note: the inverse is not true - making an Is(reference) method
// return false does not imply that errors.Is() also returns
// false. Errors can be equal because their network equality marker is
// the same. To force errors to appear different to Is(), use
// errors.Mark().
//
// Note: if any of the error types has been migrated from a previous
// package location or a different type, ensure that
// RegisterTypeMigration() was called prior to Is().
// Is determines whether one of the causes of the given error or any
// of its causes is equivalent to some reference error.
//
// As in the Go standard library, an error is considered to match a
// reference error if it is equal to that target or if it implements a
// method Is(error) bool such that Is(reference) returns true.
//
// Note: the inverse is not true - making an Is(reference) method
// return false does not imply that errors.Is() also returns
// false. Errors can be equal because their network equality marker is
// the same. To force errors to appear different to Is(), use
// errors.Mark().
//
// Note: if any of the error types has been migrated from a previous
// package location or a different type, ensure that
// RegisterTypeMigration() was called prior to Is().
func Is(err, reference error) bool {
	if reference == nil {
		return err == nil
	}

	// Direct reference comparison is the fastest, and most
	// likely to be true, so do this first.
	for c := err; c != nil; c = UnwrapOnce(c) {
		if equal(c, reference) {
			return true
		}
		// Compatibility with std go errors: if the error object itself
		// implements Is(), try to use that.
		if tryDelegateToIsMethod(c, reference) {
			return true
		}
	}

	if err == nil {
		// Err is nil and reference is non-nil, so it cannot match. We
		// want to short-circuit the loop below in this case, otherwise
		// we're paying the expense of getMark() without need.
		return false
	}

	// Not directly equal.
	return false
}

// This is only extracted to make the linters not suggest fixing it
func equal(err, reference interface{}) bool {
	return err == reference
}

func tryDelegateToIsMethod(err, reference error) bool {
	if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(reference) {
		return true
	}

	return false
}
