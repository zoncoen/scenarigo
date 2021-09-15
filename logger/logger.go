package logger

import (
	"fmt"
	"log"
	"strings"

	"github.com/fatih/color"
)

var (
	bold = color.New(color.Bold)
	red  = color.New(color.FgHiRed)
)

// Logger represents a logger.
type Logger interface {
	Info(msg string, kvs ...interface{})
	Error(err error, msg string, kvs ...interface{})
}

type logger struct {
	logger *log.Logger
	level  LogLevel
}

// NewLogger returns a new logger.
func NewLogger(l *log.Logger, level LogLevel) Logger {
	return &logger{
		logger: l,
		level:  level,
	}
}

// NewNopLogger returns a no operation logger.
func NewNopLogger() Logger {
	return &logger{
		level: LogLevelNone,
	}
}

// A LogLevel is a logging level.
type LogLevel int

const (
	LogLevelAll LogLevel = iota
	LogLevelInfo
	LogLevelError
	LogLevelNone
)

// Info logs a non-error message with the given key/value pairs.
func (l *logger) Info(msg string, kvs ...interface{}) {
	if l.level > LogLevelInfo {
		return
	}
	l.logger.Printf(bold.Sprintf("[INFO] %q%s", msg, flatten(kvs...)))
}

// Error logs an error, with the given message and key/value pairs.
func (l *logger) Error(err error, msg string, kvs ...interface{}) {
	if l.level > LogLevelError {
		return
	}
	l.logger.Printf(red.Sprintf(`[ERROR] %q "error"=%q%s`, msg, err, flatten(kvs...)))
}

func flatten(kvs ...interface{}) string {
	if len(kvs)%2 != 0 {
		kvs = append(kvs, "<no-value>")
	}
	var b strings.Builder
	for i := 0; i < len(kvs); i += 2 {
		b.WriteString(fmt.Sprintf(` "%v"="%v"`, kvs[i], kvs[i+1]))
	}
	return b.String()
}
