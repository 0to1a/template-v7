package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Short: a stolen access token can't be revoked, only waited out; the SPA refreshes on 401
const accessTokenTTL = 15 * time.Minute

var errInvalidToken = errors.New("auth: invalid token")

// Must use NewJWTManager; zero value is invalid
type JWTManager struct {
	secret []byte
	now    func() time.Time
}

func NewJWTManager(secret string) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: JWT_SECRET must be at least 32 bytes")
	}
	return &JWTManager{secret: []byte(secret), now: time.Now}, nil
}

func (m *JWTManager) Issue(publicUUID string) (string, error) {
	now := m.now()
	claims := jwt.RegisteredClaims{
		Subject:   publicUUID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing token: %w", err)
	}
	return signed, nil
}

// Returns the subject (the user's public UUID)
func (m *JWTManager) Parse(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return m.now() }),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", errInvalidToken
	}
	return claims.Subject, nil
}
