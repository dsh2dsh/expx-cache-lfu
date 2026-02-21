//go:generate mockery
package lfu

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/dsh2dsh/go-tinylfu"
)

type LFU interface {
	Get(key string) ([]byte, bool)
	Set(*tinylfu.Item[[]byte])
	Del(key string)
}

func New(size int, ttl time.Duration) *Cache {
	const maxOffset = 10 * time.Second
	offset := min(maxOffset, ttl/10)

	return &Cache{
		lfu:    tinylfu.New[[]byte](size, 100000),
		ttl:    ttl,
		offset: offset,
	}
}

type Cache struct {
	mu     sync.Mutex
	lfu    LFU
	ttl    time.Duration
	offset time.Duration
}

func (self *Cache) UseRandomizedTTL(offset time.Duration) {
	self.offset = offset
}

func (self *Cache) Set(key string, b []byte) {
	if len(b) == 0 {
		return
	}

	ttl := self.ttl
	//nolint:gosec // I think weak rand is ok here
	if self.offset > 0 {
		ttl += rand.N(self.offset)
	}

	self.mu.Lock()
	defer self.mu.Unlock()

	self.lfu.Set(tinylfu.NewItemExpire(key, b, time.Now().Add(ttl)))
}

func (self *Cache) Get(key string) []byte {
	self.mu.Lock()
	defer self.mu.Unlock()

	b, ok := self.lfu.Get(key)
	if !ok {
		return nil
	}
	return b
}

func (self *Cache) Del(key string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.lfu.Del(key)
}
