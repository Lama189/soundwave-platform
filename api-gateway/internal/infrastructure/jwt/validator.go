package jwt

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type TokenValidator interface {
	ValidateAccessToken(tokenStr string) (*UserClaims, error)
}

type JWTValidator struct {
	secretKey []byte
}

func NewValidator(secretKey string) *JWTValidator {
	return &JWTValidator{
		secretKey: []byte(secretKey),
	}
}

func (v *JWTValidator) ValidateAccessToken(tokenStr string) (*UserClaims, error) {
	var claims UserClaims

	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Type != TokenTypeAccess {
		return nil, ErrInvalidTokenType
	}

	return &claims, nil
}
