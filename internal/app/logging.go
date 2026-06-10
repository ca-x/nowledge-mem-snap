package app

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func newLogger(dataDir string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NMEM_SNAP_LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	writer := io.Writer(os.Stdout)
	logFile := strings.TrimSpace(os.Getenv("NMEM_SNAP_LOG_FILE"))
	if logFile == "" {
		logFile = filepath.Join(dataDir, "logs", "nowledge-mem-snap.log")
	}
	if logFile != "stdout" && logFile != "-" {
		_ = os.MkdirAll(filepath.Dir(logFile), 0755)
		writer = io.MultiWriter(os.Stdout, &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    envInt("NMEM_SNAP_LOG_MAX_SIZE_MB", 20),
			MaxBackups: envInt("NMEM_SNAP_LOG_MAX_BACKUPS", 7),
			MaxAge:     envInt("NMEM_SNAP_LOG_MAX_AGE_DAYS", 30),
			Compress:   envBool("NMEM_SNAP_LOG_COMPRESS", true),
		})
	}
	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level}))
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
