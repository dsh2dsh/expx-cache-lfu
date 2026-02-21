package lfu

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/dsh2dsh/go-tinylfu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "mykey"

func TestCache(t *testing.T) {
	cache := New(1000, time.Minute)
	require.NotNil(t, cache)

	const key = "key1"
	value := []byte("value1")

	cache.Set(key, value)
	b := cache.Get(key)
	assert.Equal(t, value, b)

	cache.Del(key)
	b = cache.Get(key)
	assert.Nil(t, b)
}

func TestCache_Get_CorruptionOnExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	strFor := func(i int) string {
		return fmt.Sprintf("a string %d", i)
	}
	keyName := func(i int) string {
		return fmt.Sprintf("key-%00000d", i)
	}

	cache := New(1000, time.Second)
	size := 50000
	// Put a bunch of stuff in the cache with a TTL of 1 second
	for i := range size {
		key := keyName(i)
		cache.Set(key, []byte(strFor(i)))
	}

	// Read stuff for a bit longer than the TTL - that's when the corruption
	// occurs.
	done := time.After(2 * time.Second)

loop:
	for {
		select {
		case <-done:
			// this is expected
			break loop
		default:
			i := rand.N(size)
			key := keyName(i)

			b := cache.Get(key)
			if b == nil {
				continue loop
			}

			got := string(b)
			expected := strFor(i)
			require.Equal(t, expected, got, "expected=%q got=%q key=%q", expected,
				got, key)
		}
	}
}

func TestNew_offset(t *testing.T) {
	tests := []struct {
		ttl      time.Duration
		expected time.Duration
	}{
		{
			ttl:      10 * time.Second,
			expected: time.Second,
		},
		{
			ttl:      100 * time.Second,
			expected: 10 * time.Second,
		},
		{
			ttl:      1000 * time.Second,
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.ttl.String(), func(t *testing.T) {
			cache := New(1000, tt.ttl)
			require.NotNil(t, cache)
			assert.Equal(t, tt.expected, cache.offset)
		})
	}
}

func TestCache_UseRandomizedTTL(t *testing.T) {
	cache := New(1000, 1000*time.Second)
	require.NotNil(t, cache)
	assert.Equal(t, 10*time.Second, cache.offset)

	cache.UseRandomizedTTL(10 * time.Hour)
	assert.Equal(t, 10*time.Hour, cache.offset)
}

func TestCache_Set_offset(t *testing.T) {
	ttl := 10 * time.Second
	cache := New(1000, ttl)
	require.NotNil(t, cache)

	var expireAt time.Time
	lfu := &moqBackend{
		SetFunc: func(item *tinylfu.Item[[]byte]) {
			expireAt = item.ExpireAt
		},
	}
	cache.lfu = lfu

	start := time.Now().Add(ttl)
	cache.Set(testKey, []byte("a string"))
	assert.WithinRange(t, expireAt, start, time.Now().Add(ttl+cache.offset))
}

func TestCache_Set_nil(t *testing.T) {
	cache := New(1000, 10*time.Second)
	require.NotNil(t, cache)
	cache.lfu = &moqBackend{}
	cache.Set(testKey, nil)
}
