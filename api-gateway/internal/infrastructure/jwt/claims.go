package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserClaims struct {
	UserID uuid.UUID `json:"sub"`
	Role   string    `json:"role"`
	Type   string    `json:"type"`
	jwt.RegisteredClaims
}

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)
