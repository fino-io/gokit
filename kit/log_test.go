package kit

import (
	"context"
	"errors"
	"testing"

	"github.com/fino-io/finokit/logs"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
)

type captureLogger struct {
	entry logs.Entry
}

func (l *captureLogger) SetLevel(logs.Level) {}

func (l *captureLogger) GetLevel() logs.Level { return logs.InfoLevel }

func (l *captureLogger) With(...logs.Field) logs.Logger { return l }

func (l *captureLogger) Log(_ context.Context, entry logs.Entry) {
	l.entry = entry
}

func Test_kitLogger_Log(t *testing.T) {
	testErr := errors.New("boom")

	tests := []struct {
		name   string
		kvs    []any
		assert func(t *testing.T, entry logs.Entry)
	}{
		{
			name: "default info fields",
			kvs:  []any{"foo", "bar"},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.InfoLevel, entry.Level)
				assert.Empty(t, entry.Message)
				assert.Equal(t, []logs.Field{{Key: "foo", Value: "bar"}}, entry.Fields)
			},
		},
		{
			name: "message first",
			kvs:  []any{"msg", "hello", "foo", "bar"},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.InfoLevel, entry.Level)
				assert.Equal(t, "hello", entry.Message)
				assert.Equal(t, []logs.Field{{Key: "foo", Value: "bar"}}, entry.Fields)
			},
		},
		{
			name: "level then message",
			kvs:  []any{"level", "debug", "msg", "hello", "foo", "bar"},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.DebugLevel, entry.Level)
				assert.Equal(t, "hello", entry.Message)
				assert.Equal(t, []logs.Field{{Key: "foo", Value: "bar"}}, entry.Fields)
			},
		},
		{
			name: "err promotes level",
			kvs:  []any{"err", testErr, "took", 1},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.ErrorLevel, entry.Level)
				assert.Equal(t, []logs.Field{{Key: "err", Value: testErr}, {Key: "took", Value: 1}}, entry.Fields)
			},
		},
		{
			name: "error key promotes level",
			kvs:  []any{"error", testErr, "took", 1},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.ErrorLevel, entry.Level)
				assert.Equal(t, []logs.Field{{Key: "error", Value: testErr}, {Key: "took", Value: 1}}, entry.Fields)
			},
		},
		{
			name: "odd keyvals append missing value",
			kvs:  []any{"foo"},
			assert: func(t *testing.T, entry logs.Entry) {
				assert.Equal(t, logs.InfoLevel, entry.Level)
				assert.Equal(t, []logs.Field{{Key: "foo", Value: log.ErrMissingValue}}, entry.Fields)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &captureLogger{}
			l := kitLogger{Logger: logger}

			err := l.Log(tt.kvs...)
			assert.NoError(t, err)
			tt.assert(t, logger.entry)
		})
	}
}
