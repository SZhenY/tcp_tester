// core.go — TCP 连接测试核心引擎

package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// 预定义错误常量，避免热路径中 fmt.Sprintf 开销
var (
	ErrFailureLimit = errors.New("达到失败上限")
	ErrSuccessLimit = errors.New("达到成功上限")
	ErrManualStop   = errors.New("手动停止")
	ErrNormalEnd    = errors.New("测试结束")
)

// App TCP 连接测试引擎
type App struct {
	emitter EventEmitter // 事件发射器，解耦 GUI/CLI 依赖

	isRunning  atomic.Bool // 测试是否正在运行
	stopped    atomic.Bool // 是否已请求停止
	finishOnce sync.Once   // 确保 finishTest 只执行一次

	stopChan chan struct{} // 停止信号通道

	successCount int64 // 成功连接计数
	failureCount int64 // 失败连接计数
	checkCounter int64 // 限制检查计数器（降频用）

	poolShards       []connPoolShard // 分片连接池
	poolShardCount   int             // 分片数量
	poolIndex        int64           // 连接池全局索引（原子操作）
	poolSizePerShard int             // 每个分片的容量
	poolClosed       atomic.Bool     // 连接池是否已关闭

	testWg        sync.WaitGroup    // 测试 goroutine 等待组
	testCtx       context.Context   // 测试上下文
	testCancel    context.CancelFunc // 取消测试的函数
	ctxMu         sync.RWMutex      // 上下文读写锁
	testStartTime time.Time         // 测试开始时间
}

// New 创建新的测试引擎实例
func New(emitter EventEmitter) *App {
	return &App{
		emitter:  emitter,
		stopChan: make(chan struct{}, 1),
	}
}

// Wait 等待测试完成
func (a *App) Wait() {
	a.testWg.Wait()
}

// StartTest 开始 TCP 连接测试
func (a *App) StartTest(target string, threadCount int, intervalMs int, failureLimit, successLimit int64) error {
	// 原子检查并设置运行状态，防止并发启动
	if !a.isRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("测试已在运行中")
	}

	// 校验目标地址格式
	_, portStr, err := net.SplitHostPort(target)
	if err != nil {
		a.isRunning.Store(false)
		return fmt.Errorf("目标地址格式错误: %v", err)
	}
	port, _ := strconv.Atoi(portStr)

	// 校验输入参数
	if msg := ValidateInputs(port, threadCount, intervalMs, failureLimit, successLimit); msg != "" {
		a.isRunning.Store(false)
		return fmt.Errorf(msg)
	}

	// 重置状态
	a.successCount = 0
	a.failureCount = 0
	a.stopped.Store(false)
	a.finishOnce = sync.Once{}

	// 创建新的测试上下文
	a.ctxMu.Lock()
	if a.testCancel != nil {
		a.testCancel()
	}
	a.testCtx, a.testCancel = context.WithCancel(context.Background())
	a.ctxMu.Unlock()

	// 初始化连接池
	a.initPool(int(successLimit))
	a.testStartTime = time.Now()

	// 发送启动日志
	a.emitter.EmitLog(fmt.Sprintf("[%s] 开始测试 -> %s (并发:%d 间隔:%dms)", timestamp(), target, threadCount, intervalMs))
	a.emitter.EmitLog(fmt.Sprintf("[%s] 连接池已初始化 (分片模式: %dx%d)", timestamp(), a.poolShardCount, a.poolSizePerShard))

	// 启动测试和统计
	a.testWg.Add(2)
	go a.runTest(target, threadCount, intervalMs, failureLimit, successLimit)
	go a.statsTicker()

	return nil
}

// StopTest 请求停止测试
func (a *App) StopTest() {
	if !a.isRunning.Load() || a.stopped.Swap(true) {
		return
	}
	a.emitter.EmitLog(fmt.Sprintf("[%s] 正在停止...", timestamp()))
	select {
	case a.stopChan <- struct{}{}:
	default:
	}
}

// Cleanup 清理所有资源
func (a *App) Cleanup() {
	a.stopped.Store(true)
	if a.testCancel != nil {
		a.testCancel()
	}
	a.testWg.Wait()
	a.closeAllConnections()
}

// runTest 测试主循环（Worker Pool 模式）
func (a *App) runTest(target string, threadCount int, intervalMs int, failureLimit, successLimit int64) {
	defer a.testWg.Done()

	a.ctxMu.RLock()
	ctx := a.testCtx
	a.ctxMu.RUnlock()

	// 创建工作通道和 Worker Pool（增大缓冲区减少阻塞）
	workChan := make(chan string, threadCount*4)
	var workerWg sync.WaitGroup

	// 预分配 Dialer，所有 worker 共享
	var dialer net.Dialer
	dialer.Timeout = 3 * time.Second

	// 启动持久化 worker（使用本地计数器减少原子操作竞争）
	for i := 0; i < threadCount; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			var localSucc, localFail int64
			for t := range workChan {
				a.testConnection(ctx, t, &dialer, &localSucc, &localFail)
			}
			// flush 本地计数器到全局
			atomic.AddInt64(&a.successCount, localSucc)
			atomic.AddInt64(&a.failureCount, localFail)
		}()
	}

	var lastBatchTime int64

	// 主循环：发送任务到工作通道
