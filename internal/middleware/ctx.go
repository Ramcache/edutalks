package middleware

import "context"

type ctxKey string

const (
	// ЭТОТ ФЛАГ будем ставить админам, чтобы пропускать все проверки
	ContextSkipGuards ctxKey = "skip_guards"

	// Предполагается, что эти ключи уже есть в jwt.go — ничего не меняй:
	// ContextUserID ctxKey = "user_id"
	// ContextRole   ctxKey = "role"
)

func WithSkipGuards(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextSkipGuards, true)
}

func SkipGuards(ctx context.Context) bool {
	v := ctx.Value(ContextSkipGuards)
	b, _ := v.(bool)
	return b
}
