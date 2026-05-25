package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Authenticator defines the interface for JWT token operations.
type Authenticator interface {
	SignJoinToken(userID, roomID, role string, ttl time.Duration, displayName string) (string, error)
	ParseJoinToken(tok string) (*JoinClaims, error)
}

// JWT implements Authenticator using HMAC-SHA256.
type JWT struct {
	secret []byte
}

// JoinClaims contains the JWT claims for a join token.
type JoinClaims struct {
	Rid         string `json:"rid,omitempty"`
	Role        string `json:"role,omitempty"`
	DisplayName string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// NewJWT creates a new JWT authenticator.
func NewJWT(secret string) *JWT {
	return &JWT{secret: []byte(secret)}
}

// SignJoinToken creates a new JWT token for joining a room.
func (j *JWT) SignJoinToken(userID, roomID, role string, ttl time.Duration, displayName string) (string, error) {
	claims := JoinClaims{
		Rid:         roomID,
		Role:        role,
		DisplayName: displayName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-5 * time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.secret)
}

// ParseJoinToken parses and validates a JWT token.
func (j *JWT) ParseJoinToken(tok string) (*JoinClaims, error) {
	parsed, err := jwt.ParseWithClaims(tok, &JoinClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := parsed.Claims.(*JoinClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
