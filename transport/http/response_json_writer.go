package http

import (
	"context"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type ResponseJsonWriter struct {
	Response interface{}
}

func NewResponseJsonWriter(response interface{}) *ResponseJsonWriter {
	return &ResponseJsonWriter{Response: response}
}

func (r *ResponseJsonWriter) WriteHttpResponse(ctx context.Context, writer http.ResponseWriter) error {
	_ = ctx
	stream := jsoniter.NewStream(jsoniter.ConfigFastest, writer, 512)
	stream.WriteVal(r.Response)
	if err := stream.Flush(); err != nil {
		return err
	}

	return stream.Error
}
