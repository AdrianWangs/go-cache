// Package logger provides structured logging functionality for the cache system
package logger

import (
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	// defaultLogger is the global logger instance
	defaultLogger = logrus.New()
)

// Fields represents a set of log fields
type Fields map[string]interface{}

func init() {
	// Set default configuration
	defaultLogger.SetOutput(os.Stdout)
	defaultLogger.SetLevel(logrus.InfoLevel)
	defaultLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// SetOutput sets the output destination for the default logger
func SetOutput(output io.Writer) {
	defaultLogger.SetOutput(output)
}

// SetLevel sets the logging level for the default logger
func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		defaultLogger.SetLevel(logrus.DebugLevel)
	case "info":
		defaultLogger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		defaultLogger.SetLevel(logrus.WarnLevel)
	case "error":
		defaultLogger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		defaultLogger.SetLevel(logrus.FatalLevel)
	default:
		defaultLogger.SetLevel(logrus.InfoLevel)
	}
}

// UseJSONFormat configures the logger to use JSON formatting
func UseJSONFormat() {
	defaultLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// WithFields returns a log entry with pre-populated fields
func WithFields(fields Fields) *logrus.Entry {
	return defaultLogger.WithFields(logrus.Fields(fields))
}

// Debug logs a message at the debug level
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Debugf logs a formatted message at the debug level
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info logs a message at the info level
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Infof logs a formatted message at the info level
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn logs a message at the warn level
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Warnf logs a formatted message at the warn level
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error logs a message at the error level
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Errorf logs a formatted message at the error level
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatal logs a message at the fatal level and then exits
func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

// Fatalf logs a formatted message at the fatal level and then exits
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}
