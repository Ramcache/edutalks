package middleware

import (
	"net/http"
	"runtime/debug"

	"edutalks/internal/logger"
	"go.uber.org/zap"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fields := []zap.Field{
					zap.Any("panic", rec),
					zap.ByteString("stack", debug.Stack()),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				}
				if rid, ok := r.Context().Value(ContextRequestID).(string); ok {
					fields = append(fields, zap.String("request_id", rid))
				}
				if userID, ok := r.Context().Value(ContextUserID).(int); ok {
					fields = append(fields, zap.Int("user_id", userID))
				}
				logger.Log.Error("panic recovered", fields...)

				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
