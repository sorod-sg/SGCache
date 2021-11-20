package sgcache

import "sync"

type cache struct {
	mu         sync.Mutex
	lru        *LRUqueen
	cacheBytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Push(key, value)
	return
}

func (c *cache) get(key string) (value ByteView, isExist bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	valueInterface, ok := c.lru.Get(key)
	if ok == true {
		value = valueInterface.(ByteView)
		isExist = ok
	} else {
		value = ByteView{}
		isExist = false
	}

	return
}
