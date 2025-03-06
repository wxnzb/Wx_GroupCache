package consistenthash

import (
	"fmt"
	"strconv"
	"testing"
)

func Test_Hashing(t *testing.T) {

	hash := NewMap(3, func(key []byte) uint32 {
		i, err := strconv.Atoi(string(key))
		if err != nil {
			panic(err)
		}
		return uint32(i)
	})
	// 2, 4, 6, 12, 14, 16, 22, 24, 26
	hash.Add("6", "4", "2")
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
	// Adds 8, 18, 28
	hash.Add("8")
	// 27 should now map to 8.
	testCases["27"] = "8"
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
func Test_Consistency(t *testing.T) {
	hash1 := NewMap(1, nil)
	hash2 := NewMap(1, nil)
	hash1.Add("rrr", "uuu", "iii")
	hash2.Add("iii", "rrr", "uuu")
	if hash1.Get("ppp") != hash2.Get("ppp") {
		t.Errorf("Consistent hashing failed")
	}
	hash2.Add("ppp", "ttt", "qqq")
	if hash1.Get("rrr") != hash2.Get("rrr") || hash1.Get("uuu") != hash2.Get("uuu") || hash1.Get("ppp") != hash2.Get("ppp") {
		t.Errorf("Consistent hashing failed")
	}
}
func Benchmark_8(b *testing.B)   { benchmark(b, 8) }
func Benchmark_32(b *testing.B)  { benchmark(b, 32) }
func Benchmark_128(b *testing.B) { benchmark(b, 128) }
func Benchmark_521(b *testing.B) { benchmark(b, 512) }
func benchmark(b *testing.B, n int) {
	hash := NewMap(50, nil)
	var buckets []string
	for i := 0; i < n; i++ {
		buckets = append(buckets, fmt.Sprintf("haha %d", i))
	}
	hash.Add(buckets...)
	//这句是在干什么
	b.ResetTimer()
	//b.N是什么
	for i := 0; i < b.N; i++ {
		//使他在0-n-1之间
		hash.Get(buckets[i&(n-1)])
	}
}
