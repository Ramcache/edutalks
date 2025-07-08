package middleware

import (
	"context"
	"edutalks/internal/config"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const (
	ContextUserID ContextKey = "user_id"
	ContextRole   ContextKey = "role"
)

func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg, _ := config.LoadConfig()
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Отсутствует access token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}

		userID, ok1 := claims["user_id"].(float64)
		role, ok2 := claims["role"].(string)
		if !ok1 || !ok2 {
			http.Error(w, "Недопустимый payload", http.StatusUnauthorized)
			return
		}

		// Добавим в контекст
		ctx := context.WithValue(r.Context(), ContextUserID, int(userID))
		ctx = context.WithValue(ctx, ContextRole, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
