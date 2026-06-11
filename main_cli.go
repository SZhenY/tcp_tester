//go:build cli

// main_cli.go — CLI 版本入口
// 支持 Windows (PowerShell/CMD) 和 Linux 终端

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"tcp-tester/core"
)

// CLIEmitter 实现 core.EventEmitter，输出到终端
type CLIEmitter struct{}

// EmitLog 输出日志到标准输出
func (e *CLIEmitter) EmitLog(msg string) {
	fmt.Println(msg)
}

// EmitStats 输出统计信息到标准输出（每秒一行）
func (e *CLIEmitter) EmitStats(stats core.Stats) {
	fmt.Printf("[%s] 统计 - 成功: %d | 失败: %d | 总计: %d | 新增: %d | CPS: %.0f | 平均: %.0f\n",
		time.Now().Format("15:04:05"),
		stats.Success,
		stats.Failure,
		stats.Total,
		stats.Total-stats.Success-stats.Failure,
		stats.Rate,
		stats.AvgCPS,
	)
}

// EmitTestFinished 输出测试完成信息
func (e *CLIEmitter) EmitTestFinished(reason string) {
	fmt.Printf("\n测试结束: %s\n", reason)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println(`tcp-tester - TCP 并发连接测试工具 (CLI)

用法:
  tcp-tester [选项]

选项:
  -u, --url <域名>         目标域名 (默认: www.huawei.com)
  -p, --port <端口>        目标端口 (默认: 80)
  -4, --ipv4               使用 IPv4 (默认)
  -6, --ipv6               使用 IPv6
  -t, --threads <数量>     并发线程数 (默认: 64)
  -i, --interval <ms>      发送间隔，毫秒 (默认: 0)
  -f, --fail-limit <数量>  失败上限 (默认: 50)
  -s, --succ-limit <数量>  成功上限 (默认: 1000)
  -h, --help               显示此帮助信息

示例:
  tcp-tester                                     # 默认参数测试 www.huawei.com:80 IPv4
  tcp-tester -u example.com -p 443               # 自定义域名和端口
  tcp-tester -u example.com -6                   # 测试 IPv6
  tcp-tester -u example.com -t 128 -i 10         # 128线程, 10ms间隔
  tcp-tester -u example.com -f 100 -s 5000       # 失败上限100, 成功上限5000`)
}

// httpGet 发送 GET 请求并返回响应体字符串（带 5 秒超时）
func httpGet(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
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

// fetchPublicIP 检测本机公网 IPv4 和 IPv6 地址并输出日志
// 循环重试，3 秒间隔，ip.sb ↔ ipinfo/ipify 交替，直到获取到至少一个 IP
func fetchPublicIP(emitter *CLIEmitter) (string, string) {
	emitter.EmitLog("正在检测本机公网 IP ...")

	type apiPair struct {
		ipv4 string
		ipv6 string
	}
	rounds := []apiPair{
		{"https://api-ipv4.ip.sb/ip", "https://api-ipv6.ip.sb/ip"},
		{"https://ipinfo.io/ip", "https://api64.ipify.org"},
	}

	var resultIPv4, resultIPv6 string
	round := 0
	for {
		apis := rounds[round%len(rounds)]

		var ipv4, ipv6 string
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			if ip, err := httpGet(apis.ipv4); err == nil {
				ipv4 = ip
			}
		}()
		go func() {
			defer wg.Done()
			if ip, err := httpGet(apis.ipv6); err == nil {
				ipv6 = ip
			}
		}()
		wg.Wait()

		if ipv4 != "" {
			resultIPv4 = ipv4
		}
		if ipv6 != "" {
			resultIPv6 = ipv6
		}

		// 两个都获取到了，跳出
		if resultIPv4 != "" && resultIPv6 != "" {
			break
		}

		// 至少一个还没获取到，等待 3 秒后重试
		round++
		time.Sleep(3 * time.Second)
	}

	// 输出结果
	if resultIPv4 != "" {
		emitter.EmitLog(fmt.Sprintf("本机 IPv4: %s", resultIPv4))
	} else {
		emitter.EmitLog("本机 IPv4: (未检测到)")
	}
	if resultIPv6 != "" {
		emitter.EmitLog(fmt.Sprintf("本机 IPv6: %s", resultIPv6))
	} else {
		emitter.EmitLog("本机 IPv6: (未检测到)")
	}

	// 状态判断
	has4 := resultIPv4 != ""
	has6 := resultIPv6 != ""
	switch {
	case has4 && has6:
		emitter.EmitLog("状态: 就绪")
	case has4 && !has6:
		emitter.EmitLog("状态: IPv6 缺失")
	case !has4 && has6:
		emitter.EmitLog("状态: IPv4 缺失")
	}

	return resultIPv4, resultIPv6
}

