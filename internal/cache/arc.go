package cache

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)

func NewLRU(size int) (*LRU, error) {
	c, err := lru.NewARC(size)
	if err != nil {
		return nil, fmt.Errorf("lru new instance of lru arc cache: %w", err)
	}

	return &LRU{cache: c}, nil
}

var _ Cache = (*LRU)(nil)

type LRU struct {
	cache *lru.ARCCache
}

func (c *LRU) Get(key interface{}) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *LRU) Add(key, value interface{}) {
	c.cache.Add(key, value)
}

func (c *LRU) Keys() []interface{} {
	return c.cache.Keys()
}

func (c *LRU) Delete(key interface{}) {
	c.cache.Remove(key)
}
