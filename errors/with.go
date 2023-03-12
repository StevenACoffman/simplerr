package errors

import (
	"errors"
	// reflectlite is a package internal to the stdlib, but its API is the same
	// as reflect. This renaming keeps the code below identical to that in the
	// internals of the errors package.
	reflectlite "reflect"
)

// With returns an error that represents front wrapped over back. If back is
// nil, the returned error is nil.
//
// Calling Unwrap in a loop on this error will iteratively unwrap the front
// error first, until it runs out of wrapped errors, and then return the back
// error. This is also the order that Is and As will read the wrapped errors.
//
// The returned error's message will be the concatenation of the two error strings.
func With(back, front error) error {
	if back == nil {
		return nil
	}
	if front == nil {
		return back
	}

	return &wrapper{front: front, back: back}
}

// wrapper represents a wrapped stack of errors.
type wrapper struct {
	front error
	back  error
}

// Is implements the interface needed for errors.Is. It checks s.front first, and
// then s.back.
func (s *wrapper) Is(target error) bool {
	// This code copied exactly from errors.Is, minus the code to unwrap if the
	// check fails. Thus, it is effectively like calling errors.Is(s.front,
	// target).
	//
	// Note, if s.front doesn't match the target, errors.Is will call this
	// type's Unwrap, which will iterate through the wrapped errors.

	if target == nil {
		return false
	}

	isComparable := reflectlite.TypeOf(target).Comparable()
	if isComparable && s.front == target {
		return true
	}
	if x, ok := s.front.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}

	if x, ok := s.back.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}
	for inner := s.Unwrap(); inner != nil; inner = UnwrapOnce(inner) {
		if errors.Is(inner, target) {
			return true
		}
	}

	return false
}

// As implements the interface needed for errors.As. It checks s.front first, and
// then s.back.
func (s *wrapper) As(target any) bool {
	// This code copied exactly from errors.As, minus the code to unwrap if the
	// check fails. Thus, it is effectively like calling errors.As(s.front,
	// target).
	//
	// Note, if s.front doesn't match the target, errors.As will call this types
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
	if reflectlite.TypeOf(s.front).AssignableTo(targetType) {
		val.Elem().Set(reflectlite.ValueOf(s.front))
		return true
	}
	if x, ok := s.front.(interface{ As(any) bool }); ok && x.As(target) {
		return true
	}
	for inner := s.Unwrap(); inner != nil; inner = UnwrapOnce(inner) {
		if errors.As(inner, target) {
			return true
		}
	}

	return false
}

// Unwrap iteratively unwraps the error wrapper in front until it runs, out of
// wrapped errors, and then returns the back error.
func (s *wrapper) Unwrap() error {
	if err := errors.Unwrap(s.front); err != nil {
		// return a new wrapper with the unwrapped err as front, so that we
		// support unwrapping all of front and then moving on to back.
		return &wrapper{front: err, back: s.back}
	}
	// Otherwise we ran out of errors in front to unwrap, so return the
	// underlying error.
	return s.back
}

// Error returns the two concatenated error strings, separated by a colon if
// they are both non-empty.
func (s *wrapper) Error() string {
	front := s.front.Error()
	back := s.back.Error()
	if front == "" {
		return back
	}
	if back == "" {
		return front
	}
	return front + ": " + back
}
