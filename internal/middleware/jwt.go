package middleware

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/repository"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type ContextKey string

func JWTAuth(repo repository.UserRepo, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		cfg, _ := config.LoadConfig()
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.WithCtx(r.Context()).Warn("JWTAuth: отсутствует access token")
			http.Error(w, "Отсутствует access token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			logger.WithCtx(r.Context()).Warn("JWTAuth: неверный или просроченный токен",
				zap.Error(err))
			http.Error(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}

		// 🔹 Проверка блоклиста
		if blacklisted, _ := repo.IsAccessTokenBlacklisted(r.Context(), tokenString); blacklisted {
			logger.WithCtx(r.Context()).Warn("JWTAuth: токен найден в блоклисте")
			http.Error(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}

		userID, ok1 := claims["user_id"].(float64)
		role, ok2 := claims["role"].(string)
		if !ok1 || !ok2 {
			logger.WithCtx(r.Context()).Warn("JWTAuth: недопустимый payload",
				zap.Any("claims", claims))
			http.Error(w, "Недопустимый payload", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextUserID, int(userID))
		ctx = context.WithValue(ctx, ContextRole, role)

		logger.WithCtx(ctx).Info("JWTAuth: токен валиден",
			zap.Int("user_id", int(userID)), zap.String("role", role))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
