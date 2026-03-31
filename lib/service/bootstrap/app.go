package bootstrap

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/Iori372552686/GoOne/lib/api/logger"
)

const defaultShutdownTimeout = 5 * time.Second

type LoggerConfig struct {
	Dir   string
	Level string
	Name  string
}

type Options struct {
	ServiceName      string
	LoadConfig       func() error
	LoggerConfig     func() LoggerConfig
	AdminConfig      func() AdminConfig
	InitDeps         func() error
	RegisterHandlers func() error
	StartRuntime     func() error
	OnProc           func() bool
	OnTick           func(lastMs, nowMs int64)
	OnExit           func()
}

type ServiceApp struct {
	options           Options
	admin             *AdminServer
	loggerInitialized bool
	alive             atomic.Bool
	ready             atomic.Bool
}

func NewServiceApp(opts Options) *ServiceApp {
	return &ServiceApp{options: opts}
}

func (a *ServiceApp) OnInit() error {
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
	a.alive.Store(true)
	a.ready.Store(false)

	if err := a.loadConfig(false); err != nil {
		a.alive.Store(false)
		return err
	}
	if err := a.initLogger(); err != nil {
		a.alive.Store(false)
		return err
	}
	if err := a.startAdmin(); err != nil {
		a.alive.Store(false)
		return err
	}
	if err := a.runStep("init deps", a.options.InitDeps); err != nil {
		a.cleanupOnInitError()
		return err
	}
	if err := a.runStep("register handlers", a.options.RegisterHandlers); err != nil {
		a.cleanupOnInitError()
		return err
	}
	if err := a.runStep("start runtime", a.options.StartRuntime); err != nil {
		a.cleanupOnInitError()
		return err
	}

	a.ready.Store(true)
	logger.Infof("%s init success", a.serviceName())
	return nil
}

func (a *ServiceApp) OnReload() error {
	if err := a.loadConfig(a.loggerInitialized); err != nil {
		return err
	}
	logger.Infof("%s reload success", a.serviceName())
	return nil
}

func (a *ServiceApp) OnProc() bool {
	if a.options.OnProc == nil {
		return true
	}
	return a.options.OnProc()
}

func (a *ServiceApp) OnTick(lastMs, nowMs int64) {
	if a.options.OnTick != nil {
		a.options.OnTick(lastMs, nowMs)
	}
}

func (a *ServiceApp) OnExit() {
	a.ready.Store(false)
	a.alive.Store(false)

	if err := a.shutdownAdmin(); err != nil {
		logger.Errorf("%s admin shutdown error | %v", a.serviceName(), err)
	}
	if a.options.OnExit != nil {
		a.options.OnExit()
	}
	logger.Infof("%s shutdown complete", a.serviceName())
	logger.Flush()
}

func (a *ServiceApp) loadConfig(reinitLogger bool) error {
	if a.options.LoadConfig != nil {
		if err := a.options.LoadConfig(); err != nil {
			logger.Errorf("%s load config failed | %v", a.serviceName(), err)
			return err
		}
	}
	if reinitLogger {
		return a.initLogger()
	}
	return nil
}

func (a *ServiceApp) initLogger() error {
	if a.options.LoggerConfig == nil {
		return nil
	}
	cfg := a.options.LoggerConfig()
	if _, err := logger.InitLogger(cfg.Dir, cfg.Level, cfg.Name); err != nil {
		return err
	}
	a.loggerInitialized = true
	return nil
}

func (a *ServiceApp) startAdmin() error {
	if a.options.AdminConfig == nil {
		return nil
	}
	admin, err := StartAdminServer(a.options.AdminConfig(), AdminState{
		IsAlive: func() bool { return a.alive.Load() },
		IsReady: func() bool { return a.ready.Load() },
	})
	if err != nil {
		return err
	}
	a.admin = admin
	return nil
}

func (a *ServiceApp) shutdownAdmin() error {
	if a.admin == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	err := a.admin.Shutdown(ctx)
	a.admin = nil
	return err
}

func (a *ServiceApp) cleanupOnInitError() {
	a.ready.Store(false)
	a.alive.Store(false)
	if err := a.shutdownAdmin(); err != nil {
		logger.Errorf("%s cleanup admin shutdown error | %v", a.serviceName(), err)
	}
}

func (a *ServiceApp) runStep(step string, fn func() error) error {
	if fn == nil {
		return nil
	}
	if err := fn(); err != nil {
		logger.Errorf("%s %s failed | %v", a.serviceName(), step, err)
		return err
	}
	return nil
}

func (a *ServiceApp) serviceName() string {
	if a.options.ServiceName == "" {
		return "service"
	}
	return a.options.ServiceName
}
