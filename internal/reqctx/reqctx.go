// internal/reqctx/reqctx.go
package reqctx

import "context"

type key int

const (
	keyRequestID key = iota
	keyUserID
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyRequestID, id)
}

func GetRequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyRequestID).(string)
	return v, ok
}

func WithUserID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

func GetUserID(ctx context.Context) (int, bool) {
	v, ok := ctx.Value(keyUserID).(int)
	return v, ok
}
