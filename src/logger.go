package main

import "log"

type Logger struct {
	level string
}

const (
	LevelDebug   = "debug"
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
)

func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		LevelDebug:   0,
		LevelInfo:    1,
		LevelWarning: 2,
		LevelError:   3,
	}
	return levels[level] >= levels[l.level]
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.shouldLog(LevelDebug) {
		log.Printf(format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.shouldLog(LevelDebug) {
		log.Printf(format, v...)
	}
}

func (l *Logger) Warning(format string, v ...interface{}) {
	if l.shouldLog(LevelDebug) {
		log.Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.shouldLog(LevelDebug) {
		log.Printf(format, v...)
	}
}
