package ssrpc

import (
	"context"
	"testing"

	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ---------------------------------------------------------------------------
// WrapGRPCUnary tests
// ---------------------------------------------------------------------------

func TestWrapGRPCUnary_SetsTransportGRPC(t *testing.T) {
	var gotTransport Transport
	desc := MethodDesc{Cmd: 100, Name: "test.Login"}
	h := WrapGRPCUnary(desc, nil,
		func(ctx *Context, req any) (any, error) {
			gotTransport = ctx.Transport
			return nil, nil
		},
	)

	ctx := context.Background()
	_, err := h(ctx, &fakePB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotTransport != TransportGRPC {
		t.Fatalf("expected transport %q, got %q", TransportGRPC, gotTransport)
	}
}

func TestWrapGRPCUnary_ExtractsMetadata(t *testing.T) {
	var gotUid uint64
	var gotZone uint32
	desc := MethodDesc{Cmd: 200, Name: "test.Info"}
	h := WrapGRPCUnary(desc, nil,
		func(ctx *Context, req any) (any, error) {
			gotUid = ctx.Uid()
			gotZone = ctx.Zone()
			return nil, nil
		},
	)

	md := metadata.New(map[string]string{"x-uid": "12345", "x-zone": "7"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := h(ctx, &fakePB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gotUid != 12345 {
		t.Fatalf("uid = %d, want 12345", gotUid)
	}
	if gotZone != 7 {
		t.Fatalf("zone = %d, want 7", gotZone)
	}
}

func TestWrapGRPCUnary_RunsMiddleware(t *testing.T) {
	var order []string
	mw := func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			order = append(order, "mw")
			rsp, err := next(ctx, req)
			order = append(order, "mw-after")
			return rsp, err
		}
	}

	desc := MethodDesc{Cmd: 300, Name: "test.MW"}
	h := WrapGRPCUnary(desc, []Middleware{mw},
		func(ctx *Context, req any) (any, error) {
			order = append(order, "handler")
			return nil, nil
		},
	)

	_, err := h(context.Background(), &fakePB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(order) != 3 || order[0] != "mw" || order[1] != "handler" || order[2] != "mw-after" {
		t.Fatalf("middleware order: %v", order)
	}
}

func TestWrapGRPCUnary_PropagatesMethodDesc(t *testing.T) {
	var gotMethod string
	var gotAuth, gotSign bool
	desc := MethodDesc{Cmd: 400, Name: "test.Desc", Auth: true, Sign: true}
	h := WrapGRPCUnary(desc, nil,
		func(ctx *Context, req any) (any, error) {
			gotMethod = ctx.Method
			gotAuth = ctx.AuthRequired
			gotSign = ctx.SignRequired
			return nil, nil
		},
	)

	h(context.Background(), &fakePB{})
	if gotMethod != "test.Desc" || !gotAuth || !gotSign {
		t.Fatalf("method=%q auth=%v sign=%v", gotMethod, gotAuth, gotSign)
	}
}

func TestWrapGRPCUnary_ErrorMapping(t *testing.T) {
	desc := MethodDesc{Cmd: 500, Name: "test.Err"}
	h := WrapGRPCUnary(desc, nil,
		func(ctx *Context, req any) (any, error) {
			return nil, E(g1_protocol.ErrorCode_ERR_ARGV, "bad param")
		},
	)

	_, err := h(context.Background(), &fakePB{})
	if err == nil {
		t.Fatal("expected error")
	}
	code := FromGRPCError(err)
	if code != g1_protocol.ErrorCode_ERR_ARGV {
		t.Fatalf("expected ERR_ARGV, got %v", code)
	}
}

func TestWrapGRPCUnary_NilReq_ReturnsError(t *testing.T) {
	desc := MethodDesc{Cmd: 600}
	h := WrapGRPCUnary(desc, nil,
		func(ctx *Context, req any) (any, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	)

	_, err := h(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil req")
	}
}

// ---------------------------------------------------------------------------
// Dispatcher gRPC tests
// ---------------------------------------------------------------------------

func TestDispatcher_RegisterGRPCUnary(t *testing.T) {
	d := NewDispatcher()
	called := false

	d.RegisterGRPCUnary("TestService", "TestMethod", func(ctx context.Context, req any) (any, error) {
		called = true
		return nil, nil
	})

	if len(d.grpcMethods) != 1 {
		t.Fatalf("expected 1 grpc method, got %d", len(d.grpcMethods))
	}
	m := d.grpcMethods[0]
	if m.ServiceName != "TestService" || m.MethodName != "TestMethod" {
		t.Fatalf("unexpected service/method: %s/%s", m.ServiceName, m.MethodName)
	}
	if m.IsServerStream {
		t.Fatal("expected unary, got stream")
	}

	_, err := m.UnaryHandler(context.Background(), &fakePB{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatal("handler not called")
	}
}

func TestDispatcher_RegisterGRPCStream(t *testing.T) {
	d := NewDispatcher()
	d.RegisterGRPCStream("TestService", "StreamMethod", func(_ any, _ grpc.ServerStream) error {
		return nil
	})

	if len(d.grpcMethods) != 1 {
		t.Fatalf("expected 1 grpc method, got %d", len(d.grpcMethods))
	}
	m := d.grpcMethods[0]
	if !m.IsServerStream {
		t.Fatal("expected server stream")
	}
	if m.StreamDesc == nil {
		t.Fatal("expected StreamDesc to be non-nil")
	}
}

func TestDispatcher_RegisterGRPCUnary_NilDispatcher(t *testing.T) {
	var d *Dispatcher
	// Should not panic.
	d.RegisterGRPCUnary("Svc", "M", func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
}

func TestDispatcher_RegisterGRPCUnary_NilHandler(t *testing.T) {
	d := NewDispatcher()
	d.RegisterGRPCUnary("Svc", "M", nil)
	if len(d.grpcMethods) != 0 {
		t.Fatal("nil handler should not be registered")
	}
}
