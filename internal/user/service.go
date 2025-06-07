package user

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nihsioK/go-kanban/internal/app"
)

func GenerateToken(secret []byte, username, id string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &app.Claims{
		Username: username,
		ID:       id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
