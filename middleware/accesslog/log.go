package accesslog

import (
	"context"

	"github.com/fino-io/finokit/logs"
)

type level uint8

const (
	levelInfo level = iota
	levelWarn
	levelError
)

type logFunc func(context.Context, level, string, ...any)

func defaultLog(_ context.Context, level level, message string, fields ...any) {
	switch level {
	case levelWarn:
		logs.Warnw(message, fields...)
	case levelError:
		logs.Errorw(message, fields...)
	default:
		logs.Infow(message, fields...)
	}
}
