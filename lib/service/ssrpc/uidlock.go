package ssrpc

import "github.com/golang/protobuf/proto"

// UIDLock is a placeholder middleware for Phase A+.
//
// Today, TransactionMgr's uid serialization is configured globally (InitAndRun(useUidLock=true)).
// In Phase A+, we'll make uid_lock option actually enforce a per-method serialization strategy.
func UIDLock() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			return next(ctx, req)
		}
	}
}


