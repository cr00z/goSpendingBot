package cache_lru

import (
	"container/list"
	"sync"

	"github.com/cr00z/goSpendingBot/internal/cache"
)

type LRUCache struct {
	name     string
	values   map[string]*list.Element
	queue    list.List
	capacity int
	sync.RWMutex
}

type Item struct {
	key   string
	value interface{}
}

func NewLRUCache(name string, capacity int) *LRUCache {
	return &LRUCache{
		name:     name,
		values:   make(map[string]*list.Element, capacity),
		queue:    list.List{},
		capacity: capacity,
	}
}

func (lru *LRUCache) Name() string {
	return lru.name
}

func (lru *LRUCache) Add(key string, value interface{}) (eviction bool) {
	lru.RWMutex.Lock()
	defer lru.RWMutex.Unlock()

	if element, inCache := lru.values[key]; inCache {
		element.Value.(*Item).value = value
		lru.queue.MoveToFront(element)
	} else {
		if len(lru.values) == lru.capacity {
			eviction = true
			removeElement := lru.queue.Back()
			removeItem := lru.queue.Remove(removeElement).(*Item)
			delete(lru.values, removeItem.key)
		}
		lru.values[key] = lru.queue.PushFront(&Item{key, value})
	}

	return eviction
}

func (lru *LRUCache) Get(key string) (interface{}, error) {
	lru.RWMutex.RLock()
	defer lru.RWMutex.RUnlock()

	if value, inCache := lru.values[key]; inCache {
		lru.queue.MoveToFront(value)
		return value.Value.(*Item).value, nil
	} else {
		return nil, cache.ErrElementNotInCache
	}

}

func (lru *LRUCache) Len() int {
	lru.RWMutex.RLock()
	defer lru.RWMutex.RUnlock()

	return len(lru.values)
}

func (lru *LRUCache) Delete(key string) error {
	lru.RWMutex.Lock()
	defer lru.RWMutex.Unlock()

	if value, inCache := lru.values[key]; inCache {
		delete(lru.values, value.Value.(*Item).key)
		lru.queue.Remove(value)
		return nil
	} else {
		return cache.ErrElementNotInCache
	}
}
