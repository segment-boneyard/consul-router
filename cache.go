package main

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

// The cache type is an implementation of a resolver decorator that caches
// service endpoints returned by a base resolver using a LRU cache.
type cache struct {
	timeout time.Duration
	rslv    resolver
	cache   *lru.Cache
}

// The cacheConfig struct is used to configure newly created cache objects.
type cacheConfig struct {
	timeout time.Duration // how long objects live in the cache
	size    int           // how many objects the cache can hold
	rslv    resolver      // the base resolver to query on cache misses
}

type cacheEntry struct {
	sync.RWMutex
	srv []service
	err error
	exp time.Time
}

// cached returns a new cache object configured with config.
func cached(config cacheConfig) *cache {
	lruCache, _ := lru.New(config.size)
	return &cache{
		timeout: config.timeout,
		rslv:    config.rslv,
		cache:   lruCache,
	}
}

func (c *cache) resolve(name string) (srv []service, err error) {
	now := time.Now()

	for {
		if e := c.get(name); e != nil {
			if now.After(e.exp) {
				c.remove(name)
			} else {
				e.RLock()
				srv = e.srv
				err = e.err
				e.RUnlock()
				return
			}
		}

		e := &cacheEntry{exp: now.Add(c.timeout)}
		e.Lock()

		if !c.add(name, e) {
			continue
		}

		srv, err = c.rslv.resolve(name)
		e.srv = srv
		e.err = err
		e.Unlock()
		return
	}
}

func (c *cache) add(name string, entry *cacheEntry) bool {
	ok, _ := c.cache.ContainsOrAdd(name, entry)
	return !ok
}

func (c *cache) get(name string) (entry *cacheEntry) {
	if v, ok := c.cache.Get(name); ok {
		entry = v.(*cacheEntry)
	}
	return
}

func (c *cache) remove(name string) {
	c.cache.Remove(name)
}
