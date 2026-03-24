package ssrpc

import (
	"context"
	"strings"
	"sync"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type httpRouteKey struct {
	method string
	path   string
}

// grpcMethodEntry describes a single gRPC method registered in the Dispatcher.
type grpcMethodEntry struct {
	ServiceName   string // e.g. "game.main.v1.MainService"
	MethodName    string // e.g. "Login"
	IsServerStream bool

	// Unary handler (non-nil when !IsServerStream).
	UnaryHandler GRPCUnaryHandler
	// Stream handler + descriptor (non-nil when IsServerStream).
	StreamHandler GRPCStreamHandler
	StreamDesc    *grpc.StreamDesc
}

// Dispatcher is the unified registration center for all transports.
//
// It covers four transport paths:
// - cmd -> TransactionMgr handler (SSPacket)
// - http(method+path) -> gin.HandlerFunc
// - ws(cmd) -> CmdHandlerFunc (CSPacket via WebSocket)
// - grpc(service/method) -> GRPCUnaryHandler / GRPCStreamHandler
type Dispatcher struct {
	mu sync.RWMutex

	cmdHandlers  map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc
	httpHandlers map[httpRouteKey]gin.HandlerFunc
	wsHandlers   map[uint32]cmd_handler.CmdHandlerFunc
	grpcMethods  []grpcMethodEntry
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		cmdHandlers:  make(map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc),
		httpHandlers: make(map[httpRouteKey]gin.HandlerFunc),
		wsHandlers:   make(map[uint32]cmd_handler.CmdHandlerFunc),
	}
}

func (d *Dispatcher) RegisterCmd(cmd g1_protocol.CMD, h cmd_handler.CmdHandlerFunc) {
	if d == nil || h == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cmdHandlers[cmd] = h
}

func (d *Dispatcher) RegisterHTTP(method, path string, h gin.HandlerFunc) {
	if d == nil || h == nil {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "POST"
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.httpHandlers[httpRouteKey{method: method, path: path}] = h
}

// MountGin registers all known HTTP routes onto the given gin router/group.
func (d *Dispatcher) MountGin(r gin.IRoutes) {
	if d == nil || r == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	for k, h := range d.httpHandlers {
		r.Handle(k.method, k.path, h)
	}
}

// RegisterToTransactionMgr registers all known cmd handlers into the TransactionMgr.
func (d *Dispatcher) RegisterToTransactionMgr(mgr transaction.ITransactionMgr) {
	if d == nil || mgr == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	for cmd, h := range d.cmdHandlers {
		mgr.RegisterCmd(cmd, h)
	}
}

// ---------------------------------------------------------------------------
// WS (CSPacket) transport
// ---------------------------------------------------------------------------

// RegisterWS registers a WS (CSPacket) handler for a given cmd.
//
// The key is uint32 (matching CSPacketHeader.Cmd) to avoid a cast on every
// hot-path dispatch.
func (d *Dispatcher) RegisterWS(cmd uint32, h cmd_handler.CmdHandlerFunc) {
	if d == nil || h == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.wsHandlers[cmd] = h
}

// DispatchWS looks up a WS handler by cmd and, if found, executes it.
//
// Returns (errorCode, true) when a handler was found and executed.
// Returns (0, false) when no handler is registered for the cmd -- the caller
// should fall through to its default forwarding logic (e.g. router.SendMsgByConn).
func (d *Dispatcher) DispatchWS(ic cmd_handler.IContext, cmd uint32, body []byte) (g1_protocol.ErrorCode, bool) {
	if d == nil {
		return 0, false
	}
	d.mu.RLock()
	h, ok := d.wsHandlers[cmd]
	d.mu.RUnlock()
	if !ok {
		return 0, false
	}
	return h(ic, body), true
}

// ---------------------------------------------------------------------------
// gRPC transport
// ---------------------------------------------------------------------------

// RegisterGRPCUnary registers a gRPC unary handler for the given service/method.
func (d *Dispatcher) RegisterGRPCUnary(serviceName, methodName string, h GRPCUnaryHandler) {
	if d == nil || h == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.grpcMethods = append(d.grpcMethods, grpcMethodEntry{
		ServiceName:  serviceName,
		MethodName:   methodName,
		UnaryHandler: h,
	})
}

// RegisterGRPCStream registers a gRPC server-streaming handler.
func (d *Dispatcher) RegisterGRPCStream(serviceName, methodName string, h GRPCStreamHandler) {
	if d == nil || h == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.grpcMethods = append(d.grpcMethods, grpcMethodEntry{
		ServiceName:    serviceName,
		MethodName:     methodName,
		IsServerStream: true,
		StreamHandler:  h,
		StreamDesc: &grpc.StreamDesc{
			StreamName:    methodName,
			ServerStreams:  true,
			ClientStreams:  false,
		},
	})
}

// MountGRPC registers all collected gRPC methods onto the given grpc.Server.
//
// It groups methods by service name and calls srv.RegisterService for each.
func (d *Dispatcher) MountGRPC(srv *grpc.Server) {
	if d == nil || srv == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	type svcBucket struct {
		methods []grpc.MethodDesc
		streams []grpc.StreamDesc
	}
	grouped := make(map[string]*svcBucket)

	for i := range d.grpcMethods {
		m := &d.grpcMethods[i]
		bucket, ok := grouped[m.ServiceName]
		if !ok {
			bucket = &svcBucket{}
			grouped[m.ServiceName] = bucket
		}
		if m.IsServerStream {
			sd := *m.StreamDesc
			handler := m.StreamHandler // capture for closure
			sd.Handler = func(srv any, stream grpc.ServerStream) error {
				return handler(srv, stream)
			}
			bucket.streams = append(bucket.streams, sd)
		} else {
			handler := m.UnaryHandler // capture for closure
			bucket.methods = append(bucket.methods, grpc.MethodDesc{
				MethodName: m.MethodName,
				Handler:    makeGRPCMethodHandler(handler),
			})
		}
	}

	for svcName, bucket := range grouped {
		desc := &grpc.ServiceDesc{
			ServiceName: svcName,
			HandlerType: (*any)(nil),
			Methods:     bucket.methods,
			Streams:     bucket.streams,
			Metadata:    svcName,
		}
		srv.RegisterService(desc, nil)
	}
}

// makeGRPCMethodHandler adapts a GRPCUnaryHandler to the grpc.MethodDesc.Handler
// signature. The dec callback receives a pointer to any; gRPC populates it with
// the decoded proto message.
func makeGRPCMethodHandler(h GRPCUnaryHandler) func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return func(_ any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		var req any
		if err := dec(&req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return h(ctx, req)
		}
		info := &grpc.UnaryServerInfo{}
		return interceptor(ctx, req, info, func(ctx context.Context, r any) (any, error) {
			return h(ctx, r)
		})
	}
}

