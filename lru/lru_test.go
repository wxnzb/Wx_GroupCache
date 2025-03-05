package lru

import (
	"testing"
)

type simpletest struct {
	string
	int
}
type complextest struct {
	string
	simpletest
}

var testgets = []struct {
	name   string
	ToAdd  interface{}
	ToGet  interface{}
	Expect bool
}{
	{"string_hit", "wuxi", "wuxi", true},
	{"string_miss", "wuxi", "wx", false},
	{"simple_hit", simpletest{"wuxi", 1}, simpletest{"wuxi", 1}, true},
	{"simple_miss", simpletest{"wuxi", 1}, simpletest{"wx", 1}, false},
	{"complex_miss", complextest{"wuxi", simpletest{"wuxi", 1}}, complextest{"wuxi", simpletest{"wuxi", 2}}, false},
}

func Test_Get(t *testing.T) {
	for _, tt := range testgets {
		lru := New(0) //表示容量无限大
		lru.Add(tt.ToAdd, 12345)
		val, ok := lru.Get(tt.ToGet)
		if ok != tt.Expect {
			t.Fatalf("%s:cache hit=%v, expect hit=%v", tt.name, ok, !ok)
		} else if ok && val != 12345 {
			t.Fatalf("%s expected get to return 1234 but got %v", tt.name, val)
		}
	}
}