RunLoop:
	for {
		if a.stopped.Load() {
			break RunLoop
		}

		// 降频检查限制：每 1000 次检查一次
		if atomic.AddInt64(&a.checkCounter, 1)%1000 == 0 {
			if reason := a.checkLimits(failureLimit, successLimit); reason != nil {
				a.emitter.EmitLog(fmt.Sprintf("[%s] %s", timestamp(), reason.Error()))
				break RunLoop
			}
		}

		// 间隔控制
		if intervalMs > 0 {
			now := time.Now().UnixMilli()
			if lastBatchTime > 0 && now-lastBatchTime < int64(intervalMs) {
				time.Sleep(time.Duration(int64(intervalMs)-(now-lastBatchTime)) * time.Millisecond)
			}
			lastBatchTime = time.Now().UnixMilli()
		}

		// 非阻塞发送任务，满时让出 CPU 重试
		select {
		case workChan <- target:
		default:
			runtime.Gosched()
		}
	}

	// 等待所有 worker 完成
	close(workChan)
	a.ctxMu.RLock()
	if a.testCancel != nil {
		a.testCancel()
	}
	a.ctxMu.RUnlock()
	workerWg.Wait()

	// 确定停止原因并完成测试
	a.finishTest(a.determineStopReason(failureLimit, successLimit))
}

// checkLimits 检查是否达到成功/失败上限
func (a *App) checkLimits(failureLimit, successLimit int64) error {
	if atomic.LoadInt64(&a.failureCount) >= failureLimit {
		return ErrFailureLimit
	}
	if atomic.LoadInt64(&a.successCount) >= successLimit {
		return ErrSuccessLimit
	}
	return nil
}

// determineStopReason 确定测试停止原因
func (a *App) determineStopReason(failureLimit, successLimit int64) error {
	if a.stopped.Load() {
		return ErrManualStop
	}
	if atomic.LoadInt64(&a.failureCount) >= failureLimit {
		return ErrFailureLimit
	}
	if atomic.LoadInt64(&a.successCount) >= successLimit {
		return ErrSuccessLimit
	}
	return ErrNormalEnd
}

// testConnection 执行单次 TCP 连接测试（使用本地计数器减少原子操作）
func (a *App) testConnection(ctx context.Context, target string, dialer *net.Dialer, localSucc, localFail *int64) {
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		if ctx.Err() != nil {
			return // 上下文取消，不计为失败
		}
		*localFail++
		return
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		forceCloseConn(conn)
		return
	default:
	}

	*localSucc++

	// 设置 TCP 连接参数
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetReadBuffer(1024)
		tcpConn.SetWriteBuffer(1024)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(10 * time.Second)
		tcpConn.SetNoDelay(true)
	}

	// 存入连接池
	a.storeConn(conn)
}

// finishTest 完成测试，发送最终统计和日志
func (a *App) finishTest(stopReason error) {
	a.finishOnce.Do(func() {
		a.isRunning.Store(false)
		a.stopped.Store(false)

		// 断开连接
		a.emitter.EmitLog(fmt.Sprintf("[%s] 正在断开连接...", timestamp()))
		a.closeAllConnections()
		a.emitter.EmitLog(fmt.Sprintf("[%s] 连接已断开", timestamp()))

		// 计算最终统计
		finalSucc := atomic.LoadInt64(&a.successCount)
		finalFail := atomic.LoadInt64(&a.failureCount)
		totalElapsed := time.Since(a.testStartTime).Seconds()
		avgCps := float64(0)
		if totalElapsed > 0 {
			avgCps = float64(finalSucc+finalFail) / totalElapsed
		}

		// 发送最终统计和完成事件
		a.emitter.EmitStats(Stats{
			Success: finalSucc,
			Failure: finalFail,
			Total:   finalSucc + finalFail,
			Rate:    0,
			AvgCPS:  avgCps,
		})
		a.emitter.EmitLog(fmt.Sprintf("[%s] 测试完成 - 总成功: %d 总失败: %d 原因: %s", timestamp(), finalSucc, finalFail, stopReason.Error()))
		a.emitter.EmitTestFinished(stopReason.Error())

		// 释放连接池引用，允许 GC 回收
		a.poolShards = nil
	})
}

// statsTicker 每秒发送一次统计数据
func (a *App) statsTicker() {
	defer a.testWg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastSucc, lastFail int64
	lastTime := time.Now()

	for range ticker.C {
		if !a.isRunning.Load() {
			return
		}

		succ := atomic.LoadInt64(&a.successCount)
		fail := atomic.LoadInt64(&a.failureCount)

		// 计算实时速率
		newTotal := (succ - lastSucc) + (fail - lastFail)
		elapsed := time.Since(lastTime).Seconds()
		rate := float64(0)
		if elapsed > 0 {
			rate = float64(newTotal) / elapsed
		}

		// 计算平均速率
		totalElapsed := time.Since(a.testStartTime).Seconds()
		avgCps := float64(0)
		if totalElapsed > 0 {
			avgCps = float64(succ+fail) / totalElapsed
		}

		a.emitter.EmitStats(Stats{
			Success: succ,
			Failure: fail,
			Total:   succ + fail,
			Rate:    rate,
			AvgCPS:  avgCps,
		})

		lastSucc = succ
		lastFail = fail
		lastTime = time.Now()
	}
}
