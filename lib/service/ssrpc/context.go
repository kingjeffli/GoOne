package ssrpc

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

type Transport string

const (
	TransportSS   Transport = "sspack"
	TransportHTTP Transport = "http"
	TransportWS   Transport = "ws"
	TransportGRPC Transport = "grpc"
)

// Context is the unified request context for GoOne RPC handlers (Phase A).
//
// It wraps the existing cmd_handler.IContext (implemented by Transaction),
// and adds metadata that is useful for middleware and logging.
type Context struct {
	cmd_handler.IContext

	Transport Transport
	Cmd       g1_protocol.CMD
	MCP       MCP // optional capability provider (Phase A+)

	// Method is the logical RPC method name (typically "Service.Method" or comment).
	Method string

	// Flags propagated from ssrpc.MethodDesc (set by WrapUnary).
	AuthRequired bool
	SignRequired bool

	// TraceTags are optional extra tags for tracing/metrics.
	TraceTags map[string]string

	// UIDLocker can be attached via middleware; UIDLock() will prefer it when present.
	UIDLocker UIDLocker
}

func WrapIContext(ic cmd_handler.IContext, cmd g1_protocol.CMD) *Context {
	return &Context{
		IContext:   ic,
		Transport: TransportSS,
		Cmd:       cmd,
	}
}


