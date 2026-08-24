package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fino-io/core/go/fino/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestResponseJsonWriter_WriteHttpResponse(t *testing.T) {
	const out = "string response"
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = NewResponseJsonWriter(out).WriteHttpResponse(context.Background(), w)
	}

	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	r := httptest.NewRecorder()
	handler(r, req)

	resp := r.Result()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, `"string response"`, string(body))
}

func TestResponseJsonWriter_WriteHttpResponse_UsesProtoJSONForMessages(t *testing.T) {
	body := responseBody(t, enumMessage())

	assert.JSONEq(t, `{"name":"status","type":"TYPE_ENUM","label":"LABEL_OPTIONAL"}`, body)
}

func TestResponseJsonWriter_WriteHttpResponse_UsesProtoJSONForEnvelopedMessages(t *testing.T) {
	body := responseBody(t, &EnvelopedResponse{
		Error: core.NewErrorFrom(200, "OK"),
		Data:  enumMessage(),
	})

	assert.JSONEq(t, `{"code":"200","message":"OK","data":{"name":"status","type":"TYPE_ENUM","label":"LABEL_OPTIONAL"}}`, body)
}

func responseBody(t *testing.T, response interface{}) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	err := NewResponseJsonWriter(response).WriteHttpResponse(context.Background(), recorder)
	require.NoError(t, err)

	return recorder.Body.String()
}

func enumMessage() *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:  proto.String("status"),
		Type:  descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
		Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}
}
