package webrtcutil

import (
	"github.com/pion/logging"

	mlogging "github.com/mutagen-io/mutagen/pkg/logging"
)

// leveledLogger is a LeveledLogger implementation that uses a Mutagen logger.
type leveledLogger struct {
	// logger is the underlying Mutagen logger.
	logger *mlogging.Logger
}

// Trace implements LeveledLogger.Trace.
func (l *leveledLogger) Trace(msg string) {
	l.logger.Trace(msg)
}

// Tracef implements LeveledLogger.Tracef.
func (l *leveledLogger) Tracef(format string, args ...interface{}) {
	l.logger.Tracef(format, args...)
}

// Debug implements LeveledLogger.Debug.
func (l *leveledLogger) Debug(msg string) {
	l.logger.Debug(msg)
}

// Debugf implements LeveledLogger.Debugf.
func (l *leveledLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Info implements LeveledLogger.Info.
func (l *leveledLogger) Info(msg string) {
	l.logger.Info(msg)
}

// Infof implements LeveledLogger.Infof.
func (l *leveledLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warn implements LeveledLogger.Warn.
func (l *leveledLogger) Warn(msg string) {
	l.logger.Warning(msg)
}

// Warnf implements LeveledLogger.Warnf.
func (l *leveledLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warningf(format, args...)
}

// Error implements LeveledLogger.Error.
func (l *leveledLogger) Error(msg string) {
	l.logger.Error(msg)
}

// Errorf implements LeveledLogger.Errorf.
func (l *leveledLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// loggerFactory is a LoggerFactory that creates LeveledLogger instances as
// subloggers of its internal logger.
type loggerFactory struct {
	// logger is the underlying Mutagen logger.
	logger *mlogging.Logger
}

// NewLoggerFactory creates a LoggerFactory that uses Mutagen's logging
// infrastructure. The factory contains a sublogger of the root logger with the
// specified name. Derived LeveledLogger instances are created as subloggers of
// the factory's logger.
func NewLoggerFactory(name string) logging.LoggerFactory {
	return &loggerFactory{
		logger: mlogging.RootLogger.Sublogger(name),
	}
}

// NewLogger creates a new leveled logger.
func (f *loggerFactory) NewLogger(scope string) logging.LeveledLogger {
	// Log creation of the logger.
	f.logger.Trace("Creating logger:", scope)

	// Create the logger.
	return &leveledLogger{
		logger: f.logger.Sublogger(scope),
	}
}
