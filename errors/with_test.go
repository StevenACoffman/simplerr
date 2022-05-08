package errors_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/StevenACoffman/simplerr/errors"
)

var NotFound = errors.New("not found")

func TestWithIs(t *testing.T) {
	err := errors.New("some pig")
	wrapped := fmt.Errorf("wilbur: %w", err)
	err2 := errors.With(wrapped, NotFound)
	if !errors.Is(err2, NotFound) {
		t.Fatal("failed to find flag")
	}
	if !errors.Is(err2, err) {
		t.Fatal("failed to find original error")
	}
	if !errors.Is(err2, wrapped) {
		t.Fatal("failed to find wrapped error")
	}

	err3 := fmt.Errorf("more context: %w", errors.With(err2, io.EOF))

	if !errors.Is(err3, NotFound) {
		t.Fatal("failed to find flag after wrapping")
	}
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

type myError string

func (m myError) Error() string {
	return string(m)
}

type otherError struct {
	msg string
}

func (o otherError) Error() string {
	return o.msg
}

func TestWithAs(t *testing.T) {
	err := myError("some pig")
	wrapped := fmt.Errorf("wilbur: %w", err)
	err2 := errors.With(wrapped, NotFound)
	err3 := fmt.Errorf("more context: %w", err2)

	var my myError

	if !errors.As(err3, &my) {
		t.Fatal("failed to original type after wrapping")
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

func TestErrList(t *testing.T) {
	one := errors.New("one")
	two := errors.New("two")
	three := errors.New("three")
	four := errors.New("four")
	five := errors.New("five")
	six := errors.New("six")

	fivesix := errors.With(six, five)
	threefour := errors.With(four, three)
	onetwo := errors.With(two, one)
	oneToFour := errors.With(threefour, onetwo)
	all := errors.With(fivesix, oneToFour)
	actual := all.Error()
	expected := "one: two: three: four: five: six"
	if actual != expected {
		t.Fatalf("expected %v but got %v", expected, actual)
	}
}
