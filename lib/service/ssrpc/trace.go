package ssrpc

import (
	"strconv"
	"github.com/golang/protobuf/proto"
)

// TraceProvider is a pluggable tracing hook.
//
// Start should return a finish callback (may be nil). finish is always called with the handler error.
type TraceProvider interface {
	Start(ctx *Context, tags map[string]string) (finish func(err error))
}

// TraceWith runs tracing via the provided TraceProvider (no-op if nil).
func TraceWith(tp TraceProvider) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if tp == nil || ctx == nil {
				return next(ctx, req)
			}
			tags := map[string]string{
				"cmd":    strconv.FormatUint(uint64(ctx.Cmd), 10),
				"method": ctx.Method,
			}
			if ctx.TraceTags != nil {
				for k, v := range ctx.TraceTags {
					tags[k] = v
				}
			}
			finish := tp.Start(ctx, tags)
			rsp, err := next(ctx, req)
			if finish != nil {
				finish(err)
			}
			return rsp, err
		}
	}
}

// Trace keeps backward-compatibility and stays a no-op by default.
func Trace() Middleware {
	return TraceWith(nil)
}
