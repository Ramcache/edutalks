package middleware

import (
	"edutalks/internal/logger"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", lrw.statusCode),
			zap.Duration("duration", time.Since(start)),
		}

		if rid, ok := r.Context().Value(ContextRequestID).(string); ok {
			fields = append(fields, zap.String("request_id", rid))
		}
		if userID, ok := r.Context().Value(ContextUserID).(int); ok {
			fields = append(fields, zap.Int("user_id", userID))
		}
		if role, ok := r.Context().Value(ContextRole).(string); ok {
			fields = append(fields, zap.String("role", role))
		}

		logger.Log.Info("HTTP-запрос", fields...)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
