package hostctx

import "context"

type hostIDContextKey struct{}

var hostIDKey hostIDContextKey

func WithHostID(ctx context.Context, hostID int64) context.Context {
	if ctx == nil || hostID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, hostIDKey, hostID)
}

func HostID(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	value := ctx.Value(hostIDKey)
	hostID, ok := value.(int64)
	if !ok || hostID <= 0 {
		return 0, false
	}
	return hostID, true
}
