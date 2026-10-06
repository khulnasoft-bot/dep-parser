package sonatype

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/khulnasoft/dep-parser/pkg/log"
)

func TestLogger(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)

	original := log.Logger
	t.Cleanup(func() {
		log.SetLogger(original)
	})
	log.SetLogger(zap.New(core).Sugar())

	l := logger{}

	l.Info("info message", "key", "value")
	l.Warn("warn message")
	l.Debug("debug message")
	l.Error("error message")
	// "request failed" is downgraded to debug level
	l.Error("request failed", "key", "value")
	// "performing request" is suppressed entirely
	l.Debug("performing request")

	var messages []string
	for _, e := range logs.All() {
		messages = append(messages, e.Message)
	}
	require.NotEmpty(t, messages)

	assert.Contains(t, messages, "info message")
	assert.Contains(t, messages, "warn message")
	assert.Contains(t, messages, "debug message")
	assert.Contains(t, messages, "error message")
	assert.Contains(t, messages, "request failed")
	assert.NotContains(t, messages, "performing request")

	// the downgraded message is logged at debug level, not error level
	for _, e := range logs.All() {
		if e.Message == "request failed" {
			assert.Equal(t, zapcore.DebugLevel, e.Level)
		}
	}
}
