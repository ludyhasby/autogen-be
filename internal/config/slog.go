package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func NewSlog(env *Env) *slog.Logger {

	// Make dir
	_ = os.MkdirAll("log", 0o755)

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("log", "app.log"),
		MaxSize:    1,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}

	// tulis ke stdout + file berotasi
	writer := io.MultiWriter(os.Stdout, rotator)

	log := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: GetLevelLog(env.LogLevel),
	}))

	return log
}

func GetLevelLog(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
