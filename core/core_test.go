package core

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// mockConn 模拟网络连接
type mockConn struct {
	closed bool
}

func (c *mockConn) Read(b []byte) (n int, err error)          { return 0, nil }
func (c *mockConn) Write(b []byte) (n int, err error)         { return len(b), nil }
func (c *mockConn) Close() error                              { c.closed = true; return nil }
func (c *mockConn) LocalAddr() net.Addr                       { return nil }
func (c *mockConn) RemoteAddr() net.Addr                      { return nil }
func (c *mockConn) SetDeadline(_ time.Time) error             { return nil }
func (c *mockConn) SetReadDeadline(_ time.Time) error         { return nil }
func (c *mockConn) SetWriteDeadline(_ time.Time) error        { return nil }

// mockEmitter 捕获所有事件输出
type mockEmitter struct {
	logs    []string
	stats   []Stats
	reasons []string
}

func (m *mockEmitter) EmitLog(msg string)            { m.logs = append(m.logs, msg) }
func (m *mockEmitter) EmitStats(stats Stats)          { m.stats = append(m.stats, stats) }
func (m *mockEmitter) EmitTestFinished(reason string) { m.reasons = append(m.reasons, reason) }

// --- ValidateInputs ---

func TestValidateInputs_Valid(t *testing.T) {
	if msg := ValidateInputs(80, 64, 10, 50, 1000); msg != "" {
		t.Errorf("expected empty, got %q", msg)
	}
}

func TestValidateInputs_PortTooLow(t *testing.T) {
	if msg := ValidateInputs(0, 64, 0, 50, 1000); !strings.Contains(msg, "不能小于") {
		t.Errorf("expected port too low error, got %q", msg)
	}
}

func TestValidateInputs_PortTooHigh(t *testing.T) {
	if msg := ValidateInputs(70000, 64, 0, 50, 1000); !strings.Contains(msg, "不能大于") {
		t.Errorf("expected port too high error, got %q", msg)
	}
}

func TestValidateInputs_ThreadTooLow(t *testing.T) {
	if msg := ValidateInputs(80, 0, 0, 50, 1000); !strings.Contains(msg, "并发数") {
		t.Errorf("expected thread error, got %q", msg)
	}
}

func TestValidateInputs_ThreadTooHigh(t *testing.T) {
	if msg := ValidateInputs(80, 10001, 0, 50, 1000); !strings.Contains(msg, "不能大于") {
		t.Errorf("expected thread too high error, got %q", msg)
	}
}

func TestValidateInputs_IntervalTooHigh(t *testing.T) {
	if msg := ValidateInputs(80, 64, 60001, 50, 1000); !strings.Contains(msg, "间隔") {
		t.Errorf("expected interval error, got %q", msg)
	}
}

func TestValidateInputs_FailureLimitTooLow(t *testing.T) {
	if msg := ValidateInputs(80, 64, 0, 0, 1000); !strings.Contains(msg, "失败停止") {
		t.Errorf("expected failure limit error, got %q", msg)
	}
}

func TestValidateInputs_SuccessLimitTooLow(t *testing.T) {
	if msg := ValidateInputs(80, 64, 0, 50, 0); !strings.Contains(msg, "成功上限") {
		t.Errorf("expected success limit error, got %q", msg)
	}
}

// --- validateDomain ---

func TestValidateDomain_Valid(t *testing.T) {
	for _, d := range []string{"example.com", "test-sub.domain.org", "a.b.c"} {
		if msg := validateDomain(d); msg != "" {
			t.Errorf("domain %q: expected empty, got %q", d, msg)
		}
	}
}

func TestValidateDomain_Empty(t *testing.T) {
	if msg := validateDomain(""); !strings.Contains(msg, "不能为空") {
		t.Errorf("expected empty error, got %q", msg)
	}
}

func TestValidateDomain_TooLong(t *testing.T) {
	long := strings.Repeat("a", 254)
	if msg := validateDomain(long); !strings.Contains(msg, "253") {
		t.Errorf("expected length error, got %q", msg)
	}
}

func TestValidateDomain_InvalidChars(t *testing.T) {
	for _, c := range []string{"exam ple.com", "test@domain.com", "a_b/c.com"} {
		if msg := validateDomain(c); !strings.Contains(msg, "非法字符") {
			t.Errorf("domain %q: expected invalid char error, got %q", c, msg)
		}
	}
}

// --- Pool ---

func TestInitPool(t *testing.T) {
	a := New(&mockEmitter{})
	a.initPool(100)

	if a.poolShardCount < 1 {
		t.Error("poolShardCount should be >= 1")
	}
	if a.poolSizePerShard < 1 {
		t.Error("poolSizePerShard should be >= 1")
	}
	if len(a.poolShards) != a.poolShardCount {
		t.Errorf("poolShards length %d != poolShardCount %d", len(a.poolShards), a.poolShardCount)
	}
	expectedTotal := a.poolShardCount * a.poolSizePerShard
	if expectedTotal < 100 {
		t.Errorf("total capacity %d < requested %d", expectedTotal, 100)
	}
}

