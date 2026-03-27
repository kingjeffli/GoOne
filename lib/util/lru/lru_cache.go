package lru

import (
	"container/list"
	"errors"
	"sync"
)

type CacheNode struct {
	Key, Value interface{}
}

// LRUCache is a simple thread-safe LRU cache.
// It uses a single RWMutex to guard both the list and the map to
// avoid deadlocks and inconsistent views.
type LRUCache struct {
	Capacity int32
	dropList *list.List
	cacheMap map[interface{}]*list.Element

	mu sync.RWMutex
}

func NewLRUCache(cap int32) *LRUCache {
	if cap <= 0 {
		cap = 1
	}
	return &LRUCache{
		Capacity: cap,
		dropList: list.New(),
		cacheMap: make(map[interface{}]*list.Element),
	}
}

func (lru *LRUCache) Size() int {
	lru.mu.RLock()
	defer lru.mu.RUnlock()
	if lru.dropList == nil {
		return 0
	}
	return lru.dropList.Len()
}

// Set inserts or updates a key. If capacity is exceeded, the least
// recently used entry is evicted.
func (lru *LRUCache) Set(k, v interface{}) error {
	if lru == nil {
		return errors.New("LRUCache is nil")
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()

	if lru.dropList == nil || lru.cacheMap == nil {
		return errors.New("LRUCache not init")
	}

	if elem, ok := lru.cacheMap[k]; ok {
		// update existing
		lru.dropList.MoveToFront(elem)
		if node, ok := elem.Value.(*CacheNode); ok {
			node.Value = v
		} else {
			// should not happen, but recover gracefully
			elem.Value = &CacheNode{Key: k, Value: v}
		}
		return nil
	}

	// insert new
	elem := lru.dropList.PushFront(&CacheNode{Key: k, Value: v})
	lru.cacheMap[k] = elem

	// evict least recently used if over capacity
	for int32(lru.dropList.Len()) > lru.Capacity {
		last := lru.dropList.Back()
		if last == nil {
			break
		}
		node, _ := last.Value.(*CacheNode)
		if node != nil {
			delete(lru.cacheMap, node.Key)
		}
		lru.dropList.Remove(last)
	}
	return nil
}

// Get returns value, found flag and error.
// When key is found, it is moved to front as most recently used.
func (lru *LRUCache) Get(k interface{}) (v interface{}, found bool, err error) {
	if lru == nil {
		return nil, false, errors.New("LRUCache is nil")
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()

	if lru.cacheMap == nil || lru.dropList == nil {
		return nil, false, errors.New("LRUCache not init")
	}

	if elem, ok := lru.cacheMap[k]; ok {
		lru.dropList.MoveToFront(elem)
		if node, ok := elem.Value.(*CacheNode); ok {
			return node.Value, true, nil
		}
		// unexpected type, treat as missing
		return nil, false, errors.New("LRUCache internal node type error")
	}
	return nil, false, nil
}

// Peek returns value, found flag and error without updating LRU order.
// It is useful for read-only inspection that should not affect eviction order.
func (lru *LRUCache) Peek(k interface{}) (v interface{}, found bool, err error) {
	if lru == nil {
		return nil, false, errors.New("LRUCache is nil")
	}

	lru.mu.RLock()
	defer lru.mu.RUnlock()

	if lru.cacheMap == nil || lru.dropList == nil {
		return nil, false, errors.New("LRUCache not init")
	}

	if elem, ok := lru.cacheMap[k]; ok {
		if node, ok := elem.Value.(*CacheNode); ok {
			return node.Value, true, nil
		}
		return nil, false, errors.New("LRUCache internal node type error")
	}
	return nil, false, nil
}

// Remove deletes a key if it exists, and returns true when removed.
func (lru *LRUCache) Remove(k interface{}) bool {
	if lru == nil {
		return false
	}

	lru.mu.Lock()
	defer lru.mu.Unlock()

	if lru.cacheMap == nil || lru.dropList == nil {
		return false
	}

	if elem, ok := lru.cacheMap[k]; ok {
		if node, ok := elem.Value.(*CacheNode); ok {
			delete(lru.cacheMap, node.Key)
		} else {
			delete(lru.cacheMap, k)
		}
		lru.dropList.Remove(elem)
		return true
	}
	return false
}
