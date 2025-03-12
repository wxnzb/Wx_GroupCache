package groupcache

import (
	"groupcache/lru"
	"groupcache/singleflight"
	"sync"
	//"sync/atomic"
)

// 先实现cache把
type cache struct {
	ca *lru.Cache
	mu sync.RWMutex
	//下面这是新加的
	nbytes       int64 //kv占得字节
	nhits, ngets int64 //命中次数，获取次数
	nevict       int64 //淘汰次数
}
type CacheStatus struct {
	Nbytes int64
	Nhits  int64
	Ngets  int64
	Nevict int64
}

// 不用New吗
func (c *cache) Status() CacheStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStatus{
		Nbytes: c.nbytes,
		Nhits:  c.nhits,
		Ngets:  c.ngets,
		Nevict: c.nevict,
	}
}
func (c *cache) add(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//记得强制类型转换
	c.nbytes += int64(len(key) + len(value))
	c.ca.Add(key, value)
}
func (c *cache) removeOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	//c.nevict,c.bytes我感觉都要变把
	c.ca.RemoveOldest()

}
func (c *cache) get(key string) (value ByteView, ok bool) {
	//感觉读的时候不需要锁
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ngets++
	if v, ok := c.ca.Get(key); ok {
		c.nhits++
		//ByteView是结构体，这也能强制类型转换吗
		return v.(ByteView), true
	}
	return
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////
type Group struct {
	name      string      //这个库的名字
	getter    Getter      //都找不到时，要通过回调函数在最初数据库中获得数据
	mainCache cache       //里面存放的kv,在给Cache封装一层
	loader    flightGroup //实现多次申请一次调用
	peers     PeerPicker
	status    Status
}
type AtomicInt int64
type Status struct {
	Gets AtomicInt
}

type flightGroup interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error)
}

type Getter interface {
	//这里要将接收到的value直接传到参数里面，这里还没有写
	Get(key string) error
}
type GetterFunc func(key string) error

func (f GetterFunc) Get(key string) error {
	return f(key)
}

var (
	mu sync.RWMutex
	//下面这是新加的
	initPeerServerOnce sync.Once
	initPeerServer     func()
	newGroupHook       func(*Group)
)

func RegisterInitPeerServer(fn func()) {
	if initPeerServer != nil {
		panic("RegisterInitPeerServer called more than once")
	}
	initPeerServer = fn
}
func callInitPeerServer() {
	if initPeerServer != nil {
		initPeerServer()
	}
}
func RegisterNewGroupHook(fn func(*Group)) {
	if newGroup != nil {
		panic("RegisterNewGroupHook called more than once")
	}
	newGroupHook = fn
}

var groups = make(map[string]*Group)

func GetGroup(groupname string) *Group {
	//我想不通，这里为什么要加锁
	mu.Lock()
	defer mu.Unlock()
	return groups[groupname]
}
func NewGroup(name string, getter Getter) *Group {
	return newGroup(name, getter, nil)
}
func newGroup(name string, getter Getter, peers PeerPicker) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	//这句作用是啥
	initPeerServerOnce.Do(callInitPeerServer)
	g := &Group{
		name:   name,
		getter: getter,
		loader: &singleflight.Group{},
		peers:  peers,
	}
	//还有这句
	if f := newGroupHook; f != nil {
		f(g)
	}
	groups[name] = g
	return g
}
