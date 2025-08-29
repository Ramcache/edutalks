package middleware

import (
	"net/http"
)

func OnlyRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Фастлейн для админа — пропустить любые role-проверки
			if SkipGuards(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			value := r.Context().Value(ContextRole)
			userRole, ok := value.(string)
			if !ok || userRole != role {
				http.Error(w, "Доступ запрещён", http.StatusForbidden)
				return
			}
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
			// >>> фастлейн для админа
			if SkipGuards(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			// <<< конец фастлейна

			value := r.Context().Value(ContextRole)
			userRole, ok := value.(string)
			if !ok {
				http.Error(w, "Не удалось определить роль", http.StatusForbidden)
				return
			}
			if _, found := roleSet[userRole]; !found {
				http.Error(w, "Доступ запрещён", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
