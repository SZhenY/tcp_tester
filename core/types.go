// types.go — 核心类型定义

package core

// Stats 测试统计数据
type Stats struct {
	Success int64   `json:"success"` // 成功连接数
	Failure int64   `json:"failure"` // 失败连接数
	Total   int64   `json:"total"`   // 总连接数
	Rate    float64 `json:"rate"`    // 实时速率（每秒）
	AvgCPS  float64 `json:"avgCps"`  // 平均速率（每秒）
}

// EventEmitter 事件发射接口，解耦 GUI 依赖
// GUI 版本桥接到 Wails 运行时，CLI 版本输出到终端
type EventEmitter interface {
	EmitLog(msg string)              // 发送日志消息
	EmitStats(stats Stats)           // 发送统计数据
	EmitTestFinished(reason string)  // 发送测试完成事件
}

// InputConfig 输入参数校验配置
type InputConfig struct {
	MinValue   int    // 最小值
	MaxValue   int    // 最大值
	DefaultVal int    // 默认值
	Name       string // 参数名称（中文）
}
