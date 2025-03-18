package groupcache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	stringGroupName = "string-group"
	protoGroupName  = "proto-group"
	cacheSize       = 1 << 20
	fromChan        = "from-chan"
)

var StringGroup, ProtoGroup Getter
var stringc = make(chan string)
var cacheFills AtomicInt
var once sync.Once

func testSetup() {
	StringGroup = NewGroup(stringGroupName, cacheSize, GetterFunc(func(key string, dst Sink) error {
		if key == fromChan {
			key = <-stringc
		}
		cacheFills.Add(1)
		return dst.SetString("ECHO " + key)
	}))
	//  ProtoGroup=NewGroup(protoGroupName,cacheSize,GetterFunc(func(key string,dst SinK)error{
	// if key==fromChan{
	// 	key=<-stringc
	// 	cacheFills.Add(1)

	// }
	//  }))
}
func GetDupSupressString_Test(t *testing.T) {
	once.Do(testSetup)
	resc := make(chan string, 2)
	for i := 0; i < 2; i++ {
		go func() {
			var s string
			if err := StringGroup.Get(fromChan, StringSink(&s)); err != nil {
				resc <- "ERROR" + err.Error()
			}
			resc <- s
		}()
	}
	time.Sleep(25 * time.Millisecond)
	stringc <- "foo"
	for i := 0; i < 2; i++ {
		v := <-resc
		if v != "ECHO foo" {
			t.Errorf("got %s,want %s", v, "ECHOfoo")
		}
	}
	fmt.Printf("%d", cacheFills)
}

// 这个是为了证明第一次找没找到要调用回调函数，哪个回调函数里面有cacheFills.Add(1),但是之后就不用了
func TestCaching(t *testing.T) {
	once.Do(testSetup)
	fills := countFills(func() {
		for i := 0; i < 10; i++ {
			var v string
			if err := StringGroup.Get("nnn", StringSink(&v)); err != nil {
				t.Fatal(err)
			}
		}
	})
	if fills != 1 {
		t.Errorf("expected 1 fill, got %d", fills)
	}
}
func countFills(f func()) int64 {
	fills0 := cacheFills.Get()
	f()
	return cacheFills.Get() - fills0
}

// 缓存淘汰（eviction）机制是否生效。
func TestEnviction(t *testing.T) {
	once.Do(testSetup)
	getTestKey := func() {
		for i := 0; i < 10; i++ {
			var res string
			if err := StringGroup.Get("ppp", StringSink(&res)); err != nil {
				t.Fatal(err)
			}
		}
	}
	fills := countFills(getTestKey)
	if fills != 1 {
		t.Errorf("expected 1 fill,got %d", fills)
	}
	//开始用大量无用的key淘汰上面被缓存过的ppp
	g := StringGroup.(*Group)
	envicted0 := g.mainCache.nevicts
	var BytesFloot int64
	for BytesFloot < cacheSize+1024 {
		s := fmt.Sprintf("Hello-%d", BytesFloot)
		var res string
		if err := StringGroup.Get(s, StringSink(&res)); err != nil {
			t.Fatal(err)
		}
		BytesFloot += int64(len(s) + len(res))
	}
	evicts := g.mainCache.nevicts - envicted0
	if evicts != 1 {
		t.Errorf("expected 1 eviction,got %d", evicts)
	}
	//这个时候最上面哪个ppp已经被淘汰了
	fills = countFills(getTestKey)
	if fills != 1 {
		t.Errorf("expected 1 fill,got %d", fills)
	}
}
