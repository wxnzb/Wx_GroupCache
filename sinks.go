package groupcache

import "google.golang.org/protobuf/proto"

// 结构体他一般除了所必需的byteiew之外还有一个dst,他的作用是什么
// dst 存储最终数据，外部可以访问它。
// ByteView 提供一个只读视图，供 view() 方法使用。
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

type Sink interface {
	SetString(s string) error
	SetBytes(v []byte) error
	view() (ByteView, error)
	SetProto(m proto.Message) error
}

func SetSinkValue(s Sink, v ByteView) error {
	type ViewSink interface {
		SetView(v ByteView) error
	}
	//这里vs具体是啥
	if vs, ok := s.(ViewSink); ok {
		return vs.SetView(v)
	}
	if v.b != nil {
		return s.SetBytes(v.b)
	}
	return s.SetString(v.s)
}

// 1
type stringSink struct {
	sp *string
	v  ByteView
}

func (s *stringSink) StringSink(sp *string) Sink {
	return &stringSink{sp: sp}
}
func (s *stringSink) SetString(v string) error {
	//这里为啥要把Byteview里面的b设置成nil，这里等于就没有设置ByteView的b阿，那为什么还要有SetBytes函数???
	s.v.b = nil
	*s.sp = v
	s.v.s = v
	return nil
}
func (s *stringSink) SetBytes(v []byte) error {
	return s.SetString(string(v))
}

// //我感觉要统一的话这个应该比较好把
//
//	func (s *StringSink)SetBytes(v []byte)error{
//		s.v.s=""
//		s.v.b=cloneBytes(v)
//		return nil
//	}
func (s *stringSink) view() (ByteView, error) {
	return s.v, nil
}
func (s *stringSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	*s.sp = string(b)
	//s.v.b = cloneBytes(b)这里为啥不用clone,好像这个函数的实现，都不用进行clone
	s.v.b = b
	return nil
}

// 2
type byteViewSink struct {
	dst *ByteView
}

func (b *byteViewSink) ByteViewSink(dst *ByteView) Sink {
	return &byteViewSink{dst: dst}
}
func (b *byteViewSink) SetString(v string) error {
	*b.dst = ByteView{s: v}
	return nil
}
func (b *byteViewSink) SetBytes(v []byte) error {
	*b.dst = ByteView{b: cloneBytes(v)}
	return nil
}
func (b *byteViewSink) view() (ByteView, error) {
	return *b.dst, nil
}

// 他这还多实现了一个set方法，是setBytes和setString的结合版
func (b *byteViewSink) SetView(v ByteView) error {
	*b.dst = v
	return nil
}
func (b *byteViewSink) SetProto(m proto.Message) error {
	v, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	//*b.dst = ByteView{b: cloneBytes(v)}这里为啥不用clone
	*b.dst = ByteView{b: v}
	return nil
}

// 3
type allocBytesSink struct {
	dst *[]byte
	v   ByteView
}

func (b *allocBytesSink) AllocBytesSink(dst *[]byte) Sink {
	return &allocBytesSink{dst: dst}
}
func (b *allocBytesSink) SetString(v string) error {
	*b.dst = []byte(v)
	b.v.s = v
	b.v.b = nil
	return nil
}

// 记住！！他不能改变本身，所以一定要创建副本
func (b *allocBytesSink) SetBytes(v []byte) error {
	*b.dst = cloneBytes(v)
	b.v.b = cloneBytes(v)
	b.v.s = ""
	return nil
}
func (b *allocBytesSink) view() (ByteView, error) {
	return b.v, nil
}
func (b *allocBytesSink) SetProto(m proto.Message) error {
	v, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	*b.dst = cloneBytes(v)
	b.v.b = v
	b.v.s = ""
	return nil
}

// 这个也实现了一个set方法
func (b *allocBytesSink) SetView(v ByteView) error {
	//这个应该也可以：*d.dst=[]byte(v.s)//这个就自动给string创建了一个byte[]的副本
	*b.dst = cloneBytes(v.b)
	b.v = v
	return nil
}

// 4
// 这里和上面不同的是多了个*[]byte,所以在给他赋值的时候，有这种情况，就是他是[]byte{0,0,0,0,0,0,0,0},但是你想把"hello"小于他的赋给他，那么它后面多余的0就要去掉，但是我不清楚[]byte,make是在哪里？？
type truncBytesSink struct {
	dst *[]byte
	v   ByteView
}

func (b *truncBytesSink) TruncBytesSink(dst *[]byte) Sink {
	return &truncBytesSink{dst: dst}
}
func (b *truncBytesSink) SetString(v string) error {
	n := copy(*b.dst, v)
	if n < len(*b.dst) {
		*b.dst = (*b.dst)[:n]
	}
	b.v.s = v
	b.v.b = nil
	return nil
}
func (b *truncBytesSink) SetBytes(v []byte) error {
	n := copy(*b.dst, v)
	if n < len(*b.dst) {
		*b.dst = (*b.dst)[:n]
	}
	b.v.b = cloneBytes(v)
	b.v.s = ""
	return nil
}
func (b *truncBytesSink) view() (ByteView, error) {
	return b.v, nil
}
func (b *truncBytesSink) SetProto(m proto.Message) error {
	v, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	n := copy(*b.dst, v)
	if n < len(*b.dst) {
		*b.dst = (*b.dst)[:n]
	}
	b.v.b = cloneBytes(v)
	return nil
}
