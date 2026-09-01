package kit

import (
	"context"
	"fmt"

	"github.com/fino-io/finokit/logs"
	"github.com/go-kit/kit/log"
)

type kitLogger struct {
	logs.Logger
}

func Logger() log.Logger {
	return kitLogger{Logger: logs.DefaultLogger()}
}

func (l kitLogger) Log(kvs ...interface{}) error {
	if len(kvs) == 0 {
		return nil
	}

	kvs = normalizeKeyvals(kvs)
	entry := logs.Entry{
		Level:      logs.InfoLevel,
		CallerSkip: 2,
	}

	if len(kvs) >= 2 {
		switch fmt.Sprint(kvs[0]) {
		case "level":
			entry.Level = parseLevel(kvs[1])
			kvs = kvs[2:]
		case "err", "error":
			entry.Level = logs.ErrorLevel
		}
	}

	if len(kvs) >= 2 && fmt.Sprint(kvs[0]) == "msg" {
		entry.Message = fmt.Sprint(kvs[1])
		kvs = kvs[2:]
	}

	entry.Fields = keyValuesToFields(kvs)
	l.Logger.Log(context.Background(), entry)

	return nil
}

func normalizeKeyvals(kvs []interface{}) []interface{} {
	normalized := append([]interface{}(nil), kvs...)
	if len(normalized)%2 != 0 {
		normalized = append(normalized, log.ErrMissingValue)
	}
	return normalized
}

func parseLevel(level interface{}) logs.Level {
	switch fmt.Sprint(level) {
	case "debug":
		return logs.DebugLevel
	case "warn":
		return logs.WarnLevel
	case "error":
		return logs.ErrorLevel
	default:
		return logs.InfoLevel
	}
}

func keyValuesToFields(kvs []interface{}) []logs.Field {
	if len(kvs) < 2 {
		return nil
	}

	fields := make([]logs.Field, 0, len(kvs)/2)
	for i := 0; i+1 < len(kvs); i += 2 {
		fields = append(fields, logs.Field{
			Key:   fmt.Sprint(kvs[i]),
			Value: kvs[i+1],
		})
	}
	return fields
}
