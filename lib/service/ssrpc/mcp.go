package ssrpc

import (
	"github.com/golang/protobuf/proto"
)

// MCP is a generic capability provider (tool-calling gateway).
//
// Phase A+ design goal:
// - Business code can call ctx.MCP.CallTool(...) if enabled.
// - Transport does not need to know about MCP; middleware can inject and guard it.
type MCP interface {
	CallTool(name string, input any) (any, error)
}

// MCPAttach injects an MCP implementation into Context.
func MCPAttach(m MCP) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if ctx != nil {
				ctx.MCP = m
			}
			return next(ctx, req)
		}
	}
}

// MCPGuard is a placeholder guard middleware.
//
// In Phase A+, this should enforce per-cmd / per-service allow-lists to prevent misuse.
func MCPGuard() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			return next(ctx, req)
		}
	}
}


