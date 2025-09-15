package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type dailyWriteSyncer struct {
	mu     sync.Mutex
	dir    string
	file   *os.File
	curDay string
}

func newDailyWriteSyncer(dir string) (*dailyWriteSyncer, error) {
	ws := &dailyWriteSyncer{dir: dir}
	if err := ws.rotate(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (w *dailyWriteSyncer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if today != w.curDay {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

func (w *dailyWriteSyncer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

func (w *dailyWriteSyncer) rotate() error {
	if w.file != nil {
		_ = w.file.Close()
	}
	w.curDay = time.Now().Format("2006-01-02")
	if err := os.MkdirAll(w.dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(w.dir, fmt.Sprintf("app.%s.log", w.curDay))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
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
		MessageKey:    "msg",
		CallerKey:     "caller",
		StacktraceKey: "stack",
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(time.RFC3339Nano))
		},
	}

	var enc zapcore.Encoder
	if strings.ToLower(o.Env) == "dev" {
		enc = zapcore.NewConsoleEncoder(encCfg)
	} else {
		enc = zapcore.NewJSONEncoder(encCfg)
	}

	// stdout core
	consoleCore := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl)

	// daily file core
	ws, err := newDailyWriteSyncer("logs")
	if err != nil {
		return err
	}
	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(ws), lvl)

	core := zapcore.NewTee(consoleCore, fileCore)

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
