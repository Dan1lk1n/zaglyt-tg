package z3abp

import (
	"container/list"
	"sync"
)

// lruCache is a small, size-bounded, concurrency-safe cache mapping a raw text
// line to its parsed morphological tokens. It caps memory by entry count so it
// stays predictable on RAM-constrained hosts. A single mystem subprocess still
// serializes uncached analysis, so this cache is what actually cuts the load:
// repeated lines (the corpus is re-analyzed on every response) are served from
// memory instead of round-tripping to the subprocess.
type lruCache struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List
	items map[string]*list.Element
}

type lruEntry struct {
	key string
	val []MorphToken
}

func newLRU(capacity int) *lruCache {
	return &lruCache{
		cap:   capacity,
		ll:    list.New(),
		items: make(map[string]*list.Element),
	}
}

func (c *lruCache) Get(key string) ([]MorphToken, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*lruEntry).val, true
	}
	return nil, false
}

func (c *lruCache) Add(key string, val []MorphToken) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*lruEntry).val = val
		return
	}

	el := c.ll.PushFront(&lruEntry{key: key, val: val})
	c.items[key] = el

	if c.cap > 0 && c.ll.Len() > c.cap {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*lruEntry).key)
		}
	}
}
