package tests

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/thatsneat-dev/nprt/internal/logging"
)

func TestNew_NonVerbose(t *testing.T) {
	log := logging.New(false)
	defer func() { _ = log.Sync() }()

	// Non-verbose logger should be a no-op (check that it doesn't panic on use)
	log.Debug("this should be discarded")
	log.Info("this too")

	// Verify it's effectively disabled by checking core
	if log.Core().Enabled(zapcore.DebugLevel) {
		t.Error("non-verbose logger should not enable debug level")
	}
}

func TestNew_Verbose(t *testing.T) {
	log := logging.New(true)
	defer func() { _ = log.Sync() }()

	if !log.Core().Enabled(zapcore.DebugLevel) {
		t.Error("verbose logger should enable debug level")
	}

	// Verify it can log without panicking
	log.Debug("test debug", zap.String("key", "value"))
	log.Info("test info")
}
