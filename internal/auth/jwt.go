package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds JWT payload for authenticated users.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Service handles token issuance and validation.
type Service struct {
	secret   []byte
	username string
	password string
}

// NewService creates an auth service for single-user mode.
func NewService(secret, username, password string) *Service {
	return &Service{
		secret:   []byte(secret),
		username: username,
		password: password,
	}
}

// AuthEnabled returns true when a password is configured.
func (s *Service) AuthEnabled() bool {
	return s.password != ""
}

// AdminUsername returns the configured admin username.
func (s *Service) AdminUsername() string {
	return s.username
}

// Authenticate validates credentials and returns a JWT token.
func (s *Service) Authenticate(username, password string) (string, error) {
	if !s.AuthEnabled() {
		return "", errors.New("authentication is disabled")
	}
	if username != s.username || password != s.password {
		return "", errors.New("invalid credentials")
	}
	return s.IssueToken(username)
}

// IssueToken creates a signed JWT for the given username.
func (s *Service) IssueToken(username string) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			Issuer:    "surge-host",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken parses and validates a bearer token.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ValidateBasic validates HTTP Basic Auth credentials.
func (s *Service) ValidateBasic(username, password string) bool {
	if !s.AuthEnabled() {
		return false
	}
	return username == s.username && password == s.password
}