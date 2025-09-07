package utils

import "log"

type Logger struct{}

func NewLogger() *Logger { return &Logger{} }

func (l *Logger) Infof(fmtStr string, args ...interface{})  { log.Printf("[INFO] "+fmtStr, args...) }
func (l *Logger) Info(fmtStr string, args ...interface{})   { log.Printf("[INFO] "+fmtStr, args...) }
func (l *Logger) Warnf(fmtStr string, args ...interface{})  { log.Printf("[WARN] "+fmtStr, args...) }
func (l *Logger) Errorf(fmtStr string, args ...interface{}) { log.Printf("[ERROR] "+fmtStr, args...) }
func (l *Logger) Fatalf(fmtStr string, args ...interface{}) { log.Fatalf("[FATAL] "+fmtStr, args...) }
