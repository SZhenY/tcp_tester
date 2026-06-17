//go:build !cli && !fyne

// app.go — Wails GUI 版本的 App 包装器
// 将 core.App 桥接到 Wails 运行时，暴露给前端的方法签名不变

package main

import (
	"context"
	"strconv"

	"tcp-tester/core"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsEmitter 实现 core.EventEmitter，桥接到 Wails 运行时
type WailsEmitter struct {
	ctx context.Context
}

// EmitLog 发送日志到前端
func (e *WailsEmitter) EmitLog(msg string) {
	wailsRuntime.EventsEmit(e.ctx, "log", msg)
}

// EmitStats 发送统计数据到前端
func (e *WailsEmitter) EmitStats(stats core.Stats) {
	wailsRuntime.EventsEmit(e.ctx, "stats", stats)
}

// EmitTestFinished 发送测试完成事件到前端
func (e *WailsEmitter) EmitTestFinished(reason string) {
	wailsRuntime.EventsEmit(e.ctx, "testFinished", reason)
}

// App Wails GUI 版本的应用结构体
type App struct {
	core    *core.App
	emitter *WailsEmitter
}

// NewApp 创建新的 GUI 应用实例
func NewApp() *App {
	emitter := &WailsEmitter{}
	return &App{
		core:    core.New(emitter),
		emitter: emitter,
	}
}

// startup Wails 启动生命周期回调
func (a *App) startup(ctx context.Context) {
	a.emitter.ctx = ctx
	// 扩展动态端口范围（仅 Windows，静默执行）
	core.ExpandDynamicPortRange()
}

// shutdown Wails 关闭生命周期回调
func (a *App) shutdown(ctx context.Context) {
	a.core.Cleanup()
}

// ResolveDomain 解析域名（暴露给前端）
func (a *App) ResolveDomain(domain string) (string, error) {
	return a.core.ResolveDomain(domain)
}

// StartTest 开始测试（暴露给前端）
func (a *App) StartTest(target string, threadCount int, intervalMs int, failureLimit int64, successLimit int64) error {
	return a.core.StartTest(target, threadCount, intervalMs, failureLimit, successLimit)
}

// StopTest 停止测试（暴露给前端）
func (a *App) StopTest() {
	a.core.StopTest()
}

// DefaultConfig 前端默认配置（从后端常量同步）
type DefaultConfig struct {
	Port         string `json:"port"`
	ThreadCount  string `json:"threadCount"`
	IntervalMs   string `json:"intervalMs"`
	FailureLimit string `json:"failureLimit"`
	SuccessLimit string `json:"successLimit"`
}

// GetDefaultConfig 返回后端配置的默认值（暴露给前端）
func (a *App) GetDefaultConfig() DefaultConfig {
	return DefaultConfig{
		Port:         strconv.Itoa(core.PortConfig.DefaultVal),
		ThreadCount:  strconv.Itoa(core.ThreadConfig.DefaultVal),
		IntervalMs:   strconv.Itoa(core.IntervalConfig.DefaultVal),
		FailureLimit: strconv.Itoa(core.FailureConfig.DefaultVal),
		SuccessLimit: strconv.Itoa(core.SuccessConfig.DefaultVal),
	}
}
