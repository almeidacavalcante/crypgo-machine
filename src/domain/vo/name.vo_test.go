package vo

import (
	"errors"
	"testing"
)

func TestName(t *testing.T) {
	name := "JohnDoe"
	n, err := NewName(name)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if n.GetValue() != name {
		t.Errorf("expected to be %s", name)
	}
	if n == (Name{}) {
		t.Errorf("expected to be a name but got empty")
	}
}

func TestName_Invalid(t *testing.T) {
	invalid := "John Doe"
	n, err := NewName(invalid)
	if err == nil || !errors.Is(err, ErrInvalidRune) {
		t.Errorf("expected error %v, got %v", ErrInvalidRune, err)
	}
	if n != (Name{}) {
		t.Errorf("expected to be empty name")
	}
}

func TestName_Empty(t *testing.T) {
	empty := ""
	n, err := NewName(empty)
	if err == nil || !errors.Is(err, ErrInvalidName) {
		t.Errorf("expected error %v, got %v", ErrInvalidName, err)
	}
	if n != (Name{}) {
		t.Errorf("expected to be empty name")
	}
}

func TestName_TooLong(t *testing.T) {
	tooLong := "ThisNameIsWayTooLongToBeValidAndShouldReturnAnErrorBecauseItExceedsTheMaximumLength"
	n, err := NewName(tooLong)
	if err == nil || !errors.Is(err, ErrInvalidName) {
		t.Errorf("expected error %v, got %v", ErrInvalidName, err)
	}
	if n != (Name{}) {
		t.Errorf("expected to be empty name")
	}
}

func TestName_InvalidFirstRune(t *testing.T) {
	invalidFirstRune := "johnDoe"
	n, err := NewName(invalidFirstRune)
	if err == nil || !errors.Is(err, ErrInvalidUppercase) {
		t.Errorf("expected error %v, got %v", ErrInvalidUppercase, err)
	}
	if n != (Name{}) {
		t.Errorf("expected to be empty name")
	}
}

func TestName_InvalidRune(t *testing.T) {
	invalidRune := "John@Doe"
	n, err := NewName(invalidRune)
	if err == nil || !errors.Is(err, ErrInvalidRune) {
		t.Errorf("expected error %v, got %v", ErrInvalidRune, err)
	}
	if n != (Name{}) {
		t.Errorf("expected to be empty name")
	}
}
