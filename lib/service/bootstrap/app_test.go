package bootstrap

import (
	"context"
	"errors"
	"testing"
)

func TestServiceAppSnapshotsTrackLifecycle(t *testing.T) {
	app := NewServiceApp(Options{
		ServiceName: "testsvr",
		LoadConfig:  func() error { return nil },
		BuildInfo: func() BuildInfo {
			return BuildInfo{Version: "test-version"}
		},
		ComponentStatuses: func() []ComponentStatus {
			return []ComponentStatus{{
				Name:    "redis",
				State:   componentStateReady,
				Ready:   true,
				Message: "connected",
			}}
		},
		InitDeps:         func() error { return nil },
		RegisterHandlers: func() error { return nil },
		StartRuntime:     func() error { return nil },
		OnShutdown:       func(ctx context.Context) error { return nil },
	})

	if err := app.OnInit(); err != nil {
		t.Fatalf("expected init success, got %v", err)
	}

	info := app.infoSnapshot()
	if !info.Alive || !info.Ready {
		t.Fatalf("expected app to be alive and ready, got %+v", info)
	}
	if info.Phase != phaseReady {
		t.Fatalf("expected phase %q, got %q", phaseReady, info.Phase)
	}
	if info.Build.Version != "test-version" {
		t.Fatalf("expected merged build version, got %+v", info.Build)
	}

	components := app.componentsSnapshot()
	assertComponentState(t, components, componentConfig, componentStateReady)
	assertComponentState(t, components, componentLogger, componentStateSkipped)
	assertComponentState(t, components, componentAdminServer, componentStateSkipped)
	assertComponentState(t, components, componentDependencies, componentStateReady)
	assertComponentState(t, components, componentHandlers, componentStateReady)
	assertComponentState(t, components, componentRuntime, componentStateReady)
	assertComponentState(t, components, "redis", componentStateReady)

	if err := app.OnReload(); err != nil {
		t.Fatalf("expected reload success, got %v", err)
	}
	info = app.infoSnapshot()
	if info.LastReloadAt == "" {
		t.Fatalf("expected last_reload_at to be set after reload")
	}
	if info.LastError != "" {
		t.Fatalf("expected no last error after successful reload, got %q", info.LastError)
	}

	app.OnExit()
	info = app.infoSnapshot()
	if info.Alive || info.Ready {
		t.Fatalf("expected app to be stopped after exit, got %+v", info)
	}
	if info.Phase != phaseStopped {
		t.Fatalf("expected phase %q after exit, got %q", phaseStopped, info.Phase)
	}
	assertComponentState(t, app.componentsSnapshot(), componentRuntime, componentStateStopped)
}

func TestServiceAppInitFailureCapturesComponentError(t *testing.T) {
	app := NewServiceApp(Options{
		ServiceName:      "broken",
		LoadConfig:       func() error { return nil },
		InitDeps:         func() error { return errors.New("redis unavailable") },
		RegisterHandlers: func() error { return nil },
		StartRuntime:     func() error { return nil },
	})

	err := app.OnInit()
	if err == nil {
		t.Fatal("expected init failure")
	}

	info := app.infoSnapshot()
	if info.Phase != phaseInitFailed {
		t.Fatalf("expected phase %q, got %q", phaseInitFailed, info.Phase)
	}
	if info.LastError != "redis unavailable" {
		t.Fatalf("expected last error to be recorded, got %q", info.LastError)
	}
	if info.Alive || info.Ready {
		t.Fatalf("expected app to be unavailable after init failure, got %+v", info)
	}
	assertComponentState(t, app.componentsSnapshot(), componentDependencies, componentStateFailed)
}

func assertComponentState(t *testing.T, components []ComponentStatus, name, expected string) {
	t.Helper()
	for _, component := range components {
		if component.Name == name {
			if component.State != expected {
				t.Fatalf("expected component %s state %s, got %+v", name, expected, component)
			}
			return
		}
	}
	t.Fatalf("component %s not found", name)
}
