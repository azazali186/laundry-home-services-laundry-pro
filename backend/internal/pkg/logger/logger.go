// Package logger provides a thin, leveled logging facade. It forwards to
// Hertz's hlog for human-friendly text output, or emits structured JSON
// lines when LOG_FORMAT=json so logs are machine-parseable in production.
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// Level is a log severity.
type Level int

// Severity levels in ascending order.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Options configures the logger.
type Options struct {
	Level string // debug | info | warn | error
	JSON  bool   // emit structured JSON lines
}

var (
	mu      sync.Mutex
	level   = LevelInfo
	jsonOut bool
)

// Init configures the global logger from the given options.
func Init(opts Options) {
	mu.Lock()
	defer mu.Unlock()
	level = parseLevel(opts.Level)
	jsonOut = opts.JSON
	// Keep Hertz's internal logs (startup, transport) on the same level.
	hlog.SetLevel(hlogLevel(level))
	hlog.SetOutput(os.Stdout)
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func hlogLevel(l Level) hlog.Level {
	switch l {
	case LevelDebug:
		return hlog.LevelDebug
	case LevelWarn:
		return hlog.LevelWarn
	case LevelError:
		return hlog.LevelError
	default:
		return hlog.LevelInfo
	}
}

func enabled(l Level) bool {
	mu.Lock()
	defer mu.Unlock()
	return l >= level
}

// emit writes a structured JSON line. Formatting is identical for all
// severities so parsers get a consistent schema.
func emit(l Level, name, format string, v ...any) {
	if !enabled(l) {
		return
	}
	entry := map[string]any{
		"time":  time.Now().UTC().Format(time.RFC3339Nano),
		"level": name,
		"msg":   fmt.Sprintf(format, v...),
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintln(os.Stdout, string(raw))
}

// Debug logs at debug level.
func Debug(format string, v ...any) {
	if jsonOut {
		emit(LevelDebug, "debug", format, v...)
		return
	}
	hlog.Debugf(format, v...)
}

// Info logs at info level.
func Info(format string, v ...any) {
	if jsonOut {
		emit(LevelInfo, "info", format, v...)
		return
	}
	hlog.Infof(format, v...)
}

// Warn logs at warn level.
func Warn(format string, v ...any) {
	if jsonOut {
		emit(LevelWarn, "warn", format, v...)
		return
	}
	hlog.Warnf(format, v...)
}

// Error logs at error level.
func Error(format string, v ...any) {
	if jsonOut {
		emit(LevelError, "error", format, v...)
		return
	}
	hlog.Errorf(format, v...)
}
