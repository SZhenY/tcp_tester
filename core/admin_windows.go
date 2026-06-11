//go:build windows

// admin_windows.go — Windows 管理员权限检测
// 通过 Shell32.dll 的 IsUserAnAdmin() API 检测，无弹窗

package core

import "syscall"

// Shell32 API
var (
	modShell32        = syscall.NewLazyDLL("shell32.dll")
	procIsUserAnAdmin = modShell32.NewProc("IsUserAnAdmin")
)

// IsAdmin 检测当前进程是否以管理员权限运行
func IsAdmin() bool {
	ret, _, _ := procIsUserAnAdmin.Call()
	return ret != 0
}

// EnsureAdmin Windows 版本由 manifest 保证管理员权限
// 此函数仅用于兼容接口，实际不做任何操作
func EnsureAdmin(emitter EventEmitter) {
	// manifest requireAdministrator 已保证管理员权限
	// 拒绝 UAC 时进程不会启动，不会执行到这里
}
