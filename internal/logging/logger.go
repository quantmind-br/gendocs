package logging

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is a type alias for zap.Field
type Field = zap.Field

// Common field constructors
var (
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Any      = zap.Any
	Error    = zap.Error
	Err      = zap.NamedError
	Duration = zap.Duration
	Time     = zap.Time
)

// LevelFromString converts a string level to zapcore.Level
func LevelFromString(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Logger wraps zap.Logger with application-specific methods
type Logger struct {
	zap *zap.Logger
}

// Config holds logger configuration
type Config struct {
	LogDir         string
	FileLevel      zapcore.Level
	ConsoleLevel   zapcore.Level
	EnableCaller   bool
	ConsoleEnabled bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		LogDir:         ".ai/logs",
		FileLevel:      zapcore.InfoLevel,
		ConsoleLevel:   zapcore.DebugLevel,
		EnableCaller:   true,
		ConsoleEnabled: true,
	}
}

// NewLogger creates a new logger with file and optional console output
func NewLogger(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, err
	}

	// File encoder (JSON)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.TimeKey = "timestamp"
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// File writer
	logFile := filepath.Join(cfg.LogDir, "gendocs.log")
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	fileWriter := zapcore.AddSync(file)

	var core zapcore.Core

	if cfg.ConsoleEnabled {
		// Console encoder (human-readable with colors)
		consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
		consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

		// Console writer
		consoleWriter := zapcore.AddSync(os.Stderr)

		// Core with both outputs
		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, fileWriter, cfg.FileLevel),
			zapcore.NewCore(consoleEncoder, consoleWriter, cfg.ConsoleLevel),
		)
	} else {
		// File-only logging when console is disabled
		core = zapcore.NewCore(fileEncoder, fileWriter, cfg.FileLevel)
	}

	// Create logger
	opts := []zap.Option{zap.AddStacktrace(zapcore.ErrorLevel)}
	if cfg.EnableCaller {
		opts = append(opts, zap.AddCaller())
	}
	zapLogger := zap.New(core, opts...)

	return &Logger{zap: zapLogger}, nil
}

// NewNopLogger creates a no-op logger for testing
func NewNopLogger() *Logger {
	return &Logger{zap: zap.NewNop()}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

// Named creates a named child logger
func (l *Logger) Named(name string) *Logger {
	return &Logger{zap: l.zap.Named(name)}
}
