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

func defaultLog(ctx context.Context, level level, message string, fields ...any) {
	switch level {
	case levelWarn:
		logs.Ctx(ctx).Warnw(message, fields...)
	case levelError:
		logs.Ctx(ctx).Errorw(message, fields...)
	default:
		logs.Ctx(ctx).Infow(message, fields...)
	}
}
