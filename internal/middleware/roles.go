package middleware

import (
	"net/http"

	"edutalks/internal/logger"
	"go.uber.org/zap"
)

func OnlyRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if SkipGuards(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			value := r.Context().Value(ContextRole)
			userRole, ok := value.(string)
			if !ok || userRole != role {
				logger.WithCtx(r.Context()).Warn("Доступ запрещён (OnlyRole)",
					zap.String("required_role", role), zap.Any("got", value))
				http.Error(w, "Доступ запрещён", http.StatusForbidden)
				return
			}

			logger.WithCtx(r.Context()).Info("Доступ разрешён (OnlyRole)",
				zap.String("role", userRole))
			next.ServeHTTP(w, r)
		})
	}
}

func AnyRole(allowedRoles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{})
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if SkipGuards(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			value := r.Context().Value(ContextRole)
			userRole, ok := value.(string)
			if !ok {
				logger.WithCtx(r.Context()).Warn("Роль не определена (AnyRole)")
				http.Error(w, "Не удалось определить роль", http.StatusForbidden)
				return
			}
			if _, found := roleSet[userRole]; !found {
				logger.WithCtx(r.Context()).Warn("Доступ запрещён (AnyRole)",
					zap.String("user_role", userRole), zap.Any("allowed", allowedRoles))
				http.Error(w, "Доступ запрещён", http.StatusForbidden)
				return
			}

			logger.WithCtx(r.Context()).Info("Доступ разрешён (AnyRole)",
				zap.String("role", userRole))
			next.ServeHTTP(w, r)
		})
	}
}
