package http

import (
	"testing"

	"github.com/fino-io/core/go/fino/core"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestEnvelopedResponse_ToErrorWrapped(t *testing.T) {
	code := &core.ErrorCode{
		Code:           600121001,
		Name:           "TASK_NOT_FOUND",
		Domain:         "tasks",
		Description:    "Task does not exist",
		HttpStatusCode: 404,
	}
	resp := &EnvelopedResponse{
		Error: core.NewError(code, "task not found"),
		Data:  "payload",
	}
	require.NoError(t, resp.Error.AddDetail("task-1"))

	data, err := jsoniter.Marshal(resp.ToErrorWrapped())
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"error": {
			"code": {"code":600121001,"name":"TASK_NOT_FOUND","domain":"tasks","description":"Task does not exist","httpStatusCode":404},
			"message":"task not found",
			"details":["task-1"]
		},
		"data":"payload"
	}`, string(data))

	var wrapped ErrorWrappedEnvelopedResponse
	require.NoError(t, jsoniter.Unmarshal(data, &wrapped))
	assert.True(t, proto.Equal(resp.Error, wrapped.Error))
	assert.Equal(t, resp.Data, wrapped.Data)
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
