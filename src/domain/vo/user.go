package vo

import (
	"errors"
	"regexp"
)

// User represents a system user with authentication credentials
type User struct {
	email    string
	password string
}

// NewUser creates a new User with validation
func NewUser(email, password string) (*User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	
	return &User{
		email:    email,
		password: password,
	}, nil
}

// GetEmail returns the user's email
func (u *User) GetEmail() string {
	return u.email
}

// GetPassword returns the user's password
func (u *User) GetPassword() string {
	return u.password
}

// validateEmail validates email format
func validateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	
	return nil
}

// validatePassword validates password strength
func validatePassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	return nil
}