package http

import (
	"strconv"
	"unsafe"

	"github.com/fino-io/core/go/fino/core"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	core.RegisterJSONTypeDecoder("http.EnvelopedResponse", &envelopedResponseCodec{})
	core.RegisterJSONTypeEncoder("http.EnvelopedResponse", &envelopedResponseCodec{})
}

type envelopedResponseCodec struct{}

type BareEnvelopedResponse EnvelopedResponse

func (codec *envelopedResponseCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	resp := (*EnvelopedResponse)(ptr)
	for filed := iter.ReadObject(); filed != ""; filed = iter.ReadObject() {
		switch filed {
		case "code":
			if resp.Error == nil {
				resp.Error = &core.Error{}
			}
			if resp.Error.Code == nil {
				resp.Error.Code = &core.ErrorCode{}
			}

			codeStr := iter.ReadString()
			if v, err := strconv.ParseInt(codeStr, 10, 32); err == nil {
				resp.Error.Code.Code = int32(v)
			}
		case "message":
			if resp.Error == nil {
				resp.Error = &core.Error{}
			}
			resp.Error.Message = iter.ReadString()
		case "data":
			iter.ReadVal(&resp.Data)
		case "totalCount":
			iter.ReadVal(&resp.TotalCount)
		case "nextPageToken":
			iter.ReadVal(&resp.NextPageToken)
		default:
			iter.Skip()
		}
	}
}

func (codec *envelopedResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	resp := (*EnvelopedResponse)(ptr)
	stream.WriteObjectStart()
	first := true
	if resp.Error != nil {
		if resp.Error.Code != nil {
			stream.WriteObjectField("code")
			stream.WriteString(resp.Error.Code.Format())
			first = false
		}

		if len(resp.Error.Message) > 0 {
			if !first {
				stream.WriteMore()
			}
			stream.WriteObjectField("message")
			stream.WriteString(resp.Error.Message)
		}

		if len(resp.Error.Details) > 0 {
			if !first {
				stream.WriteMore()
			}
			stream.WriteObjectField("details")
			stream.WriteVal(resp.Error.Details)
		}
	}

	if !first {
		stream.WriteMore()
	}
	stream.WriteObjectField("data")
	stream.WriteVal(resp.Data)

	if resp.TotalCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("totalCount")
		stream.WriteVal(resp.TotalCount)
	}

	if len(resp.NextPageToken) > 0 {
		stream.WriteMore()
		stream.WriteObjectField("nextPageToken")
		stream.WriteString(resp.NextPageToken)
	}

	stream.WriteObjectEnd()
}

func (codec *envelopedResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	e := (*EnvelopedResponse)(ptr)
	return e == nil
}
