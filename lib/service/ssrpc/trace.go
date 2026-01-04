package ssrpc

import (
	"github.com/golang/protobuf/proto"
)

// Trace is a placeholder middleware for Phase A+.
//
// We don't yet have a unified trace context in the project; when you introduce one,
// this middleware is the single place to:
// - extract trace/span ids from the underlying transport (SSPacket header / router metadata)
// - attach to ctx (or logger fields)
// - propagate to outgoing Call/Send
func Trace() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			return next(ctx, req)
		}
	}
}


