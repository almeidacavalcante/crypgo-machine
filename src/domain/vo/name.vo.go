package vo

import "errors"

var ErrInvalidName = errors.New("name must be in PascalCase and have a maximum of 50 characters")
var ErrInvalidRune = errors.New("name contains invalid characters, only letters and digits are allowed")
var ErrInvalidUppercase = errors.New("name must start with an uppercase letter")

type Name struct {
	value string
}

func NewName(val string) (Name, error) {
	err := validateName(val)
	if err != nil {
		return Name{}, err
	}
	return Name{value: val}, nil
}

func (n Name) GetValue() string {
	return n.value
}

func validateName(val string) error {
	if len(val) == 0 || len(val) > 50 {
		return ErrInvalidName
	}

	for i, r := range val {
		if i == 0 && !isUppercaseLetter(r) {
			return ErrInvalidUppercase
		}
		if !isValidRune(r) {
			return ErrInvalidRune
		}
	}

	return nil
}

func isValidRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func isUppercaseLetter(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
