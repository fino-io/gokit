package http

import (
	"testing"

	"github.com/fino-io/core/go/fino/core"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var json = `{
  "code": "400",
  "message": "Invalid Arguments",
  "data": "This is data",
  "totalCount": 1,
  "nextPageToken": "{\"page\": 2}"
}`

func TestEnvelopedResponseJSON_Decode(t *testing.T) {
	resp := &EnvelopedResponse{}
	err := jsoniter.UnmarshalFromString(json, resp)
	assert.NoError(t, err)
	assert.Equal(t, int32(400), resp.Error.Code.Code)
	assert.Equal(t, "Invalid Arguments", resp.Error.Message)
	assert.Equal(t, "This is data", resp.Data.(string))
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Equal(t, "{\"page\": 2}", resp.NextPageToken)
}

func TestEnvelopedResponseJSON_Encode(t *testing.T) {
	resp := &EnvelopedResponse{
		Error:         core.NewErrorFrom(400, "Invalid Arguments"),
		Data:          "This is data",
		TotalCount:    1,
		NextPageToken: "{\"page\": 2}",
	}

	out, err := jsoniter.MarshalIndent(resp, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, json, string(out))
}
