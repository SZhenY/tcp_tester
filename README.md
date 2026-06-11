# TCP Tester

基于 Go + Wails 2 + Vue 3 的 TCP 并发连接测试工具，用于测试家庭宽带连接会话数上限。

## 特性

- **高并发 TCP 连接测试** — Worker Pool 模式，分片连接池减少锁竞争
- **实时统计** — 成功/失败/CPS，500ms 平滑刷新
- **IPv4/IPv6 双栈** — 自动解析域名，支持 IPv4/IPv6 分别选择
- **多 DNS 解析** — 223.5.5.5 → 119.29.29.29 → 系统 DNS 自动回退
- **动态端口扩展** — 自动扩展 Windows 动态出站端口范围（需管理员权限）
- **双版本架构** — Windows GUI（Wails 2）+ CLI（Windows/Linux/macOS）
- **虚拟滚动日志** — 环形缓冲区 + 虚拟列表

## 快速开始

### Windows GUI

双击 `tcp-tester.exe`，自动请求管理员权限（用于端口扩展）。

### CLI 版本

```bash
# Windows CLI
tcp-tester-cli.exe -u example.com -p 443 -t 128

# Linux CLI
./tcp-tester-linux-cli -u example.com -p 443 -t 128

# 显示帮助
tcp-tester-cli.exe --help
```

### CLI 参数

| 参数 | 短写 | 默认值 | 说明 |
|------|------|--------|------|
| `--url` | `-u` | `www.huawei.com` | 目标域名 |
| `--port` | `-p` | `80` | 目标端口 |
| `--ipv4` | `-4` | `true` | 使用 IPv4 |
| `--ipv6` | `-6` | `false` | 使用 IPv6 |
| `--threads` | `-t` | `64` | 并发线程数 |
| `--interval` | `-i` | `0` | 发送间隔 (ms) |
| `--fail-limit` | `-f` | `50` | 失败上限 |
| `--succ-limit` | `-s` | `1000` | 成功上限 |

## 开发

### 环境要求

- Go 1.23+
- Node.js 18+
- Wails CLI v2.12+

### 开发模式

```bash
# 安装前端依赖
cd frontend && npm install

# 启动开发服务器（热重载）
wails dev
```

### 构建

```bash
# 构建 Windows GUI
wails build

# 构建所有版本（Windows GUI + CLI + Linux CLI）
.\build.ps1 all

# 仅构建 CLI 版本
.\build.ps1 cli
```

## 架构

```
tcp-tester/
├── core/                   ← 纯 Go 核心引擎（零 GUI 依赖）
│   ├── core.go             ← 测试引擎 + EventEmitter 接口
│   ├── config.go           ← 参数校验配置
│   ├── pool.go             ← 分片连接池（Worker Pool）
│   ├── dns.go              ← 多 DNS 解析
│   ├── admin_windows.go    ← Windows 管理员权限检测
│   ├── admin_other.go      ← Linux root 检测
│   ├── netopt_windows.go   ← Windows 端口范围扩展
│   └── netopt_other.go     ← 非 Windows 空实现
├── app.go                  ← Wails GUI 包装器（EventEmitter → Wails Events）
├── main.go                 ← GUI 入口（//go:build !cli）
├── main_cli.go             ← CLI 入口（//go:build cli）
├── frontend/               ← Vue 3 前端
│   └── src/
│       ├── components/     ← LogViewer（虚拟滚动）
│       ├── stores/         ← Pinia store + CircularBuffer
│       └── assets/         ← MD3 Motion CSS + 字体 + 图标
├── ico/                    ← 应用图标（SVG/PNG/ICO）
└── build.ps1               ← 交叉编译脚本
```

### 核心设计

- **EventEmitter 接口** — 解耦 GUI/CLI 依赖，核心引擎不引用任何 UI 框架
- **`//go:build` 标签** — GUI 和 CLI 共享同一 `core` 包，通过 build tags 分离入口
- **Worker Pool** — 预创建持久化 goroutine，channel 分发任务，零创建开销
- **分片连接池** — `runtime.NumCPU()` 分片，原子索引分配，减少锁竞争
- **环形缓冲区** — 固定 2000 容量，O(1) 写入，虚拟滚动只渲染可见行

## 许可证

[Apache License 2.0](LICENSE)
