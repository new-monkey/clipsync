package hub

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	interval time.Duration
	clients  map[string]*clientRate
}

type clientRate struct {
	count     int
	window    time.Time
	blockedUntil time.Time
}

func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		interval: interval,
		clients:  make(map[string]*clientRate),
	}
}

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	cr, ok := rl.clients[clientID]
	if !ok {
		cr = &clientRate{count: 0, window: now}
		rl.clients[clientID] = cr
	}

	if now.Before(cr.blockedUntil) {
		return false
	}

	if now.Sub(cr.window) > rl.interval {
		cr.count = 1
		cr.window = now
		return true
	}

	if cr.count >= rl.limit {
		cr.blockedUntil = now.Add(rl.interval)
		return false
	}

	cr.count++
	return true
}