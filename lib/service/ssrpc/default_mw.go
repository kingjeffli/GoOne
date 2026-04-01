package ssrpc

// DefaultMWOptions configures the default middleware chain for ssrpc servers.
//
// The defaults are intentionally conservative:
// - Recover/Logging are always included
// - Trace is included (currently a no-op placeholder)
// - Metrics is included (built-in Prometheus recorder when Metrics=nil)
// - MCP is attached/guarded only when MCP is non-nil
// - Extra middlewares are appended at the end
type DefaultMWOptions struct {
	Trace     TraceProvider
	Auth      Authenticator
	Sign      SignVerifier
	UIDLocker UIDLocker
	Metrics   MetricsRecorder
	MCP       MCP
	MCPGuard  MCPGuardFunc
	Extra     []Middleware
}

// DefaultMiddlewares returns a standard middleware chain for SSPacket RPC.
func DefaultMiddlewares(opts DefaultMWOptions) []Middleware {
	recorder := opts.Metrics
	if recorder == nil {
		recorder = DefaultMetricsRecorder()
	}
	mws := []Middleware{
		Recover(),
		Logging(),
		TraceWith(opts.Trace),
		AuthWith(opts.Auth),
		SignWith(opts.Sign),
		UIDLockAttach(opts.UIDLocker),
		Metrics(recorder),
	}
	if opts.MCP != nil {
		mws = append(mws, MCPAttach(opts.MCP), MCPGuardWith(opts.MCPGuard))
	}
	if len(opts.Extra) > 0 {
		mws = append(mws, opts.Extra...)
	}
	return mws
}
