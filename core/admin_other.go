//go:build !windows

// admin_other.go — Linux/macOS 权限检测
// 检测 root 权限，仅提示不阻止

package core

import "os"

// IsAdmin 检测当前进程是否以 root 权限运行
func IsAdmin() bool {
	return os.Getuid() == 0
}

// EnsureAdmin 检测 root 权限，仅提示用户，不阻止运行
func EnsureAdmin(emitter EventEmitter) {
	if IsAdmin() {
		return
	}
	emitter.EmitLog("提示: 当前非 root 权限，建议使用 sudo 运行以获得最佳性能")
}
