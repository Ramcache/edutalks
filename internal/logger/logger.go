// internal/logger/logger.go
package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"edutalks/internal/reqctx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

type Options struct {
	Env     string // "prod" | "dev"
	Level   string // "debug" | "info" | "warn" | "error"
	Service string // опционально
}

func Init(o Options) error {
	lvl := zapcore.InfoLevel
	switch strings.ToLower(o.Level) {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		MessageKey:    "msg",
		CallerKey:     "caller",
		StacktraceKey: "stack",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.UTC().Format(time.RFC3339Nano))
		},
	}

	var enc zapcore.Encoder
	if strings.ToLower(o.Env) == "dev" {
		enc = zapcore.NewConsoleEncoder(encCfg)
	} else {
		enc = zapcore.NewJSONEncoder(encCfg)
	}

	core := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl)
	Log = zap.New(core, zap.AddCaller())
	if o.Service != "" {
		Log = Log.With(zap.String("service", o.Service))
	}
	return nil
}

// WithCtx — добавляет request_id и user_id из контекста
func WithCtx(ctx context.Context) *zap.Logger {
	l := Log
	if l == nil {
		return zap.NewNop()
	}
	if rid, ok := reqctx.GetRequestID(ctx); ok && rid != "" {
		l = l.With(zap.String("request_id", rid))
	}
	if uid, ok := reqctx.GetUserID(ctx); ok && uid != 0 {
		l = l.With(zap.Int("user_id", uid))
	}
	return l
}
