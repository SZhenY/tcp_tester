//go:build fyne

package main

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"tcp-tester/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type ipOption struct {
	label string
	value string
	typ   string
}

type FyneApp struct {
	fyneApp fyne.App
	window  fyne.Window
	core    *core.App

	domainEntry   *widget.Entry
	portEntry     *widget.Entry
	threadEntry   *widget.Entry
	intervalEntry *widget.Entry
	failureEntry  *widget.Entry
	successEntry  *widget.Entry

	ipSelect   *widget.Select
	ipOptions  []ipOption
	selectedIP string
	resolveBtn *widget.Button

	startBtn *widget.Button
	stopBtn  *widget.Button

	ipStatusLabel *widget.Label
	ipv4Label     *widget.Label
	ipv6Label     *widget.Label
	hideIPCheck   *widget.Check

	targetTag *widget.Label
	threadTag *widget.Label

	successVal *bindingString
	failureVal *bindingString
	cpsVal     *bindingString

	logCh      chan string
	logEntries []string
	logMu      sync.Mutex
	logList    *widget.List

	publicIPv4 string
	publicIPv6 string
	isRunning  bool
	statsMu    sync.Mutex
	lastStats  core.Stats
}

type bindingString struct {
	val string
	mu  sync.RWMutex
}

func newBindingString(init string) *bindingString {
	return &bindingString{val: init}
}

func (b *bindingString) Get() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.val
}

func (b *bindingString) Set(v string) {
	b.mu.Lock()
	b.val = v
	b.mu.Unlock()
}

func newFyneApp() *FyneApp {
	return &FyneApp{
		logCh:      make(chan string, 512),
		successVal: newBindingString("0"),
		failureVal: newBindingString("0"),
		cpsVal:     newBindingString("0"),
	}
}

func (a *FyneApp) run() {
	a.fyneApp = app.New()
	a.fyneApp.Settings().SetTheme(&md3DarkTheme{})
	a.window = a.fyneApp.NewWindow("TCP 并发连接测试工具")

	if runtime.GOOS == "android" {
		a.window.SetFullScreen(true)
	}

	a.core = core.New(&FyneEmitter{app: a})

	defaults := a.coreDefaults()
	a.buildUI(defaults)

	a.window.SetOnClosed(func() {
		a.core.Cleanup()
	})

	go a.fetchPublicIP()
	a.window.ShowAndRun()
}

type appDefaults struct {
	port     string
	thread   string
	interval string
	failure  string
	success  string
}

func (a *FyneApp) coreDefaults() appDefaults {
	d := appDefaults{port: "80", thread: "64", interval: "0", failure: "50", success: "1000"}
	if runtime.GOOS == "android" {
		d.thread = "32"
	}
	return d
}

func (a *FyneApp) buildUI(d appDefaults) {
	a.domainEntry = widget.NewEntry()
	a.domainEntry.SetPlaceHolder("例如 www.example.com")

	a.portEntry = widget.NewEntry()
	a.portEntry.SetText(d.port)

	a.threadEntry = widget.NewEntry()
	a.threadEntry.SetText(d.thread)

	a.intervalEntry = widget.NewEntry()
	a.intervalEntry.SetText(d.interval)

	a.failureEntry = widget.NewEntry()
	a.failureEntry.SetText(d.failure)

	a.successEntry = widget.NewEntry()
	a.successEntry.SetText(d.success)

	a.ipSelect = widget.NewSelect([]string{}, func(val string) {
		a.selectedIP = val
	})
	a.ipSelect.PlaceHolder = "解析后选择 IP"

	a.resolveBtn = widget.NewButton("解析域名", a.onResolve)

	a.startBtn = widget.NewButton("开始测试", a.onStart)
	a.startBtn.Importance = widget.HighImportance

	a.stopBtn = widget.NewButton("停止测试", a.onStop)
	a.stopBtn.Importance = widget.DangerImportance
	a.stopBtn.Disable()

	a.ipStatusLabel = widget.NewLabel("无 IP")
	a.ipv4Label = widget.NewLabel("IPv4: 未检测到")
	a.ipv6Label = widget.NewLabel("IPv6: 未检测到")
	a.hideIPCheck = widget.NewCheck("隐藏 IP", func(checked bool) {
		a.updateIPDisplay()
	})
	a.hideIPCheck.SetChecked(true)

	a.targetTag = widget.NewLabel("目标: 未设置")
	a.threadTag = widget.NewLabel("并发: " + d.thread)

	successStat := a.makeStatCard("成功", a.successVal)
	failureStat := a.makeStatCard("失败", a.failureVal)
	cpsStat := a.makeStatCard("CPS", a.cpsVal)

	a.logList = widget.NewList(
		func() int {
			a.logMu.Lock()
			defer a.logMu.Unlock()
			return len(a.logEntries)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			a.logMu.Lock()
			defer a.logMu.Unlock()
			if id < len(a.logEntries) {
				label := obj.(*widget.Label)
				label.SetText(a.logEntries[id])
				label.TextStyle = fyne.TextStyle{Monospace: true}
			}
		},
	)

	go a.consumeLogs()

	settingsTab := a.buildSettingsTab(d)
	testTab := container.NewVBox(
		a.buildIPStatusBar(),
		widget.NewSeparator(),
		a.buildActionBar(),
		widget.NewSeparator(),
		container.NewGridWithColumns(3, successStat, failureStat, cpsStat),
		widget.NewSeparator(),
		widget.NewLabel("  最近日志"),
		a.buildLogPreview(),
	)
	logsTab := container.NewBorder(
		widget.NewLabel("  日志输出"), nil, nil, nil,
		a.logList,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("设置", settingsTab),
		container.NewTabItem("测试", testTab),
		container.NewTabItem("日志", logsTab),
	)
	tabs.SetTabLocation(container.TabLocationBottom)

	a.window.SetContent(tabs)
}

