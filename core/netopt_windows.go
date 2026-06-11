//go:build windows

// netopt_windows.go — Windows 网络优化：扩展动态出站端口范围
// 默认 49152-65535 (16384个)，扩展为 1025-65535 (64511个)
// 需要管理员权限

package core

import (
	"os/exec"
	"syscall"
)

// ExpandDynamicPortRange 扩展 Windows 动态出站端口范围
// 静默执行，不弹出 cmd 窗口，失败时不报错
func ExpandDynamicPortRange() {
	commands := [][]string{
		{"int", "ipv4", "set", "dynamicport", "tcp", "start=1025", "num=64511"},
		{"int", "ipv4", "set", "dynamicport", "udp", "start=1025", "num=64511"},
		{"int", "ipv6", "set", "dynamicport", "tcp", "start=1025", "num=64511"},
		{"int", "ipv6", "set", "dynamicport", "udp", "start=1025", "num=64511"},
	}
	for _, args := range commands {
		cmd := exec.Command("netsh", args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
	}
}
