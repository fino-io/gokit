package http

import (
	"testing"

	"github.com/fino-io/core/go/fino/core"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopedResponse_ToErrorWrapped(t *testing.T) {
	resp := &EnvelopedResponse{
		Error:         core.NewErrorFrom(400, "Invalid Arguments"),
		Data:          nil,
		TotalCount:    0,
		NextPageToken: "",
	}

	wer := resp.ToErrorWrapped()
	json, err := jsoniter.Marshal(wer)
	assert.NoError(t, err)

	wrapped := &ErrorWrappedEnvelopedResponse{}
	err = jsoniter.Unmarshal(json, wrapped)
	assert.NoError(t, err)
	assert.NotNil(t, wrapped.Error)
}

func TestEnvelopedResponse_CheckErrorReturnsBusinessError(t *testing.T) {
	resp := &EnvelopedResponse{
		Error: core.NewErrorFrom(600121001, "task not found"),
		Data:  nil,
	}

	err := resp.CheckError(200)

	assert.Error(t, err)
	coreErr, ok := err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, int32(600121001), coreErr.Code.Code)
	assert.Equal(t, "task not found", err.Error())
}

func TestEnvelopedResponse_CheckErrorAllowsSuccessCode(t *testing.T) {
	resp := &EnvelopedResponse{
		Error: core.NewErrorFrom(200, "OK"),
		Data:  "payload",
	}

	assert.NoError(t, resp.CheckError(200))
}
