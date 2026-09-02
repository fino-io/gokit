package http

import (
	"context"
	"strings"
)

func IsEnvelopeStyle(ctx context.Context, style string) bool {
	enveloped := false
	style = strings.ToLower(style)
	if style == EnvelopeStyle || style == underScoreEnvelopeStyle {
		enveloped = true
	}
	if val, ok := ctx.Value(EnvelopeStyle).(bool); ok {
		enveloped = val
	}
	if val, ok := ctx.Value(underScoreEnvelopeStyle).(bool); ok {
		enveloped = val
	}
	return enveloped
}

func IsAIPStyle(ctx context.Context, style string) bool {
	aip := false
	style = strings.ToLower(style)
	if style == AIPStyle || style == underScoreAIPStyle {
		aip = true
	}
	if val, ok := ctx.Value(AIPStyle).(bool); ok {
		aip = val
	}
	if val, ok := ctx.Value(underScoreAIPStyle).(bool); ok {
		aip = val
	}
	return aip
}
