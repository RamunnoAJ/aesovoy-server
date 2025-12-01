package testutils

import (
	"testing"
)

// Minimal test logger to satisfy app.Application requirement
type TestLogger struct {
	t *testing.T
}

func (tl *TestLogger) Debug(msg string, args ...any) {
	// tl.t.Logf("DEBUG: %s %v", msg, args)
}
func (tl *TestLogger) Info(msg string, args ...any) {
	// tl.t.Logf("INFO: %s %v", msg, args)
}
func (tl *TestLogger) Warn(msg string, args ...any) {
	tl.t.Logf("WARN: %s %v", msg, args)
}
func (tl *TestLogger) Error(msg string, args ...any) {
	tl.t.Errorf("ERROR: %s %v", msg, args)
}
func (tl *TestLogger) With(args ...any) *TestLogger {
	return tl
}

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}
