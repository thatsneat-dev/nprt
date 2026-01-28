// Package logging provides a configured zap logger for the CLI.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a zap logger based on the verbose flag.
// If verbose is false, returns a no-op logger that discards all output.
// If verbose is true, returns a debug-level console logger that writes
// to stderr with timestamps, log levels, and caller information.
func New(verbose bool) *zap.Logger {
	if !verbose {
		return zap.NewNop()
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:      "ts",
		LevelKey:     "level",
		MessageKey:   "msg",
		CallerKey:    "caller",
		FunctionKey:  "func",
		EncodeTime:   zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
		LineEnding:   zapcore.DefaultLineEnding,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encCfg),
		zapcore.Lock(os.Stderr),
		zapcore.DebugLevel,
	)

	return zap.New(core, zap.AddCaller())
}
