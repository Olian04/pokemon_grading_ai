package market

import (
	"sync"
	"time"
)

type singlesCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[int]singlesCacheEntry
}

type singlesCacheEntry struct {
	until time.Time
	rows  []singleRow
}

func newSinglesCache(ttl time.Duration) *singlesCache {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &singlesCache{
		ttl:   ttl,
		items: make(map[int]singlesCacheEntry),
	}
}

func (c *singlesCache) get(idExpansion int) ([]singleRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[idExpansion]
	if !ok || time.Now().After(e.until) {
		return nil, false
	}
	return e.rows, true
}

func (c *singlesCache) set(idExpansion int, rows []singleRow) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[idExpansion] = singlesCacheEntry{
		until: time.Now().Add(c.ttl),
		rows:  rows,
	}
}