func (a *FyneApp) consumeLogs() {
	for msg := range a.logCh {
		a.logMu.Lock()
		a.logEntries = append(a.logEntries, msg)
		if len(a.logEntries) > 2000 {
			a.logEntries = a.logEntries[len(a.logEntries)-2000:]
		}
		a.logMu.Unlock()

		a.logList.Refresh()

		a.logMu.Lock()
		count := len(a.logEntries)
		a.logMu.Unlock()
		if count > 0 {
			a.logList.ScrollToBottom()
		}
	}
}

func (a *FyneApp) buildSettingsTab(d appDefaults) fyne.CanvasObject {
	return container.NewVScroll(
		container.NewVBox(
			widget.NewLabel("  目标设置"),
			widget.NewLabel("域名"),
			a.domainEntry,
			container.NewGridWithColumns(2,
				container.NewVBox(widget.NewLabel("端口"), a.portEntry),
				container.NewVBox(widget.NewLabel(" "), a.resolveBtn),
			),
			widget.NewLabel("目标 IP"),
			a.ipSelect,

			widget.NewSeparator(),
			widget.NewLabel("  参数设置"),
			container.NewGridWithColumns(2,
				container.NewVBox(widget.NewLabel("并发线程数"), a.threadEntry),
				container.NewVBox(widget.NewLabel("发送间隔 (ms)"), a.intervalEntry),
			),
			container.NewGridWithColumns(2,
				container.NewVBox(widget.NewLabel("失败上限 (停止)"), a.failureEntry),
				container.NewVBox(widget.NewLabel("成功上限 (停止)"), a.successEntry),
			),
			widget.NewSeparator(),
		),
	)
}

func (a *FyneApp) buildIPStatusBar() fyne.CanvasObject {
	return container.NewHBox(
		a.ipStatusLabel,
		widget.NewSeparator(),
		a.ipv4Label,
		a.ipv6Label,
		a.hideIPCheck,
	)
}

func (a *FyneApp) buildActionBar() fyne.CanvasObject {
	return container.NewVBox(
		container.NewHBox(a.startBtn, a.stopBtn),
		container.NewHBox(a.targetTag, a.threadTag),
	)
}

func (a *FyneApp) buildLogPreview() fyne.CanvasObject {
	scroll := container.NewVScroll(widget.NewLabel(""))
	scroll.SetMinSize(fyne.NewSize(0, 200))
	return scroll
}

func (a *FyneApp) makeStatCard(label string, val *bindingString) fyne.CanvasObject {
	title := widget.NewLabel(label)
	title.Alignment = fyne.TextAlignCenter

	valueLabel := widget.NewLabel("0")
	valueLabel.Alignment = fyne.TextAlignCenter
	valueLabel.TextStyle = fyne.TextStyle{Bold: true}
	valueLabel.Importance = widget.HighImportance

	go func() {
		for {
			v := val.Get()
			valueLabel.SetText(v)
			canvas.Refresh(valueLabel)
			time.Sleep(500 * time.Millisecond)
		}
	}()

	return container.NewVBox(title, valueLabel)
}

func (a *FyneApp) onResolve() {
	domain := a.domainEntry.Text
	if domain == "" {
		return
	}
	a.resolveBtn.Disable()
	a.resolveBtn.SetText("解析中...")

	go func() {
		result, err := a.core.ResolveDomain(domain)
		if err != nil {
			dialog.ShowError(err, a.window)
			a.resolveBtn.Enable()
			a.resolveBtn.SetText("解析域名")
			return
		}

		options := parseIPResult(result)
		a.ipOptions = options

		labels := make([]string, len(options))
		for i, o := range options {
			labels[i] = o.label
		}
		a.ipSelect.Options = labels
		a.ipSelect.ClearSelected()

		if len(options) > 0 {
			first := options[0]
			for _, o := range options {
				if o.typ == "ipv4" {
					first = o
					break
				}
			}
			a.selectedIP = first.value
			a.ipSelect.SetSelected(first.label)
		}

		a.resolveBtn.Enable()
		a.resolveBtn.SetText("解析域名")

		ts := time.Now().Format("15:04:05")
		select {
		case a.logCh <- fmt.Sprintf("[%s] 域名解析完成: %s -> %d 个地址", ts, domain, len(options)):
		default:
		}
	}()
}

