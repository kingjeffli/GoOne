package ssrpc

import (
	"hash/fnv"
	"sync"

	"github.com/golang/protobuf/proto"
)

// UIDLocker provides a per-uid lock primitive.
// The returned unlock func MUST be called.
type UIDLocker interface {
	Lock(uid uint64) (unlock func())
}

// stripedUIDLocker is a low-overhead locker that avoids unbounded uid->mutex growth.
// Different uids may contend if they hash to the same stripe.
type stripedUIDLocker struct {
	stripes []sync.Mutex
}

func NewStripedUIDLocker(n int) UIDLocker {
	if n <= 0 {
		n = 1024
	}
	return &stripedUIDLocker{stripes: make([]sync.Mutex, n)}
}

func (l *stripedUIDLocker) Lock(uid uint64) func() {
	idx := int(hash64(uid) % uint64(len(l.stripes)))
	l.stripes[idx].Lock()
	return l.stripes[idx].Unlock
}

func hash64(v uint64) uint64 {
	h := fnv.New64a()
	var b [8]byte
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
	_, _ = h.Write(b[:])
	return h.Sum64()
}

var defaultUIDLocker UIDLocker = NewStripedUIDLocker(1024)

// SetDefaultUIDLocker sets the process-wide default UIDLocker used by UIDLock().
func SetDefaultUIDLocker(l UIDLocker) {
	if l != nil {
		defaultUIDLocker = l
	}
}

// UIDLockAttach injects a UIDLocker into Context so UIDLock can use it.
func UIDLockAttach(l UIDLocker) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if ctx != nil {
				ctx.UIDLocker = l
			}
			return next(ctx, req)
		}
	}
}

// UIDLock enforces per-uid serialization when invoked (generator enables it when uid_lock=true).
//
// Default key: ctx.Uid(); if uid==0, it's a no-op.
func UIDLock() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context, req proto.Message) (proto.Message, error) {
			if ctx == nil {
				return next(ctx, req)
			}
			uid := ctx.Uid()
			if uid == 0 {
				return next(ctx, req)
			}
			locker := ctx.UIDLocker
			if locker == nil {
				locker = defaultUIDLocker
			}
			if locker == nil {
				return next(ctx, req)
			}
			unlock := locker.Lock(uid)
			if unlock != nil {
				defer unlock()
			}
			return next(ctx, req)
		}
	}
}


