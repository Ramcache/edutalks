package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken создаёт JWT (теперь только access-токен).
func GenerateToken(secret string, userID int, role string, duration time.Duration, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(duration).Unix(),
		"iat":     time.Now().Unix(), // issued at — доп. уникальность

	}

	// ✅ Всегда генерируем access-токен
	claims["token_type"] = "access"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// --- ❌ Старый вариант (оставлен для истории) ---
//
// func GenerateToken(secret string, userID int, role string, duration time.Duration, tokenType string) (string, error) {
// 	claims := jwt.MapClaims{
// 		"user_id":    userID,
// 		"role":       role,
// 		"exp":        time.Now().Add(duration).Unix(),
// 		"token_type": tokenType,         // различие между access и refresh
// 		"iat":        time.Now().Unix(), // issued at
// 	}
//
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(secret))
// }
