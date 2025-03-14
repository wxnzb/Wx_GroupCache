package groupcache

import (
	pb "groupcache/groupcachepb"
	"groupcache/lru"
	"groupcache/singleflight"
	"strconv"
	"sync"
	"sync/atomic"
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
	Bytes int64
	Hits  int64
	Gets  int64
	Evict int64
	Items int64
}

// 不用New吗
func (c *cache) Status() CacheStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStatus{
		Bytes: c.nbytes,
		Hits:  c.nhits,
		Gets:  c.ngets,
		Evict: c.nevict,
		Items: c.LockItems(),
	}
}
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//记得强制类型转换
	c.nbytes += int64(len(key) + value.Len())
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
func (c *cache) Bytes() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nbytes
}

// 获取cache里面lru里面list的长度
func (c *cache) items() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LockItems()
}
func (c *cache) LockItems() int64 {
	//我真的感觉这种判断真的能用上吗？
	if c.ca == nil {
		return 0
	}
	return int64(c.ca.Len())
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////
type Group struct {
	name      string      //这个库的名字
	getter    Getter      //都找不到时，要通过回调函数在最初数据库中获得数据
	mainCache cache       //里面存放的kv,在给Cache封装一层
	loader    flightGroup //实现多次申请一次调用
	peers     PeerPicker
	///////////////
	status     Status
	hotCache   cache
	cacheBytes int64
}
type AtomicInt int64
type Status struct {
	Gets             AtomicInt //获取次数
	CacheHits        AtomicInt //缓存命中次数
	Loads            AtomicInt //加载次数
	PeersLoads       AtomicInt //从其他节点加载次数
	PeersErrors      AtomicInt //从其他节点加载失败次数,为什么会加载失败
	LocalLoads       AtomicInt //从本地加载次数
	LocalLoadsErrors AtomicInt //从本地加载失败次数
}

// 把关于Status里面的实现了
// 给i加上n
func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

// 获取i的值
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// 将i转换为字符串
func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}

type flightGroup interface {
	Do(key string, fn func() (interface{}, error)) (interface{}, error)
}

type Getter interface {
	//这里要将接收到的value直接传到参数Sink中，Sink还没有定义
	Get(key string, dest Sink) error
}
type GetterFunc func(key string, dest Sink) error

func (f GetterFunc) Get(key string, dest Sink) error {
	return f(key, dest)
}

var (
	mu sync.RWMutex
	//下面这是新加的
	initPeerServerOnce sync.Once
	initPeerServer     func()
	newGroupHook       func(*Group)
)

// ////////////////////////////////////////////
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
	if newGroupHook != nil {
		panic("RegisterNewGroupHook called more than once")
	}
	newGroupHook = fn
}

// /////////////////////////////////////////////
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
func (g *Group) Getname() string {
	return g.name
}
func (g *Group) InitPeers() {
	g.peers = GetPeers(g.name)
}
func (g *Group) Get(key string, dest Sink) error {
	initPeerServerOnce.Do(g.InitPeers)
	g.status.Gets.Add(1)
	if dest == nil {
		panic("nil dest")
	}
	if v, ok := g.LookUpCache(key); ok {
		//缓存命中
		g.status.CacheHits.Add(1)
		//这个函数还没有实现
		return SetSinkValue(dest, v)
	}
	//缓存未命中
	//加入destPopulated看是否被填充
	destpopulated := false
	v, destpopulated, err := g.Load(key, dest)
	if err != nil {
		return err
	}
	if destpopulated {
		return nil
	}
	return SetSinkValue(dest, v)
}

// 在本地查找缓存
func (g *Group) LookUpCache(key string) (v ByteView, ok bool) {
	v, ok = g.mainCache.get(key)
	if ok {
		return
	}
	v, ok = g.hotCache.get(key)
	return
}
func (g *Group) Load(key string, dest Sink) (v ByteView, destpopulated bool, err error) {
	g.status.Loads.Add(1)
	//实现独立性
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		//先在远程节点找，要是找不到，就在本地通过回调函数加载并存储
		//这里！！！
		if v, ok := g.LookUpCache(key); ok {
			//缓存命中
			g.status.CacheHits.Add(1)
			return v, nil
		}
		//远程
		if peer, ok := g.peers.PickPeer(key); ok {
			if v, err := g.getFromPeer(peer, key); err == nil {
				g.status.PeersLoads.Add(1)
				return v, nil
			}
			g.status.PeersErrors.Add(1)
		}
		//本地回调加载
		v, err = g.getLocally(key, dest)
		if err != nil {
			g.status.LocalLoadsErrors.Add(1)
			return nil, err
		}
		g.status.LocalLoads.Add(1)
		destpopulated = true
		g.populateCache(key, v, &g.mainCache)
		return v, nil

	})
	if err == nil {
		v = viewi.(ByteView)
		return
	}
	return
}
func (g *Group) populateCache(key string, v ByteView, cache *cache) {
	cache.add(key, v)
	for {
		mainBytes := g.mainCache.Bytes()
		hotBytes := g.hotCache.Bytes()
		if mainBytes+hotBytes < g.cacheBytes {
			return
		}
		victm := &g.mainCache
		if hotBytes > mainBytes/8 {
			victm = &g.hotCache
		}
		victm.removeOldest()
	}
}
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.GetRequest{
		Group: &g.name,
		Key:   &key,
	}
	res := &pb.GetResponse{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: res.Value}
	//这里还要加：通过随机数控制是否将数据存入 hotCache，且每次只有 10% 的概率会触发 populateCache 方法将数据放入 hotCache，这个之后在加
	return value, nil
}
func (g *Group) getLocally(key string, dest Sink) (ByteView, error) {
	err := g.getter.Get(key, dest)
	if err != nil {
		return ByteView{}, err
	}
	return dest.view()
}
