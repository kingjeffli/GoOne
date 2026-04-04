package application

import (
	"os"
	"sync/atomic"
	"testing"
	"time"
)

type testAppHandler struct {
	onInit   func() error
	onReload func() error
	onProc   func() bool
	onTick   func(lastMs, nowMs int64)
	onExit   func()
}

func (h *testAppHandler) OnInit() error {
	if h.onInit != nil {
		return h.onInit()
	}
	return nil
}

func (h *testAppHandler) OnReload() error {
	if h.onReload != nil {
		return h.onReload()
	}
	return nil
}

func (h *testAppHandler) OnProc() bool {
	if h.onProc != nil {
		return h.onProc()
	}
	return true
}

func (h *testAppHandler) OnTick(lastMs, nowMs int64) {
	if h.onTick != nil {
		h.onTick(lastMs, nowMs)
	}
}

func (h *testAppHandler) OnExit() {
	if h.onExit != nil {
		h.onExit()
	}
}

func withTestApplication(t *testing.T, handler AppInterface, tickIntervalMs int64) {
	t.Helper()
	oldApp := app
	oldSig := sig
	t.Cleanup(func() {
		app = oldApp
		sig = oldSig
	})

	sig = make(chan os.Signal, 8)
	app = Application{
		appHandler:    handler,
		tickInterval:  tickIntervalMs,
		lastTickTime: 0,
	}
}

func TestRunInitialTickAndExit(t *testing.T) {
	tickCh := make(chan [2]int64, 1)
	exitCh := make(chan struct{}, 1)
	done := make(chan struct{})

	h := &testAppHandler{
		onProc: func() bool {
			return true
		},
		onTick: func(lastMs, nowMs int64) {
			select {
			case tickCh <- [2]int64{lastMs, nowMs}:
			default:
			}
		},
		onExit: func() {
			select {
			case exitCh <- struct{}{}:
			default:
			}
		},
	}
	withTestApplication(t, h, 50)

	go func() {
		Run()
		close(done)
	}()

	select {
	case tick := <-tickCh:
		if tick[0] != 0 {
			t.Fatalf("expected first tick lastMs=0, got %d", tick[0])
		}
		if tick[1] <= 0 {
			t.Fatalf("expected first tick nowMs>0, got %d", tick[1])
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for initial tick")
	}

	sig <- os.Interrupt

	select {
	case <-exitCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for OnExit")
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for Run to return")
	}
}

func TestRunIdleProcBackoff(t *testing.T) {
	var procCount atomic.Int64
	done := make(chan struct{})

	h := &testAppHandler{
		onProc: func() bool {
			procCount.Add(1)
			return true
		},
	}
	withTestApplication(t, h, 100)

	go func() {
		Run()
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	sig <- os.Interrupt

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for Run to return")
	}

	got := procCount.Load()
	if got < 2 {
		t.Fatalf("expected OnProc to run at least twice, got %d", got)
	}
	if got > 20 {
		t.Fatalf("expected idle OnProc polling to be bounded, got %d calls", got)
	}
}

