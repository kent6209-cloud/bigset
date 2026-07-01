package datasource

import (
	"container/list"
	"fmt"
	"sync"
)

type cacheEntry struct {
	z, x, y int
	data     []byte
	format   string
}

type CacheSource struct {
	inner DataSource
	lru   *list.List
	mu    sync.Mutex
	items map[[3]int]*list.Element
	cap   int
	hits  int64
	miss  int64
}

func NewCacheSource(inner DataSource, capacity int) *CacheSource {
	return &CacheSource{
		inner: inner,
		lru:   list.New(),
		items: make(map[[3]int]*list.Element),
		cap:   capacity,
	}
}

func (c *CacheSource) Name() string { return c.inner.Name() }
func (c *CacheSource) Type() string { return c.inner.Type() }
func (c *CacheSource) CRS() string  { return c.inner.CRS() }

func (c *CacheSource) GetTile(z, x, y int) ([]byte, string, error) {
	key := [3]int{z, x, y}
	c.mu.Lock()
	if el, ok := c.items[key]; ok {
		c.lru.MoveToFront(el)
		c.hits++
		entry := el.Value.(*cacheEntry)
		c.mu.Unlock()
		return entry.data, entry.format, nil
	}
	c.miss++
	c.mu.Unlock()

	data, format, err := c.inner.GetTile(z, x, y)
	if err != nil {
		return nil, "", err
	}

	entry := &cacheEntry{z: z, x: x, y: y, data: data, format: format}
	c.mu.Lock()
	if c.lru.Len() >= c.cap {
		back := c.lru.Back()
		if back != nil {
			evicted := back.Value.(*cacheEntry)
			delete(c.items, [3]int{evicted.z, evicted.x, evicted.y})
			c.lru.Remove(back)
		}
	}
	el := c.lru.PushFront(entry)
	c.items[key] = el
	c.mu.Unlock()
	return data, format, nil
}

func (c *CacheSource) GetFeatures(bbox BBox, targetCRS string) ([]Feature, error) {
	return c.inner.GetFeatures(bbox, targetCRS)
}

func (c *CacheSource) Stats() (hits, miss int64, ratio float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hits = c.hits
	miss = c.miss
	total := hits + miss
	if total > 0 {
		ratio = float64(hits) / float64(total) * 100
	}
	return
}

func (c *CacheSource) String() string {
	hits, miss, ratio := c.Stats()
	return fmt.Sprintf("Cache{%s: hits=%d miss=%d ratio=%.1f%%}", c.Name(), hits, miss, ratio)
}
