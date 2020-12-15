package cachelru

import lru "github.com/hashicorp/golang-lru"

func NewLRU(lru *lru.ARCCache) *LRU {
	return &LRU{cache: lru}
}

type LRU struct {
	cache *lru.ARCCache
}

func (c *LRU) Get(x interface{}) (interface{}, bool) {
	return c.cache.Get(x)
}

func (c *LRU) Add(key, value interface{}) {
	c.cache.Add(key, value)
}

func (c *LRU) Keys() []interface{} {
	return c.cache.Keys()
}
