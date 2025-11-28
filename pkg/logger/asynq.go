package logger

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// AsynqLoggerAdapter adapts the common logger to asynq.Logger interface
type AsynqLoggerAdapter struct {
	logger Logger
}

// NewAsynqLoggerAdapter creates a new asynq logger adapter
func NewAsynqLoggerAdapter(log Logger) asynq.Logger {
	return &AsynqLoggerAdapter{logger: log}
}

// Debug implements asynq.Logger
func (a *AsynqLoggerAdapter) Debug(args ...interface{}) {
	a.logger.Debug("asynq debug", String("message", fmt.Sprint(args...)))
}

// Info implements asynq.Logger
func (a *AsynqLoggerAdapter) Info(args ...interface{}) {
	a.logger.Info("asynq info", String("message", fmt.Sprint(args...)))
}

// Warn implements asynq.Logger
func (a *AsynqLoggerAdapter) Warn(args ...interface{}) {
	a.logger.Warn("asynq warn", String("message", fmt.Sprint(args...)))
}

// Error implements asynq.Logger
func (a *AsynqLoggerAdapter) Error(args ...interface{}) {
	a.logger.Error("asynq error", String("message", fmt.Sprint(args...)))
}

// Fatal implements asynq.Logger
func (a *AsynqLoggerAdapter) Fatal(args ...interface{}) {
	a.logger.Error("asynq fatal", String("message", fmt.Sprint(args...)))
}
