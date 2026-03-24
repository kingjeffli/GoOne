package ssrpc

import (
	"fmt"
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

// MCPGuardFunc decides whether a tool call is allowed.
// Return nil to allow, otherwise return an error to block.
type MCPGuardFunc func(ctx *Context, tool string, input any) error

type guardedMCP struct {
	ctx   *Context
	inner MCP
	guard MCPGuardFunc
}

func (g *guardedMCP) CallTool(name string, input any) (any, error) {
	if g == nil || g.inner == nil {
		return nil, fmt.Errorf("mcp not configured")
	}
	if g.guard != nil {
		if err := g.guard(g.ctx, name, input); err != nil {
			return nil, err
		}
	}
	return g.inner.CallTool(name, input)
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

// MCPGuardWith wraps ctx.MCP with a guard function (no-op if ctx.MCP is nil).
func MCPGuardWith(guard MCPGuardFunc) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if ctx != nil && ctx.MCP != nil && guard != nil {
				ctx.MCP = &guardedMCP{ctx: ctx, inner: ctx.MCP, guard: guard}
			}
			return next(ctx, req)
		}
	}
}

// MCPGuard keeps backward-compatibility and defaults to allowing all calls.
func MCPGuard() Middleware { return MCPGuardWith(nil) }


