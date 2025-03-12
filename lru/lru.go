package lru

import (
	"container/list"
)

// 定义Cache结构体
type Cache struct {
	maxEntries int
	list       *list.List
	cache      map[interface{}]*list.Element
}

// 定义Entry结构体
type Entry struct {
	key   interface{}
	value interface{}
}

// New()
func New(maxEntries int) *Cache {
	return &Cache{
		maxEntries: maxEntries,
		list:       list.New(),
		cache:      make(map[interface{}]*list.Element),
		//这里还没有补充OnEvicted，将废除掉的kv加进去
	}
}

// Add()
func (c *Cache) Add(key, value interface{}) {
	if ele, ok := c.cache[key]; ok {
		c.list.MoveToFront(ele)
		ele.Value.(*Entry).value = value
		return
	}
	ele := c.list.PushFront(&Entry{key, value})
	c.cache[key] = ele
	if c.maxEntries != 0 && c.list.Len() > c.maxEntries {
		c.RemoveOldest()
	}
}

// RemoveOldest()
func (c *Cache) RemoveOldest() {
	ele := c.list.Back()
	if ele != nil {
		//这是固定用法
		c.list.Remove(ele)
		delete(c.cache, ele.Value.(*Entry).key)
	}
}

// Get()
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.list.MoveToFront(ele)
		return ele.Value.(*Entry).value, true
	}
	return nil, false
}
