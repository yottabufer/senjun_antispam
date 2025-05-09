package main

import "log"

type logger struct {
	level string
}

const (
	LevelDebug   = "debug"
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
)

func (l *logger) shouldLog(level string) bool {
	levels := map[string]int{
		LevelDebug:   0,
		LevelInfo:    1,
		LevelWarning: 2,
		LevelError:   3,
	}
	return levels[level] >= levels[l.level]
}

func (l *logger) debug(format string, v ...interface{}) {
	if l.shouldLog(LevelDebug) {
		log.Printf(format, v...)
	}
}

func (l *logger) info(format string, v ...interface{}) {
	if l.shouldLog(LevelInfo) {
		log.Printf(format, v...)
	}
}

func (l *logger) warning(format string, v ...interface{}) {
	if l.shouldLog(LevelWarning) {
		log.Printf(format, v...)
	}
}

func (l *logger) error(format string, v ...interface{}) {
	if l.shouldLog(LevelError) {
		log.Printf(format, v...)
	}
}
