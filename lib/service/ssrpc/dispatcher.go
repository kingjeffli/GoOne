package ssrpc

import (
	"strings"
	"sync"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/gin-gonic/gin"
)

type httpRouteKey struct {
	method string
	path   string
}

// Dispatcher is a Phase-2 unified registration center for multiple transports.
//
// It is intentionally small:
// - cmd -> TransactionMgr handler (SSPacket)
// - http(method+path) -> gin.HandlerFunc
//
// Future: ws/grpc registrations can be added without changing business service impls.
type Dispatcher struct {
	mu sync.RWMutex

	cmdHandlers  map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc
	httpHandlers map[httpRouteKey]gin.HandlerFunc
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		cmdHandlers:  make(map[g1_protocol.CMD]cmd_handler.CmdHandlerFunc),
		httpHandlers: make(map[httpRouteKey]gin.HandlerFunc),
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

