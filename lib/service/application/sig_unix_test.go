//go:build !windows
// +build !windows

package application

import (
	"syscall"
	"testing"
	"time"
)

func TestRunReloadSignalKeepsProcessAlive(t *testing.T) {
	reloadCh := make(chan struct{}, 1)
	exitCh := make(chan struct{}, 1)
	done := make(chan struct{})

	h := &testAppHandler{
		onProc: func() bool {
			return true
		},
		onReload: func() error {
			select {
			case reloadCh <- struct{}{}:
			default:
			}
			return nil
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

	sig <- syscall.SIGUSR1

	select {
	case <-reloadCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for reload callback")
	}

	select {
	case <-done:
		t.Fatal("Run should continue after reload signal")
	case <-time.After(40 * time.Millisecond):
	}

	sig <- syscall.SIGTERM

	select {
	case <-exitCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for exit callback")
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for Run to return after exit signal")
	}
}

