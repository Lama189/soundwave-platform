package jwt

import "errors"

var (
	ErrInvalidToken     = errors.New("invalid or malformed token")
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("unevpected toket type")
)
