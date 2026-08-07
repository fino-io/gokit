package http

import "github.com/fino-io/core/go/fino/core"

type EnvelopedResponse struct {
	Error *core.Error `json:"error"`
	Data  interface{} `json:"data"`

	TotalCount    int32  `json:"totalCount,omitempty"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

func (r *EnvelopedResponse) ToErrorWrapped() *ErrorWrappedEnvelopedResponse {
	if r != nil {
		return (*ErrorWrappedEnvelopedResponse)(r)
	}
	return nil
}

func (r *EnvelopedResponse) CheckError(successCodes ...int32) error {
	if r == nil || r.Error == nil || r.Error.Code == nil {
		return nil
	}

	if len(successCodes) == 0 {
		successCodes = []int32{200}
	}

	code := r.Error.Code.Code
	for _, successCode := range successCodes {
		if code == successCode {
			return nil
		}
	}
	return r.Error
}