func (a *FyneApp) onStart() {
	ip := a.selectedIP
	if ip == "" {
		dialog.ShowError(fmt.Errorf("请先解析域名并选择 IP"), a.window)
		return
	}

	port := a.portEntry.Text
	target := ip + ":" + port
	if strings.Contains(ip, ":") {
		target = "[" + ip + "]:" + port
	}

	threads, _ := strconv.Atoi(a.threadEntry.Text)
	interval, _ := strconv.Atoi(a.intervalEntry.Text)
	failLimit, _ := strconv.ParseInt(a.failureEntry.Text, 10, 64)
	succLimit, _ := strconv.ParseInt(a.successEntry.Text, 10, 64)

	if err := a.core.StartTest(target, threads, interval, failLimit, succLimit); err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	a.isRunning = true
	a.logMu.Lock()
	a.logEntries = nil
	a.logMu.Unlock()
	a.logList.Refresh()

	a.successVal.Set("0")
	a.failureVal.Set("0")
	a.cpsVal.Set("0")

	a.startBtn.Disable()
	a.stopBtn.Enable()

	a.targetTag.SetText(fmt.Sprintf("目标: %s:%s", ip, port))
	a.threadTag.SetText(fmt.Sprintf("并发: %d", threads))

	canvas.Refresh(a.startBtn)
	canvas.Refresh(a.stopBtn)
}

func (a *FyneApp) onStop() {
	a.core.StopTest()
}

func (a *FyneApp) updateIPDisplay() {
	hide := a.hideIPCheck.Checked
	if a.publicIPv4 != "" {
		if hide {
			a.ipv4Label.SetText("IPv4: " + maskIP(a.publicIPv4))
		} else {
			a.ipv4Label.SetText("IPv4: " + a.publicIPv4)
		}
	} else {
		a.ipv4Label.SetText("IPv4: 未检测到")
	}
	if a.publicIPv6 != "" {
		if hide {
			a.ipv6Label.SetText("IPv6: " + maskIP(a.publicIPv6))
		} else {
			a.ipv6Label.SetText("IPv6: " + a.publicIPv6)
		}
	} else {
		a.ipv6Label.SetText("IPv6: 未检测到")
	}
}

func (a *FyneApp) fetchPublicIP() {
	rounds := []struct{ ipv4, ipv6 string }{
		{"https://api-ipv4.ip.sb/ip", "https://api-ipv6.ip.sb/ip"},
		{"https://ipinfo.io/ip", "https://api64.ipify.org"},
	}
	client := &http.Client{Timeout: 5 * time.Second}

	for round := 0; round < 10; round++ {
		apis := rounds[round%len(rounds)]
		var wg sync.WaitGroup
		var ipv4, ipv6 string
		wg.Add(2)

		go func() {
			defer wg.Done()
			if ip, err := httpGet(client, apis.ipv4); err == nil {
				ipv4 = ip
			}
		}()
		go func() {
			defer wg.Done()
			if ip, err := httpGet(client, apis.ipv6); err == nil {
				ipv6 = ip
			}
		}()
		wg.Wait()

		if ipv4 != "" {
			a.publicIPv4 = ipv4
		}
		if ipv6 != "" {
			a.publicIPv6 = ipv6
		}
		if a.publicIPv4 != "" && a.publicIPv6 != "" {
			break
		}
		if round < 9 {
			time.Sleep(3 * time.Second)
		}
	}

	a.updateIPStatus()
	a.updateIPDisplay()
}

func (a *FyneApp) updateIPStatus() {
	has4 := a.publicIPv4 != ""
	has6 := a.publicIPv6 != ""
	switch {
	case has4 && has6:
		a.ipStatusLabel.SetText("就绪")
	case has4 && !has6:
		a.ipStatusLabel.SetText("IPv6 缺失")
	case !has4 && has6:
		a.ipStatusLabel.SetText("IPv4 缺失")
	default:
		a.ipStatusLabel.SetText("无 IP，请检查网络")
	}
}

func parseIPResult(result string) []ipOption {
	lines := strings.Split(result, "\n")
	var options []ipOption
	currentType := "ipv4"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "IPv4:" {
			currentType = "ipv4"
		} else if trimmed == "IPv6:" {
			currentType = "ipv6"
		} else if trimmed != "" {
			label := fmt.Sprintf("[%s] %s", strings.ToUpper(currentType), trimmed)
			options = append(options, ipOption{label: label, value: trimmed, typ: currentType})
		}
	}
	return options
}

func httpGet(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func maskIP(ip string) string {
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) > 4 {
			return strings.Join(parts[:2], ":") + ":****:****:" + strings.Join(parts[len(parts)-2:], ":")
		}
		return "****:" + strings.Join(parts[len(parts)-2:], ":")
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + ".***.***." + parts[3]
	}
	return "***.***.***"
}