func TestInitPool_ZeroSize(t *testing.T) {
	a := New(&mockEmitter{})
	a.initPool(0)
	if a.poolSizePerShard < 1 {
		t.Error("poolSizePerShard should be >= 1 for zero size")
	}
}

func TestInitPool_ExceedsMax(t *testing.T) {
	a := New(&mockEmitter{})
	a.initPool(MaxConnPoolSize + 500000)
	total := a.poolShardCount * a.poolSizePerShard
	if total > MaxConnPoolSize+a.poolShardCount {
		t.Errorf("total capacity %d exceeds max %d by too much", total, MaxConnPoolSize)
	}
}

func TestStoreConn_And_CloseAll(t *testing.T) {
	a := New(&mockEmitter{})
	a.initPool(10)

	// Create a mock connection
	conn := &mockConn{}
	a.storeConn(conn)

	if a.poolClosed.Load() {
		t.Error("pool should not be closed yet")
	}

	a.closeAllConnections()
	if !a.poolClosed.Load() {
		t.Error("pool should be closed after closeAllConnections")
	}
	if !conn.closed {
		t.Error("connection should be closed")
	}
}

func TestForceCloseConn_Nil(t *testing.T) {
	forceCloseConn(nil) // should not panic
}

// --- checkLimits ---

func TestCheckLimits(t *testing.T) {
	a := New(&mockEmitter{})
	a.successCount = 100
	a.failureCount = 30

	if err := a.checkLimits(50, 200); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCheckLimits_FailureReached(t *testing.T) {
	a := New(&mockEmitter{})
	a.failureCount = 50

	err := a.checkLimits(50, 1000)
	if !errors.Is(err, ErrFailureLimit) {
		t.Errorf("expected ErrFailureLimit, got %v", err)
	}
}

func TestCheckLimits_SuccessReached(t *testing.T) {
	a := New(&mockEmitter{})
	a.successCount = 1000

	err := a.checkLimits(50, 1000)
	if !errors.Is(err, ErrSuccessLimit) {
		t.Errorf("expected ErrSuccessLimit, got %v", err)
	}
}

// --- determineStopReason ---

func TestDetermineStopReason_Manual(t *testing.T) {
	a := New(&mockEmitter{})
	a.stopped.Store(true)

	err := a.determineStopReason(50, 1000)
	if !errors.Is(err, ErrManualStop) {
		t.Errorf("expected ErrManualStop, got %v", err)
	}
}

func TestDetermineStopReason_FailureLimit(t *testing.T) {
	a := New(&mockEmitter{})
	a.failureCount = 50

	err := a.determineStopReason(50, 1000)
	if !errors.Is(err, ErrFailureLimit) {
		t.Errorf("expected ErrFailureLimit, got %v", err)
	}
}

func TestDetermineStopReason_SuccessLimit(t *testing.T) {
	a := New(&mockEmitter{})
	a.successCount = 1000

	err := a.determineStopReason(50, 1000)
	if !errors.Is(err, ErrSuccessLimit) {
		t.Errorf("expected ErrSuccessLimit, got %v", err)
	}
}

func TestDetermineStopReason_NormalEnd(t *testing.T) {
	a := New(&mockEmitter{})

	err := a.determineStopReason(50, 1000)
	if !errors.Is(err, ErrNormalEnd) {
		t.Errorf("expected ErrNormalEnd, got %v", err)
	}
}

// --- New ---

func TestNew(t *testing.T) {
	emitter := &mockEmitter{}
	a := New(emitter)
	if a == nil {
		t.Fatal("New returned nil")
	}
	if a.emitter != emitter {
		t.Error("emitter not set")
	}
	if a.stopChan == nil {
		t.Error("stopChan not initialized")
	}
}

// --- StartTest ---

func TestStartTest_AlreadyRunning(t *testing.T) {
	a := New(&mockEmitter{})
	a.isRunning.Store(true)

	err := a.StartTest("127.0.0.1:80", 1, 0, 10, 100)
	if err == nil || !strings.Contains(err.Error(), "已在运行") {
		t.Errorf("expected already running error, got %v", err)
	}
}

func TestStartTest_InvalidTarget(t *testing.T) {
	a := New(&mockEmitter{})

	err := a.StartTest("invalid-no-port", 1, 0, 10, 100)
	if err == nil || !strings.Contains(err.Error(), "目标地址格式错误") {
		t.Errorf("expected format error, got %v", err)
	}
}

// --- timestamp ---

func TestTimestamp(t *testing.T) {
	ts := timestamp()
	// Format: HH:MM:SS.mmm
	if len(ts) != 12 || ts[2] != ':' || ts[5] != ':' || ts[8] != '.' {
		t.Errorf("unexpected timestamp format: %q", ts)
	}
}

// --- Stats ---

func TestStats_Fields(t *testing.T) {
	s := Stats{Success: 10, Failure: 5, Total: 15, Rate: 2.5, AvgCPS: 3.0}
	if s.Success != 10 || s.Failure != 5 || s.Total != 15 {
		t.Error("stats fields mismatch")
	}
	if s.Rate != 2.5 || s.AvgCPS != 3.0 {
		t.Error("stats rate fields mismatch")
	}
}
