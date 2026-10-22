// Package pagination provides compatibility with the former pagination package.
// Deprecated: use github.com/fino-io/finokit/pagination. Load configuration in the application.
package pagination

import "github.com/fino-io/finokit/pagination"

type CursorCodec = pagination.CursorCodec
type Page[T any] = pagination.Page[T]
type Paginator = pagination.Paginator

const (
	DefaultPageSize = pagination.DefaultPageSize
	MaxPageSize     = pagination.MaxPageSize
)

var (
	ErrInvalidPageSize  = pagination.ErrInvalidPageSize
	ErrInvalidPageToken = pagination.ErrInvalidPageToken
)

func NewCursorCodec(key []byte) (*CursorCodec, error) {
	return pagination.NewCursorCodec(key)
}

func Resolve(codec *CursorCodec, token string, pageSize int, namespace string, scope any) (int, int, string, error) {
	return pagination.Resolve(codec, token, pageSize, namespace, scope)
}

func ResolveKeyset(codec *CursorCodec, token string, pageSize int, namespace string, scope any) (string, int, string, error) {
	return pagination.ResolveKeyset(codec, token, pageSize, namespace, scope)
}
