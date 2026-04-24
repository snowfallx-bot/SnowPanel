package requestctx

import (
	"context"
	"strings"
)

type requestIDContextKey struct{}

var requestIDKey requestIDContextKey

func WithRequestID(ctx context.Context, requestID string) context.Context {
	normalized := strings.TrimSpace(requestID)
	if normalized == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, normalized)
}

func RequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value := ctx.Value(requestIDKey)
	id, ok := value.(string)
	if !ok {
		return "", false
	}
	id = strings.TrimSpace(id)
	return id, id != ""
}
