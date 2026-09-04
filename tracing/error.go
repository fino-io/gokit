package tracing

import (
	"strings"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func recordError(span trace.Span, err error) {
	if err == nil {
		return
	}

	// Keep the original error for business logic; only the telemetry copy is sanitized.
	rawMessage := err.Error()
	message := telemetryString(rawMessage)
	recordedErr := err
	if message != rawMessage {
		recordedErr = sanitizedError{cause: err, message: message}
	}

	span.RecordError(recordedErr)
	span.SetStatus(codes.Error, message)
}

func telemetryString(value string) string {
	return strings.ToValidUTF8(value, "\uFFFD")
}

type sanitizedError struct {
	cause   error
	message string
}

func (e sanitizedError) Error() string {
	return e.message
}

func (e sanitizedError) Unwrap() error {
	return e.cause
}
