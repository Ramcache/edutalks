// internal/logger/logger.go
package logger

import (
	"edutalks/internal/config"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("не удалось загрузить конфиг: " + err.Error())
	}

	logLevel := parseLevel(cfg.LogLevel) // выстави "debug" на проде, если хочешь видеть всё
	retentionDays := 14                  // держим 14 дней

	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		panic("не удалось создать папку для логов: " + err.Error())
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		MessageKey:    "message",
		CallerKey:     "caller",
		StacktraceKey: "stack",
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeLevel:   zapcore.CapitalLevelEncoder, // INFO/WARN/ERROR...
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	jsonEnc := zapcore.NewJSONEncoder(encCfg)

	// Ротация по дням + линк на актуальный файл logs/app.log
	rotate, err := rotatelogs.New(
		filepath.Join("logs", "app.%Y-%m-%d.log"),
		rotatelogs.WithLinkName(filepath.Join("logs", "app.log")),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(time.Duration(retentionDays)*24*time.Hour),
		rotatelogs.WithClock(rotatelogs.Local),
	)
	if err != nil {
		panic("rotatelogs init error: " + err.Error())
	}

	core := zapcore.NewCore(jsonEnc, zapcore.AddSync(rotate), logLevel)
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(Log)
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
