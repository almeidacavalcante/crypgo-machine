package usecase

import (
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/auth"
	"crypto/subtle"
	"errors"
)

// AuthUseCase handles authentication operations
type AuthUseCase struct {
	jwtService     *auth.JWTService
	validEmail     string
	validPassword  string
}

// LoginInput represents login request data
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginOutput represents login response data
type LoginOutput struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// NewAuthUseCase creates a new authentication use case
func NewAuthUseCase(jwtService *auth.JWTService, validEmail, validPassword string) *AuthUseCase {
	return &AuthUseCase{
		jwtService:     jwtService,
		validEmail:     validEmail,
		validPassword:  validPassword,
	}
}

// Login authenticates user and returns JWT token
func (a *AuthUseCase) Login(input LoginInput) (*LoginOutput, error) {
	// Validate input
	user, err := vo.NewUser(input.Email, input.Password)
	if err != nil {
		return nil, errors.New("invalid credentials format")
	}

	// Check credentials (constant-time comparison to prevent timing attacks)
	emailMatch := subtle.ConstantTimeCompare([]byte(user.GetEmail()), []byte(a.validEmail)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(user.GetPassword()), []byte(a.validPassword)) == 1

	if !emailMatch || !passwordMatch {
		return nil, errors.New("invalid email or password")
	}

	// Generate token
	token, err := a.jwtService.GenerateToken(user.GetEmail())
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &LoginOutput{
		Token: token,
		Email: user.GetEmail(),
	}, nil
}

// ValidateToken validates a JWT token
func (a *AuthUseCase) ValidateToken(tokenString string) (*auth.Claims, error) {
	return a.jwtService.ValidateToken(tokenString)
}

// RefreshToken refreshes an existing JWT token
func (a *AuthUseCase) RefreshToken(tokenString string) (string, error) {
	return a.jwtService.RefreshToken(tokenString)
}