package logging

import (
	"log"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Creates a Logger with a specific level
func NewLogger(level int) Logger {
	return &simpleLog{
		logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		level:  level,
	}
}

// Wraps a standard lib log.Logger with support for levels
type simpleLog struct {
	logger *log.Logger
	level  int
}

// Levels are just numbers, same values as the slog package
const (
	DEBUG = -4
	INFO  = 0
	WARN  = 4
	ERROR = 8
)

// Logs the provided message if Debug is enabled
func (l simpleLog) Debug(msg string, args ...any) {
	if l.level <= DEBUG {
		l.logger.Printf("[Debug] "+msg, args...)
	}
}

// Logs the provided message if Info is enabled
func (l simpleLog) Info(msg string, args ...any) {
	if l.level <= INFO {
		l.logger.Printf(" [Info] "+msg, args...)
	}
}

// Logs the provided message if Warn is enabled
func (l simpleLog) Warn(msg string, args ...any) {
	if l.level <= WARN {
		l.logger.Printf(" [Warn] "+msg, args...)
	}
}

// Logs the provided message if Error is enabled
func (l simpleLog) Error(msg string, args ...any) {
	if l.level <= ERROR {
		l.logger.Printf("[Error] "+msg, args...)
	}
}

type PrefixLogger struct {
	Log    Logger
	Prefix string
}

func (l *PrefixLogger) Debug(msg string, args ...any) {
	l.Log.Debug(l.Prefix+msg, args...)
}
func (l *PrefixLogger) Info(msg string, args ...any) {
	l.Log.Info(l.Prefix+msg, args...)
}
func (l *PrefixLogger) Warn(msg string, args ...any) {
	l.Log.Warn(l.Prefix+msg, args...)
}
func (l *PrefixLogger) Error(msg string, args ...any) {
	l.Log.Error(l.Prefix+msg, args...)
}
