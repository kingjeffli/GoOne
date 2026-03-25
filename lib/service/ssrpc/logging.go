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
			if ctx == nil {
				return next(ctx, req)
			}
			start := time.Now()
			session := ctx.Session
			transID := session.TransID
			if transID != 0 {
				ctx.Infof("ssrpc recv {transport:%s, method:%s, cmd:%v, uid:%v, rid:%v, trans:%v}", session.Transport, ctx.Method, uint32(ctx.Cmd), session.UID, session.RID, transID)
			} else {
				ctx.Infof("ssrpc recv {transport:%s, method:%s, cmd:%v, uid:%v, rid:%v}", session.Transport, ctx.Method, uint32(ctx.Cmd), session.UID, session.RID)
			}
			rsp, err := next(ctx, req)
			cost := time.Since(start)
			if err != nil {
				ctx.Warningf("ssrpc done {transport:%s, method:%s, cmd:%v, cost:%v} err=%v", session.Transport, ctx.Method, uint32(ctx.Cmd), cost, err)
				return nil, err
			}
			ctx.Infof("ssrpc done {transport:%s, method:%s, cmd:%v, cost:%v}", session.Transport, ctx.Method, uint32(ctx.Cmd), cost)
			return rsp, nil
		}
	}
}
