package http

import (
	"context"
	"encoding/json"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ResponseJsonWriter struct {
	Response interface{}
}

func NewResponseJsonWriter(response interface{}) *ResponseJsonWriter {
	return &ResponseJsonWriter{Response: response}
}

func (r *ResponseJsonWriter) WriteHttpResponse(ctx context.Context, writer http.ResponseWriter) error {
	_ = ctx
	response, err := normalizeProtoJSONResponse(r.Response)
	if err != nil {
		return err
	}

	stream := jsoniter.NewStream(jsoniter.ConfigFastest, writer, 512)
	stream.WriteVal(response)
	if err := stream.Flush(); err != nil {
		return err
	}

	return stream.Error
}

func normalizeProtoJSONResponse(response interface{}) (interface{}, error) {
	switch response := response.(type) {
	case proto.Message:
		return marshalProtoJSON(response)
	case *EnvelopedResponse:
		if response == nil {
			return response, nil
		}
		if message, ok := response.Data.(proto.Message); ok {
			data, err := marshalProtoJSON(message)
			if err != nil {
				return nil, err
			}

			_copy := *response
			_copy.Data = data
			return &_copy, nil
		}
	}

	return response, nil
}

func marshalProtoJSON(message proto.Message) (json.RawMessage, error) {
	data, err := (protojson.MarshalOptions{
		UseProtoNames:  false,
		UseEnumNumbers: false,
	}).Marshal(message)
	if err != nil {
		return nil, err
	}
	return data, nil
}