// resolveAndSelect 解析域名并选择指定协议的首个 IP 地址
func resolveAndSelect(domain string, useIPv6 bool) (string, error) {
	engine := core.New(&CLIEmitter{})
	result, err := engine.ResolveDomain(domain)
	if err != nil {
		return "", err
	}

	lines := strings.Split(result, "\n")
	var ipv4s, ipv6s []string
	currentType := "ipv4"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "IPv4:":
			currentType = "ipv4"
		case "IPv6:":
			currentType = "ipv6"
		case "":
			continue
		default:
			if currentType == "ipv4" {
				ipv4s = append(ipv4s, trimmed)
			} else {
				ipv6s = append(ipv6s, trimmed)
			}
		}
	}

	if useIPv6 {
		if len(ipv6s) == 0 {
			return "", fmt.Errorf("未找到 IPv6 地址")
		}
		return ipv6s[0], nil
	}
	if len(ipv4s) == 0 {
		return "", fmt.Errorf("未找到 IPv4 地址")
	}
	return ipv4s[0], nil
}

func main() {
	// 定义命令行参数
	var (
		url       string
		port      int
		useIPv4   bool
		useIPv6   bool
		threads   int
		interval  int
		failLimit int64
		succLimit int64
		showHelp  bool
	)

	// 计算默认线程数：Linux 使用 CPU 核心数，Windows 使用后端默认值
	defaultThreads := core.ThreadConfig.DefaultVal
	if runtime.GOOS == "linux" {
		cpuCount := runtime.NumCPU()
		if cpuCount > defaultThreads {
			defaultThreads = cpuCount
		}
	}

	// 注册参数（支持短写和长写）
	flag.StringVar(&url, "u", "www.huawei.com", "目标域名")
	flag.StringVar(&url, "url", "www.huawei.com", "目标域名")
	flag.IntVar(&port, "p", core.PortConfig.DefaultVal, "目标端口")
	flag.IntVar(&port, "port", core.PortConfig.DefaultVal, "目标端口")
	flag.BoolVar(&useIPv4, "4", true, "使用 IPv4 (默认)")
	flag.BoolVar(&useIPv4, "ipv4", true, "使用 IPv4 (默认)")
	flag.BoolVar(&useIPv6, "6", false, "使用 IPv6")
	flag.BoolVar(&useIPv6, "ipv6", false, "使用 IPv6")
	flag.IntVar(&threads, "t", defaultThreads, "并发线程数")
	flag.IntVar(&threads, "threads", defaultThreads, "并发线程数")
	flag.IntVar(&interval, "i", core.IntervalConfig.DefaultVal, "发送间隔 (ms)")
	flag.IntVar(&interval, "interval", core.IntervalConfig.DefaultVal, "发送间隔 (ms)")
	flag.Int64Var(&failLimit, "f", int64(core.FailureConfig.DefaultVal), "失败上限")
	flag.Int64Var(&failLimit, "fail-limit", int64(core.FailureConfig.DefaultVal), "失败上限")
	flag.Int64Var(&succLimit, "s", int64(core.SuccessConfig.DefaultVal), "成功上限")
	flag.Int64Var(&succLimit, "succ-limit", int64(core.SuccessConfig.DefaultVal), "成功上限")
	flag.BoolVar(&showHelp, "h", false, "显示帮助")
	flag.BoolVar(&showHelp, "help", false, "显示帮助")

	flag.Parse()

	// 显示帮助
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// -6 覆盖 -4
	if useIPv6 {
		useIPv4 = false
	}

	// 扩展动态端口范围（仅 Windows，需要管理员权限）
	// Windows: manifest requireAdministrator 保证管理员权限
	// Linux: 无操作
	core.ExpandDynamicPortRange()

	// 检测 root 权限（仅 Linux，仅提示）
	emitter := &CLIEmitter{}
	core.EnsureAdmin(emitter)

	// 解析域名
	emitter.EmitLog(fmt.Sprintf("正在解析域名: %s ...", url))

	ip, err := resolveAndSelect(url, useIPv6)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	proto := "IPv4"
	if useIPv6 {
		proto = "IPv6"
	}
	emitter.EmitLog(fmt.Sprintf("解析成功: %s -> [%s] %s", url, proto, ip))

	// 检测本机公网 IP
	fetchPublicIP(emitter)

	// 构建目标地址
	target := net.JoinHostPort(ip, fmt.Sprintf("%d", port))

	// 创建测试引擎
	app := core.New(emitter)

	// 注册信号处理（Ctrl+C 优雅停止）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\n收到停止信号，正在停止...\n")
		app.StopTest()
	}()

	// 启动测试
	emitter.EmitLog(fmt.Sprintf("开始测试 -> %s (并发:%d 间隔:%dms)", target, threads, interval))
	if err := app.StartTest(target, threads, interval, failLimit, succLimit); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 等待测试完成并清理
	app.Wait()
	app.Cleanup()

	// 当终端直接运行时（非管道/重定向），等待用户按回车再退出
	// 防止双击运行时窗口立即关闭
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Print("\n按回车键退出...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
}
