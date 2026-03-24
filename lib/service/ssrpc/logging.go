package ssrpc

import (
	"time"

	"github.com/golang/protobuf/proto"
)

// Logging logs handler lifecycle with cmd/uid/rid/trans info (via ctx's logger methods).
//
// NOTE: this relies on cmd_handler.IContext's logging methods (Infof/Warningf/Errorf/Debugf).
func Logging() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			start := time.Now()
			var transID uint32
			if v, ok := any(ctx.IContext).(interface{ TransID() uint32 }); ok {
				transID = v.TransID()
			}
			if transID != 0 {
				ctx.Infof("ssrpc recv {cmd:%v, uid:%v, rid:%v, trans:%v}", uint32(ctx.Cmd), ctx.Uid(), ctx.Rid(), transID)
			} else {
				ctx.Infof("ssrpc recv {cmd:%v, uid:%v, rid:%v}", uint32(ctx.Cmd), ctx.Uid(), ctx.Rid())
			}
			rsp, err := next(ctx, req)
			cost := time.Since(start)
			if err != nil {
				ctx.Warningf("ssrpc done {cmd:%v, cost:%v} err=%v", uint32(ctx.Cmd), cost, err)
				return nil, err
			}
			ctx.Infof("ssrpc done {cmd:%v, cost:%v}", uint32(ctx.Cmd), cost)
			return rsp, nil
		}
	}
}


