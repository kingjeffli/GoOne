package ssrpc

import "github.com/golang/protobuf/proto"

// Handler is the unified handler signature used by the generated wrappers.
// req is a concrete *pb.Request message; rsp must be a concrete *pb.Response message.
type Handler func(ctx *Context, req proto.Message) (proto.Message, error)

type Middleware func(next Handler) Handler

func Chain(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}


