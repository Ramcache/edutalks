package middleware

import "net/http"

// ДОЛЖЕН стоять ПОСЛЕ JWTAuth, чтобы роль уже была в контексте.
func AdminFastLane(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ContextRole).(string)
		if role == "admin" {
			r = r.WithContext(WithSkipGuards(r.Context()))
		}
		next.ServeHTTP(w, r)
	})
}
