package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
	With(key string, value any) Logger
}

type ZeroLogger struct {
	logger zerolog.Logger
}

func New(level string) *ZeroLogger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		l = zerolog.InfoLevel
	}

	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	z := zerolog.New(output).Level(l).With().Timestamp().Logger()

	return &ZeroLogger{logger: z}
}

func (l *ZeroLogger) Debug(msg string, args ...any) {
	if len(args) > 0 {
		l.logger.Debug().Fields(toFields(args...)).Msg(msg)
	} else {
		l.logger.Debug().Msg(msg)
	}
}

func (l *ZeroLogger) Info(msg string, args ...any) {
	if len(args) > 0 {
		l.logger.Info().Fields(toFields(args...)).Msg(msg)
	} else {
		l.logger.Info().Msg(msg)
	}
}

func (l *ZeroLogger) Warn(msg string, args ...any) {
	if len(args) > 0 {
		l.logger.Warn().Fields(toFields(args...)).Msg(msg)
	} else {
		l.logger.Warn().Msg(msg)
	}
}

func (l *ZeroLogger) Error(msg string, args ...any) {
	if len(args) > 0 {
		l.logger.Error().Fields(toFields(args...)).Msg(msg)
	} else {
		l.logger.Error().Msg(msg)
	}
}

func (l *ZeroLogger) Fatal(msg string, args ...any) {
	if len(args) > 0 {
		l.logger.Fatal().Fields(toFields(args...)).Msg(msg)
	} else {
		l.logger.Fatal().Msg(msg)
	}
}

func (l *ZeroLogger) With(key string, value any) Logger {
	return &ZeroLogger{logger: l.logger.With().Interface(key, value).Logger()}
}

func toFields(args ...any) map[string]any {
	fields := make(map[string]any)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				fields[key] = args[i+1]
			}
		}
	}
	return fields
}
