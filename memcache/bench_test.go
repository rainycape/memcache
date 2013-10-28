package memcache

import (
	"runtime"
	"strings"
	"sync"
	"testing"
)

func benchmarkSetGet(b *testing.B, item *Item) {
	cmd, c := newUnixServer(b)
	key := item.Key
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Set(item); err != nil {
			b.Fatal(err)
		}
		if _, err := c.Get(key); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	cmd.Process.Kill()
	cmd.Wait()
}

func largeItem() *Item {
	key := strings.Repeat("f", 240)
	value := make([]byte, 1024)
	return &Item{Key: key, Value: value}
}

func BenchmarkSetGet(b *testing.B) {
	benchmarkSetGet(b, &Item{Key: "foo", Value: []byte("bar")})
}

func BenchmarkSetGetLarge(b *testing.B) {
	benchmarkSetGet(b, largeItem())
}

func benchmarkConcurrentSetGetLarge(b *testing.B, count int, opcount int) {
	mp := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(mp)
	runtime.GOMAXPROCS(count)
	cmd, c := newUnixServer(b)
	c = New("localhost:11211")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(count)
		for j := 0; j < count; j++ {
			item := largeItem()
			key := item.Key
			go func() {
				defer wg.Done()
				for k := 0; k < opcount; k++ {
					if err := c.Set(item); err != nil {
						b.Fatal(err)
					}
					if _, err := c.Get(key); err != nil {
						b.Fatal(err)
					}
				}
			}()
		}
		wg.Wait()
	}
	b.StopTimer()
	cmd.Process.Kill()
	cmd.Wait()
}

func BenchmarkConcurrentSetGetLarge10(b *testing.B) {
	benchmarkConcurrentSetGetLarge(b, 10, 100)
}

func BenchmarkConcurrentSetGetLarge50(b *testing.B) {
	benchmarkConcurrentSetGetLarge(b, 50, 1000)
}
