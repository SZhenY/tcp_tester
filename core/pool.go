// pool.go — 分片连接池操作

package core

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

// connPoolShard 分片连接池，减少锁竞争
type connPoolShard struct {
	mu    sync.Mutex
	conns []net.Conn
}

// forceCloseConn 强制关闭连接（发送 RST 而非正常四次挥手）
func forceCloseConn(conn net.Conn) {
	if conn == nil {
		return
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetLinger(0)
	}
	conn.Close()
}

// initPool 初始化分片连接池
func (a *App) initPool(totalSize int) {
	a.poolShardCount = runtime.NumCPU()
	if a.poolShardCount < 1 {
		a.poolShardCount = 1
	}
	if totalSize <= 0 {
		totalSize = 1000
	}
	if totalSize > MaxConnPoolSize {
		totalSize = MaxConnPoolSize
	}
	a.poolSizePerShard = (totalSize + a.poolShardCount - 1) / a.poolShardCount
	if a.poolSizePerShard < 1 {
		a.poolSizePerShard = 1
	}
	a.poolShards = make([]connPoolShard, a.poolShardCount)
	for i := range a.poolShards {
		a.poolShards[i].conns = make([]net.Conn, a.poolSizePerShard)
	}
	a.poolIndex = 0
	a.poolClosed.Store(false)
}

// storeConn 将连接存入分片池，自动淘汰旧连接
func (a *App) storeConn(conn net.Conn) {
	if a.poolClosed.Load() {
		forceCloseConn(conn)
		return
	}

	idx := atomic.AddInt64(&a.poolIndex, 1) - 1
	shardIdx := int(idx % int64(a.poolShardCount))
	offset := int((idx / int64(a.poolShardCount)) % int64(a.poolSizePerShard))

	shard := &a.poolShards[shardIdx]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if a.poolClosed.Load() {
		forceCloseConn(conn)
		return
	}
	if offset >= 0 && offset < len(shard.conns) {
		if old := shard.conns[offset]; old != nil {
			forceCloseConn(old)
		}
		shard.conns[offset] = conn
	} else {
		forceCloseConn(conn)
	}
}

// closeAllConnections 同步关闭连接池中所有连接
func (a *App) closeAllConnections() {
	a.poolClosed.Store(true)
	for i := range a.poolShards {
		shard := &a.poolShards[i]
		shard.mu.Lock()
		conns := shard.conns
		shard.conns = make([]net.Conn, 0)
		shard.mu.Unlock()
		for _, c := range conns {
			forceCloseConn(c)
		}
	}
}
