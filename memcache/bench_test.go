package memcache

import (
	"strings"
	"testing"
)

func benchmarkSetGet(b *testing.B, item *Item) {
	cmd, c := newUnixServer(b)
	key := item.Key
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Set(item); err != nil {
			b.Error(err)
		}
		if _, err := c.Get(key); err != nil {
			b.Error(err)
		}
	}
	b.StopTimer()
	cmd.Process.Kill()
	cmd.Wait()
}

func BenchmarkSetGet(b *testing.B) {
	benchmarkSetGet(b, &Item{Key: "foo", Value: []byte("bar")})
}

func BenchmarkSetGetLarge(b *testing.B) {
	key := strings.Repeat("f", 240)
	value := make([]byte, 1024)
	benchmarkSetGet(b, &Item{Key: key, Value: value})
}
