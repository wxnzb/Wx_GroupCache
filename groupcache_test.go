package groupcache

import (
	"context"
	"errors"
	"fmt"
	pb "groupcache/groupcachepb"

	// pb "groupcache/groupcachepb"
	// "groupcache/singleflight"
	"hash/crc32"
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
var dummyCtx = context.TODO()

func testSetup() {
	StringGroup = NewGroup(stringGroupName, cacheSize, GetterFunc(func(_ context.Context, key string, dst Sink) error {
		if key == fromChan {
			key = <-stringc
		}
		cacheFills.Add(1)
		return dst.SetString("ECHO:" + key)
	}))
	//  ProtoGroup=NewGroup(protoGroupName,cacheSize,GetterFunc(func(key string,dst SinK)error{
	// if key==fromChan{
	// 	key=<-stringc
	// 	cacheFills.Add(1)

	// }
	//  }))
}
func TestGetDupSupressString_Test(t *testing.T) {
	once.Do(testSetup)
	resc := make(chan string, 2)
	for i := 0; i < 2; i++ {
		go func() {
			var s string
			if err := StringGroup.Get(dummyCtx, fromChan, StringSink(&s)); err != nil {
				resc <- "ERROR" + err.Error()
			}
			resc <- s
		}()
	}
	time.Sleep(25 * time.Millisecond)
	stringc <- "foo"
	for i := 0; i < 2; i++ {
		select {
		case v := <-resc:
			if v != "ECHO:foo" {
				t.Errorf("got %s,want %s", v, "ECHOfoo")
			}
		case <-time.After(5 * time.Second):
			t.Errorf("timeout waiting om getter #%d of 2", i+1)
		}
	}
}

// 这个是为了证明第一次找没找到要调用回调函数，哪个回调函数里面有cacheFills.Add(1),但是之后就不用了
func TestCaching(t *testing.T) {
	once.Do(testSetup)
	fills := countFills(func() {
		for i := 0; i < 10; i++ {
			var v string
			if err := StringGroup.Get(dummyCtx, "nnn", StringSink(&v)); err != nil {
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
			if err := StringGroup.Get(dummyCtx, "ppp", StringSink(&res)); err != nil {
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
		if err := StringGroup.Get(dummyCtx, s, StringSink(&res)); err != nil {
			t.Fatal(err)
		}
		BytesFloot += int64(len(s) + len(res))
	}
	evicts := g.mainCache.nevicts - envicted0
	if evicts <= 0 {
		t.Errorf("expected 1 eviction,got %d", evicts)
	}
	//这个时候最上面哪个ppp已经被淘汰了
	fills = countFills(getTestKey)
	if fills != 1 {
		t.Errorf("expected 1 fill,got %d", fills)
	}
}

type fackPeer struct {
	hits int
	fail bool
}

// 让他可以使用peers里面的这个
//
//	type PeerGetter interface {
//		Get(*pb.GetRequest, *pb.GetResponse) error
//	}
//
// 这个是调用远程节点是用的
func (f *fackPeer) Get(_ context.Context, request *pb.GetRequest, response *pb.GetResponse) error {
	f.hits++
	if f.fail {
		return errors.New("fail")
	}
	response.Value = []byte("got:" + request.GetKey())
	return nil
}

type fackPeers []PeerGetter

func (f fackPeers) PickPeer(key string) (PeerGetter, bool) {
	if len(f) == 0 {
		return nil, false
	}
	n := crc32.Checksum([]byte(key), crc32.IEEETable) % uint32(len(f))
	return f[n], f[n] != nil
}
func TestPeers(t *testing.T) {
	once.Do(testSetup)
	peer0 := &fackPeer{}
	peer1 := &fackPeer{}
	peer2 := &fackPeer{}
	peersList := fackPeers([]PeerGetter{peer0, peer1, peer2, nil})
	localHits := 0
	//这个是本地调用
	get := func(_ context.Context, key string, dst Sink) error {
		localHits++
		return dst.SetString("got:" + key)
	}
	testGroup := newGroup("test-peers-group", 0, GetterFunc(get), peersList)
	run := func(name string, n int) {
		localHits = 0
		for _, p := range peersList {
			p.(*fackPeer).hits = 0
		}
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("key-%d", i)
			want := "got:" + key
			var got string
			if err := testGroup.Get(dummyCtx, key, StringSink(&got)); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatal("bad value")
			}
		}
	}
	resetCacheSize := func(cachesize int64) {
		g := testGroup
		g.mainCache = cache{}
		g.hotCache = cache{}
		g.cacheBytes = cachesize
	}
	resetCacheSize(1 << 20)
	run("bbb", 200)
	run("ggg", 200)
	resetCacheSize(0)
	peersList[0] = nil
	run("one_peer_done", 200)
	peersList[0] = peer0
	peer0.fail = true
	run("peer0_fail", 200)
}
func TesttruncBytesSink(t *testing.T) {
	var buf [100]byte
	s := buf[:]
	if err := StringGroup.Get(dummyCtx, "wxnzb", TruncBytesSink(&s)); err != nil {
		t.Fatal(err)
	}
	if want := "ECHO:wxnzb"; string(s) != want {
		t.Errorf("want %q,got %q", want, s)
	}
	s = buf[:6]
	if err := StringGroup.Get(dummyCtx, "wxnzb", TruncBytesSink(&s)); err != nil {
		t.Fatal(err)
	}
	if want := "ECHO:w"; string(s) != want {
		t.Errorf("want %q,got %q", want, s)
	}
}

// // 确保 dst、inBytes 和 view.b 之间 不共享内存
// func TestallocBytesSink(t *testing.T) {
// 	var dst []byte
// 	sink := AllocBytesSink(&dst)
// 	inBytes := []byte("hello wx!")
// 	sink.SetBytes(inBytes)
// 	if want := "hello wx!"; string(dst) != want {
// 		t.Errorf("want %q,got %q", want, dst)
// 	}
// 	v, err := sink.view()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if &inBytes[0] == &v.b[0] {
// 		t.Error("inBytes and dst share memory")
// 	}
// 	if &inBytes[0] == &dst[0] {
// 		t.Error("inBytes and dst share memory")
// 	}
// 	if &v.b[0] == &dst[0] {
// 		t.Error("inBytes and dst share memory")
// 	}
// }

// // 证 singleflight 无法去重的情况下 groupcache 是否还能正确缓存
// type orderedFlight struct {
// 	mu     sync.Mutex
// 	stage1 chan bool
// 	stage2 chan bool
// 	org    *singleflight.Group
// }

// func (o *orderedFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
// 	<-o.stage1
// 	<-o.stage2
// 	mu.Lock()
// 	defer mu.Unlock()
// 	return o.org.Do(key, fn)
// }
// func testNodup(t *testing.T) {

// }
