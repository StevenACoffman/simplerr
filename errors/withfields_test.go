package errors_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/StevenACoffman/simplerr/errors"
)

func TestWithFieldsAs(t *testing.T) {
	err := myError("some pig")
	wrapped := fmt.Errorf("wilbur: %w", err)
	err2 := errors.WrapWithFields(wrapped, errors.Fields{"key": "value"})
	err3 := fmt.Errorf("more context: %w", err2)

	var my myError

	if !errors.As(err3, &my) {
		t.Fatal("failed to find original type after wrapping")
	}

	other := otherError{msg: "hi!"}

	err4 := errors.With(err3, fmt.Errorf("some other error: %w", other))

	var o otherError
	if !errors.As(err4, &o) {
		t.Fatal("failed to find flagged type")
	}
	if !errors.As(err4, &my) {
		t.Fatal("failed to original type after wrapping")
	}

	if !errors.Is(err4, err) {
		t.Fatal("failed to find original error")
	}
	if !errors.Is(err4, other) {
		t.Fatal("failed to find flagged error")
	}
}

func TestWithFieldsIs(t *testing.T) {
	err := errors.New("some pig")
	wrapped := fmt.Errorf("wilbur: %w", err)
	err2 := errors.WrapWithFields(wrapped, errors.Fields{"key": "value"})
	if !errors.Is(err2, err) {
		t.Fatal("failed to find original error")
	}
	if !errors.Is(err2, wrapped) {
		t.Fatal("failed to find wrapped error")
	}

	err3 := fmt.Errorf("more context: %w", errors.With(err2, io.EOF))

	if !errors.Is(err3, err) {
		t.Fatal("failed to find original error after wrapping")
	}
	if !errors.Is(err3, wrapped) {
		t.Fatal("failed to find wrapped error after second wrapping")
	}
	if !errors.Is(err3, io.EOF) {
		t.Fatal("failed to find flagged wrapped error after wrapping")
	}
}

func TestWithFieldsErrList(t *testing.T) {
	one := fmt.Errorf("one")
	two := errors.WrapWithFields(one, errors.Fields{"b": "true", "msg": "two"})
	three := errors.WrapWithFields(two, errors.Fields{"c": "true", "msg": "three"})
	four := errors.WrapWithFields(three, errors.Fields{"d": "true", "msg": "four"})
	five := errors.WrapWithFields(four, errors.Fields{"e": "true", "msg": "five"})
	six := errors.WrapWithFields(five, errors.Fields{"f": "true", "msg": "six"})
	actual := six.Error()
	expected := "fields:[f:true,msg:six],fields:[e:true,msg:five],fields:[d:true,msg:four],fields:[c:true,msg:three],fields:[b:true,msg:two],one"
	if actual != expected {
		t.Fatalf("expected %v but got %v", expected, actual)
	}
}

func TestWithFieldsErrAllList(t *testing.T) {
	one := fmt.Errorf("one")
	two := errors.WrapWithFields(one, errors.Fields{"b": "true", "msg": "two"})
	three := errors.WrapWithFields(two, errors.Fields{"c": "true", "msg": "three"})
	actual := fmt.Sprintf("%+v", three)
	expected := `fields:[b:true,c:true,msg:three],cause:fields:[b:true,msg:two],one
Wraps: (1) fields:[b:true,msg:two],one
  -- Stack trace:github.com/StevenACoffman/simplerr/errors_test.TestWithFieldsErrAllList
  | 	github.com/StevenACoffman/simplerr/errors/withfields_test.go:83
  | testing.tRunner
  | 	testing/testing.go:1576
Wraps: (2) one
Error types: (1) *errors.withFields (2) *errors.errorString`
	if actual != expected {
		t.Fatalf("expected:\n %v\nbut got:\n%v", expected, actual)
	}
}
