// config.go — 配置常量、校验函数

package core

import (
	"fmt"
	"time"
)

const (
	MaxConnPoolSize = 1000000 // 最大连接池容量
	MaxThreadCount  = 10000   // 最大并发线程数
)

// 各参数的校验配置
var (
	PortConfig     = InputConfig{MinValue: 1, MaxValue: 65535, DefaultVal: 80, Name: "端口"}
	ThreadConfig   = InputConfig{MinValue: 1, MaxValue: MaxThreadCount, DefaultVal: 64, Name: "并发数"}
	IntervalConfig = InputConfig{MinValue: 0, MaxValue: 60000, DefaultVal: 0, Name: "间隔"}
	FailureConfig  = InputConfig{MinValue: 1, MaxValue: 1000000, DefaultVal: 50, Name: "失败停止"}
	SuccessConfig  = InputConfig{MinValue: 1, MaxValue: MaxConnPoolSize, DefaultVal: 1000, Name: "成功上限"}
)

// timestamp 返回当前时间戳字符串（毫秒精度）
func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

// validateInput 校验单个输入值，返回校验后的值和错误信息
func validateInput(value int, cfg InputConfig) (int, string) {
	if value < cfg.MinValue {
		return cfg.MinValue, fmt.Sprintf("%s不能小于%d", cfg.Name, cfg.MinValue)
	}
	if value > cfg.MaxValue {
		return cfg.MaxValue, fmt.Sprintf("%s不能大于%d", cfg.Name, cfg.MaxValue)
	}
	return value, ""
}

// ValidateInputs 批量校验所有输入参数，返回第一个错误信息（空字符串表示通过）
func ValidateInputs(port, threadCount, intervalMs int, failureLimit, successLimit int64) string {
	validations := []struct {
		value int
		cfg   InputConfig
	}{
		{port, PortConfig},
		{threadCount, ThreadConfig},
		{intervalMs, IntervalConfig},
		{int(failureLimit), FailureConfig},
		{int(successLimit), SuccessConfig},
	}
	for _, v := range validations {
		if _, msg := validateInput(v.value, v.cfg); msg != "" {
			return msg
		}
	}
	return ""
}
