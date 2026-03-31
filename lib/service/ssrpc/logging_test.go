package ssrpc

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

type captureIContext struct {
	fakeIContext
	infos  []string
	debugs []string
	warns  []string
}

func (c *captureIContext) Infof(format string, args ...interface{}) {
	c.infos = append(c.infos, fmt.Sprintf(format, args...))
}

func (c *captureIContext) Debugf(format string, args ...interface{}) {
	c.debugs = append(c.debugs, fmt.Sprintf(format, args...))
}

func (c *captureIContext) Warningf(format string, args ...interface{}) {
	c.warns = append(c.warns, fmt.Sprintf(format, args...))
}

func TestLoggingWithOptions_FastRequestUsesDebug(t *testing.T) {
	ic := &captureIContext{fakeIContext: fakeIContext{uid: 7}}
	ctx := WrapIContext(ic, 123)
	ctx.SetMethod("Room.Tick")

	h := LoggingWithOptions(LoggingOptions{
		SlowThreshold: time.Second,
	})(func(ctx *Context, req proto.Message) (proto.Message, error) {
		return nil, nil
	})

	if _, err := h(ctx, &fakePB{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(ic.debugs) != 1 || len(ic.infos) != 0 || len(ic.warns) != 0 {
		t.Fatalf("unexpected log counts debug=%d info=%d warn=%d", len(ic.debugs), len(ic.infos), len(ic.warns))
	}
	if !strings.Contains(ic.debugs[0], "ssrpc ok") || !strings.Contains(ic.debugs[0], "reqType:*ssrpc.fakePB") {
		t.Fatalf("unexpected debug log: %q", ic.debugs[0])
	}
}

func TestLoggingWithOptions_SlowRequestUsesInfo(t *testing.T) {
	ic := &captureIContext{fakeIContext: fakeIContext{uid: 8}}
	ctx := WrapIContext(ic, 456)
	ctx.SetMethod("Room.SlowTick")

	h := LoggingWithOptions(LoggingOptions{
		SlowThreshold: time.Millisecond,
	})(func(ctx *Context, req proto.Message) (proto.Message, error) {
		time.Sleep(2 * time.Millisecond)
		return nil, nil
	})

	if _, err := h(ctx, &fakePB{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(ic.debugs) != 0 || len(ic.infos) != 1 || len(ic.warns) != 0 {
		t.Fatalf("unexpected log counts debug=%d info=%d warn=%d", len(ic.debugs), len(ic.infos), len(ic.warns))
	}
	if !strings.Contains(ic.infos[0], "ssrpc slow") || !strings.Contains(ic.infos[0], "slowThreshold:1ms") {
		t.Fatalf("unexpected info log: %q", ic.infos[0])
	}
}

func TestLoggingWithOptions_ErrorUsesWarn(t *testing.T) {
	ic := &captureIContext{fakeIContext: fakeIContext{uid: 9}}
	ctx := WrapIContext(ic, 789)
	ctx.SetMethod("Room.Fail")

	boom := errors.New("boom")
	h := LoggingWithOptions(LoggingOptions{
		SlowThreshold: time.Second,
	})(func(ctx *Context, req proto.Message) (proto.Message, error) {
		return nil, boom
	})

	if _, err := h(ctx, &fakePB{}); !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
	if len(ic.debugs) != 0 || len(ic.infos) != 0 || len(ic.warns) != 1 {
		t.Fatalf("unexpected log counts debug=%d info=%d warn=%d", len(ic.debugs), len(ic.infos), len(ic.warns))
	}
	if !strings.Contains(ic.warns[0], "ssrpc err") || !strings.Contains(ic.warns[0], "boom") {
		t.Fatalf("unexpected warn log: %q", ic.warns[0])
	}
}
