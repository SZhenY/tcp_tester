//go:build fyne

package main

import (
	"fmt"
	"time"

	"tcp-tester/core"

	"fyne.io/fyne/v2"
)

type FyneEmitter struct {
	app *FyneApp
}

func (e *FyneEmitter) EmitLog(msg string) {
	ts := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", ts, msg)
	select {
	case e.app.logCh <- formatted:
	default:
	}
}

func (e *FyneEmitter) EmitStats(stats core.Stats) {
	e.app.statsMu.Lock()
	e.app.lastStats = stats
	e.app.statsMu.Unlock()

	e.app.successVal.Set(formatNum(stats.Success))
	e.app.failureVal.Set(formatNum(stats.Failure))
	e.app.cpsVal.Set(formatNum(int64(stats.Rate)))
}

func (e *FyneEmitter) EmitTestFinished(reason string) {
	e.app.isRunning = false
	e.app.startBtn.Enable()
	e.app.stopBtn.Disable()

	ts := time.Now().Format("15:04:05")
	select {
	case e.app.logCh <- fmt.Sprintf("[%s] 测试结束: %s", ts, reason):
	default:
	}
	fyne.CurrentApp().Driver().CanvasForObject(e.app.startBtn).Refresh(e.app.startBtn)
	fyne.CurrentApp().Driver().CanvasForObject(e.app.stopBtn).Refresh(e.app.stopBtn)
}

func formatNum(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%d,%03d", n/1000, n%1000)
}
