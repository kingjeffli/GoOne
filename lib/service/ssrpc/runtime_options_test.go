package ssrpc

import (
	"testing"

	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

type fakeIContext struct {
	uid uint64
}

func (f *fakeIContext) Uid() uint64        { return f.uid }
func (f *fakeIContext) Zone() uint32       { return 0 }
func (f *fakeIContext) Rid() uint64        { return 0 }
func (f *fakeIContext) OriSrcBusId() uint32 { return 0 }
func (f *fakeIContext) Ip() uint32         { return 0 }
func (f *fakeIContext) Flag() uint32       { return 0 }

func (f *fakeIContext) ParseMsg(data []byte, msg proto.Message) error { return nil }

func (f *fakeIContext) CallMsgBySvrType(svrType uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	panic("not used")
}
func (f *fakeIContext) CallMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	panic("not used")
}
func (f *fakeIContext) CallOtherMsgBySvrType(svrType uint32, routerId, uid uint64, zone uint32, cmd g1_protocol.CMD, req proto.Message, rsp proto.Message) error {
	panic("not used")
}
func (f *fakeIContext) SendMsgBack(pbMsg proto.Message) { /* no-op */ }
func (f *fakeIContext) SendMsgByServerType(svrType uint32, cmd g1_protocol.CMD, req proto.Message) error {
	panic("not used")
}
func (f *fakeIContext) SendMsgByRouter(svrType uint32, routerId uint64, cmd g1_protocol.CMD, req proto.Message) error {
	panic("not used")
}
func (f *fakeIContext) Errorf(format string, args ...interface{})   {}
func (f *fakeIContext) Warningf(format string, args ...interface{}) {}
func (f *fakeIContext) Infof(format string, args ...interface{})    {}
func (f *fakeIContext) Debugf(format string, args ...interface{})   {}

var _ cmd_handler.IContext = (*fakeIContext)(nil)

type fakeTraceProvider struct {
	gotTags map[string]string
	finish  int
}

func (f *fakeTraceProvider) Start(ctx *Context, tags map[string]string) func(err error) {
	f.gotTags = tags
	return func(err error) { f.finish++ }
}

type fakeLocker struct {
	lockCnt   int
	unlockCnt int
}

func (l *fakeLocker) Lock(uid uint64) func() {
	l.lockCnt++
	return func() { l.unlockCnt++ }
}

func TestTraceWith_MergesTagsAndFinishes(t *testing.T) {
	tp := &fakeTraceProvider{}
	h := TraceWith(tp)(func(ctx *Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	})

	_, err := h(&Context{
		Cmd:    123,
		Method: "S.M",
		TraceTags: map[string]string{
			"a": "1",
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tp.finish != 1 {
		t.Fatalf("expected finish called once, got %d", tp.finish)
	}
	if tp.gotTags["method"] != "S.M" || tp.gotTags["a"] != "1" || tp.gotTags["cmd"] == "" {
		t.Fatalf("unexpected tags: %#v", tp.gotTags)
	}
}

func TestAuthWith_RequiredButMissingAuthenticator(t *testing.T) {
	h := AuthWith(nil)(func(ctx *Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	})
	_, err := h(&Context{AuthRequired: true}, nil)
	if ToErrorCode(err) != g1_protocol.ErrorCode_ERR_INTERNAL {
		t.Fatalf("expected ERR_INTERNAL, got %v err=%v", ToErrorCode(err), err)
	}
}

func TestSignWith_RequiredButMissingVerifier(t *testing.T) {
	h := SignWith(nil)(func(ctx *Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	})
	_, err := h(&Context{SignRequired: true}, nil)
	if ToErrorCode(err) != g1_protocol.ErrorCode_ERR_INTERNAL {
		t.Fatalf("expected ERR_INTERNAL, got %v err=%v", ToErrorCode(err), err)
	}
}

func TestUIDLock_UsesContextLocker(t *testing.T) {
	locker := &fakeLocker{}
	ctx := &Context{IContext: &fakeIContext{uid: 42}, UIDLocker: locker}
	h := UIDLock()(func(ctx *Context, req proto.Message) (proto.Message, error) { return nil, nil })
	_, err := h(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if locker.lockCnt != 1 || locker.unlockCnt != 1 {
		t.Fatalf("expected lock/unlock once, got lock=%d unlock=%d", locker.lockCnt, locker.unlockCnt)
	}
}

type fakeMCP struct {
	called int
}

func (m *fakeMCP) CallTool(name string, input any) (any, error) {
	m.called++
	return "ok", nil
}

func TestMCPGuardWith_BlocksCall(t *testing.T) {
	m := &fakeMCP{}
	ctx := &Context{MCP: m}

	h := MCPGuardWith(func(ctx *Context, tool string, input any) error {
		return E(g1_protocol.ErrorCode_ERR_INTERNAL, "blocked")
	})(func(ctx *Context, req proto.Message) (proto.Message, error) {
		_, err := ctx.MCP.CallTool("x", nil)
		return nil, err
	})

	_, err := h(ctx, nil)
	if ToErrorCode(err) != g1_protocol.ErrorCode_ERR_INTERNAL {
		t.Fatalf("expected blocked ERR_INTERNAL, got %v err=%v", ToErrorCode(err), err)
	}
	if m.called != 0 {
		t.Fatalf("expected inner MCP not called, got %d", m.called)
	}
}


