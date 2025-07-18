package middleware

import (
	"crypgo-machine/src/application/usecase"
	"net/http"
	"strings"
)

// AuthMiddleware handles JWT authentication for protected routes
type AuthMiddleware struct {
	authUseCase *usecase.AuthUseCase
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authUseCase *usecase.AuthUseCase) *AuthMiddleware {
	return &AuthMiddleware{
		authUseCase: authUseCase,
	}
}

// RequireAuth is a middleware that requires valid JWT authentication
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Check if it starts with "Bearer "
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := tokenParts[1]

		// Validate token
		claims, err := m.authUseCase.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user email to request context for use in handlers
		r.Header.Set("X-User-Email", claims.Email)

		// Continue to next handler
		next.ServeHTTP(w, r)
	}
}

// OptionalAuth is a middleware that validates JWT if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				tokenString := tokenParts[1]
				if claims, err := m.authUseCase.ValidateToken(tokenString); err == nil {
					r.Header.Set("X-User-Email", claims.Email)
				}
			}
		}

		// Continue to next handler regardless of auth status
		next.ServeHTTP(w, r)
	}
}