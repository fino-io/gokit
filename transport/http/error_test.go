package http

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/fino-io/core/go/fino/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCodedError struct {
	code    int32
	message string
	cause   error
}

func (e testCodedError) Error() string {
	return e.message
}

func (e testCodedError) Code() int32 {
	return e.code
}

func (e testCodedError) Message() string {
	return e.message
}

func (e testCodedError) Extra() map[string]string {
	return nil
}

func (e testCodedError) Unwrap() error {
	return e.cause
}

func (e testCodedError) StatusCode() int {
	return http.StatusNotFound
}

func TestCoreErrorFromErrorPreservesErrorXCodeAndMessage(t *testing.T) {
	got := CoreErrorFromError(testCodedError{
		code:    600121001,
		message: "task not found",
	})

	require.NotNil(t, got)
	require.NotNil(t, got.Code)
	assert.Equal(t, int32(600121001), got.Code.Code)
	assert.Equal(t, "task not found", got.Message)
}

func TestCoreErrorFromErrorPrefersBusinessCode(t *testing.T) {
	got := CoreErrorFromError(testCodedError{
		code:    600121001,
		message: "task not found",
		cause:   core.NewErrorFrom(http.StatusNotFound, "not found"),
	})

	assert.Equal(t, int32(600121001), got.Code.Code)
	assert.Equal(t, "task not found", got.Message)
}

func TestStatusCodeFromErrorFindsWrappedStatusCoder(t *testing.T) {
	err := fmt.Errorf("endpoint failed: %w", testCodedError{})
	assert.Equal(t, http.StatusNotFound, StatusCodeFromError(err))
	assert.Equal(t, http.StatusInternalServerError, StatusCodeFromError(errors.New("unknown")))
}

func TestWrapErrorAddsAllHeaderPairs(t *testing.T) {
	err := WrapError(
		errors.New("bad request"),
		400,
		"decode failed",
		"X-First", "one",
		"X-Second", "two",
	)

	assert.Equal(t, "one", err.Headers().Get("X-First"))
	assert.Equal(t, "two", err.Headers().Get("X-Second"))
}
