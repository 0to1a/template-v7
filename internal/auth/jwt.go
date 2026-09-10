package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenTTL = 24 * time.Hour

var errInvalidToken = errors.New("auth: invalid token")

type Principal struct {
	PublicUUID string
}

// Must use NewJWTManager; zero value is invalid
type JWTManager struct {
	secret []byte
	now    func() time.Time
}

// NewJWTManager requires a secret of at least 32 bytes.
func NewJWTManager(secret string) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: JWT secret must be at least 32 bytes")
	}
	return &JWTManager{secret: []byte(secret), now: time.Now}, nil
}

// Reuses JWT secret as OTP key material
func (m *JWTManager) generateLoginCode(publicUUID, normalizedEmail string, now time.Time) string {
	return generateCode(m.secret, publicUUID, normalizedEmail, now)
}

// Verifies without exposing the signing secret
func (m *JWTManager) verifyLoginCode(publicUUID, normalizedEmail, code string, now time.Time) bool {
	return verifyCode(m.secret, publicUUID, normalizedEmail, code, now)
}

func (m *JWTManager) Issue(publicUUID string) (string, error) {
	now := m.now()
	claims := jwt.RegisteredClaims{
		Subject:   publicUUID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing token: %w", err)
	}
	return signed, nil
}

// Rejects alg=none; requires exp and subject
func (m *JWTManager) Parse(tokenString string) (Principal, error) {
	var claims jwt.RegisteredClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return m.now() }),
	)
	if err != nil || !token.Valid {
		return Principal{}, errInvalidToken
	}

	if claims.Subject == "" {
		return Principal{}, errInvalidToken
	}

	return Principal{PublicUUID: claims.Subject}, nil
}
