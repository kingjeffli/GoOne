package ssrpc

import (
	"time"

	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// MetricsRecorder is an optional hook to record ssrpc handler metrics.
// The project currently has no built-in metrics backend; wire your own recorder.
type MetricsRecorder interface {
	Observe(cmd g1_protocol.CMD, cost time.Duration, code g1_protocol.ErrorCode)
}

// MetricsRecorderWithContext is an optional richer hook for recorders that want
// access to session, method, and transport metadata without breaking the older
// MetricsRecorder interface.
type MetricsRecorderWithContext interface {
	ObserveWithContext(ctx *Context, cost time.Duration, code g1_protocol.ErrorCode)
}

// Metrics creates a middleware that records duration + final error code.
func Metrics(rec MetricsRecorder) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if rec == nil {
				return next(ctx, req)
			}
			start := time.Now()
			rsp, err := next(ctx, req)
			code := ToErrorCode(err)
			cost := time.Since(start)
			if rich, ok := any(rec).(MetricsRecorderWithContext); ok {
				rich.ObserveWithContext(ctx, cost, code)
				return rsp, err
			}
			if ctx != nil {
				rec.Observe(ctx.Cmd, cost, code)
			} else {
				rec.Observe(0, cost, code)
			}
			return rsp, err
		}
	}
}
