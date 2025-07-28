package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(secret string, userID int, role string, duration time.Duration, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    userID,
		"role":       role,
		"exp":        time.Now().Add(duration).Unix(),
		"token_type": tokenType,         // различие между access и refresh
		"iat":        time.Now().Unix(), // issued at — доп. уникальность
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
