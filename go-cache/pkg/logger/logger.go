package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var (
	// 默认日志实例
	defaultLogger = logrus.New()
)

// InitLogger 初始化日志配置
func InitLogger(level string) {
	// 设置输出格式为JSON
	defaultLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置输出为标准输出
	defaultLogger.SetOutput(os.Stdout)

	// 根据传入的级别设置日志级别
	switch level {
	case "debug":
		defaultLogger.SetLevel(logrus.DebugLevel)
	case "info":
		defaultLogger.SetLevel(logrus.InfoLevel)
	case "warn":
		defaultLogger.SetLevel(logrus.WarnLevel)
	case "error":
		defaultLogger.SetLevel(logrus.ErrorLevel)
	default:
		defaultLogger.SetLevel(logrus.InfoLevel)
	}
}

// 获取日志实例
func GetLogger() *logrus.Logger {
	return defaultLogger
}

// Debug logs a message at level Debug
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Debugf logs a message at level Debug with format
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info logs a message at level Info
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Infof logs a message at level Info with format
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn logs a message at level Warn
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Warnf logs a message at level Warn with format
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error logs a message at level Error
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Errorf logs a message at level Error with format
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatal logs a message at level Fatal
func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

// Fatalf logs a message at level Fatal with format
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}
