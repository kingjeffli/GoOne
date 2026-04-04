package application

// mainloop boot,  如有问题飞书联系 to: Iori
import (
	"fmt"
	"os"
	"time"

	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/api/logger"
)

type AppInterface interface {
	OnInit() error
	OnReload() error
	OnProc() bool // return: isIdle
	OnTick(lastMs, nowMs int64)
	OnExit()
}

type Application struct {
	appHandler AppInterface

	tickInterval int64
	lastTickTime int64
}

var sig = make(chan os.Signal, 1)
var app Application

const (
	defaultTickIntervalMs = 10
	idleProcInterval      = 5 * time.Millisecond
	busyProcInterval      = 1 * time.Millisecond
)

func Init(handler AppInterface) *Application {
	app.appHandler = handler
	err := app.appHandler.OnInit()
	if err != nil {
		logger.Fatalf("Initialized fail | %v", err)
		os.Exit(1)
		return nil
	}

	app.tickInterval = defaultTickIntervalMs

	SignalNotify()
	return &app
}

// 每秒执行多少帧
func (a *Application) SetTickInterval(interval int64) {
	if interval > 0 && interval < 1000 {
		a.tickInterval = interval
	}
}

func (a *Application) exit() {
	a.appHandler.OnExit()
}

func (a *Application) reload() error {
	return a.appHandler.OnReload()
}

func (a *Application) loopOnce() bool {
	return a.appHandler.OnProc()
}

func (a *Application) tick(lastMs, nowMs int64) {
	a.appHandler.OnTick(lastMs, nowMs)
}

func Run() {
	fmt.Println("-----------  SvrImpl  is  Runing ------------ ")
	if app.appHandler == nil {
		return
	}

	datetime.Tick()
	nowMs := datetime.NowMs()
	app.tick(app.lastTickTime, nowMs)
	app.lastTickTime = nowMs

	tickTicker := time.NewTicker(app.tickDuration())
	defer tickTicker.Stop()

	procTimer := time.NewTimer(0)
	defer stopTimer(procTimer)

	for {
		select {
		case s := <-sig:
			if isReloadSignal(s) {
				if err := app.reload(); err != nil {
					logger.Errorf("reload failed | %v", err)
				}
				continue
			}
			app.exit()
			return
		case <-tickTicker.C:
			datetime.Tick()
			nowMs := datetime.NowMs()
			app.tick(app.lastTickTime, nowMs)
			app.lastTickTime = nowMs
		case <-procTimer.C:
			datetime.Tick()
			isIdle := app.loopOnce()
			resetTimer(procTimer, nextProcInterval(isIdle))
		}
	}
}

func (a *Application) tickDuration() time.Duration {
	interval := a.tickInterval
	if interval <= 0 {
		interval = defaultTickIntervalMs
	}
	return time.Duration(interval) * time.Millisecond
}

func nextProcInterval(isIdle bool) time.Duration {
	if isIdle {
		return idleProcInterval
	}
	return busyProcInterval
}

func resetTimer(t *time.Timer, d time.Duration) {
	if t == nil {
		return
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func stopTimer(t *time.Timer) {
	if t == nil {
		return
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
