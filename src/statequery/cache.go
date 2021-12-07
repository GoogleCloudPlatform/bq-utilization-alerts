// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statequery

import (
	"fmt"
	"sync"
	"time"
)

// Cache type to hold frequently requested responses from the Resource Manager API.
type Cache struct {
	mutex  sync.Mutex
	maxTTL time.Duration
	items  map[string]*cacheItem
}

// Single cacheItem, timestamped to control freshness
type cacheItem struct {
	updated  time.Time
	projects []string
}

// Add an item to the cache
func (cache *Cache) Add(key string, projects []string) {
	cache.mutex.Lock()

	cache.items[key] = &cacheItem{
		updated:  time.Now().UTC(),
		projects: projects,
	}

	cache.mutex.Unlock()
}

// Get an item from the cache. Will return items if available and fresh enough.
func (cache *Cache) Get(key string) ([]string, error) {
	cache.mutex.Lock()

	item, ok := cache.items[key]
	if !ok {
		// Cache miss
		cache.mutex.Unlock()
		return nil, fmt.Errorf("key not in cache: %s", key)
	}
	// Cache hit

	// Check cache freshness
	diff := time.Now().UTC().Sub(item.updated)
	if diff >= cache.maxTTL {
		// Cache is stale
		delete(cache.items, key)
		cache.mutex.Unlock()
		return nil, fmt.Errorf("cache appears stale for key: %s", key)
	}

	// Cache item OK
	cache.mutex.Unlock()
	return cache.items[key].projects, nil
}

// Initialize and configure cache
func (cache *Cache) Initialize(maxTTL time.Duration) {
	cache.items = make(map[string]*cacheItem)
	cache.maxTTL = maxTTL
}
