//go:build !windows

// netopt_other.go — 非 Windows 平台无需扩展端口范围

package core

// ExpandDynamicPortRange 非 Windows 平台空实现
func ExpandDynamicPortRange() {}
