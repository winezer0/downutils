package downutils

import (
	"sync"
	"time"
)

// RateLimiter 基于令牌桶算法的速率限制器
type RateLimiter struct {
	rate       int64   // 令牌生成速率(字节/秒)
	bucketSize int64   // 桶容量(最大令牌数)
	tokens     float64 // 当前可用令牌数
	lastTime   time.Time
	mu         sync.Mutex
}

// NewRateLimiter 创建速率限制器
// rate: 令牌生成速率(字节/秒)，0表示不限速
func NewRateLimiter(rate int64) *RateLimiter {
	if rate <= 0 {
		return nil
	}
	return &RateLimiter{
		rate:       rate,
		bucketSize: rate,
		tokens:     float64(rate),
		lastTime:   time.Now(),
	}
}

// Wait 等待足够的令牌，返回需要等待的时间
// 如果不限速或令牌充足，返回0
func (rl *RateLimiter) Wait(bytes int64) time.Duration {
	if rl == nil {
		return 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.tokens += elapsed * float64(rl.rate)
	if rl.tokens > float64(rl.bucketSize) {
		rl.tokens = float64(rl.bucketSize)
	}
	rl.lastTime = now

	// 检查是否有足够的令牌
	if rl.tokens >= float64(bytes) {
		rl.tokens -= float64(bytes)
		return 0
	}

	// 计算需要等待的时间
	needed := float64(bytes) - rl.tokens
	waitDuration := time.Duration(needed/float64(rl.rate)*1000) * time.Millisecond

	// 扣除令牌(预扣)
	rl.tokens = 0

	return waitDuration
}

// LimitReader 包装io.Reader，在读取时应用速率限制
type LimitReader struct {
	reader  interface{ Read([]byte) (int, error) }
	limiter *RateLimiter
}

// NewLimitReader 创建限速读取器
func NewLimitReader(reader interface{ Read([]byte) (int, error) }, rate int64) *LimitReader {
	return &LimitReader{
		reader:  reader,
		limiter: NewRateLimiter(rate),
	}
}

// Read 读取数据并应用速率限制
func (lr *LimitReader) Read(p []byte) (n int, err error) {
	n, err = lr.reader.Read(p)
	if n > 0 && lr.limiter != nil {
		waitDuration := lr.limiter.Wait(int64(n))
		if waitDuration > 0 {
			time.Sleep(waitDuration)
		}
	}
	return n, err
}
