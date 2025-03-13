package groupcache

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

type Sink interface {
	SetString(s string) error
	SetBytes(v []byte) error
	view() (ByteView, error)
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
type StringSink struct {
	sp *string
	v  ByteView
}

func (s *StringSink) StringSink(sp *string) Sink {
	return &StringSink{sp: sp}
}
func (s *StringSink) SetString(v string) error {
	//这里为啥要把Byteview里面的b设置成nil，这里等于就没有设置ByteView的b阿，那为什么还要有SetBytes函数???
	s.v.b = nil
	*s.sp = v
	s.v.s = v
	return nil
}
func (s *StringSink) SetBytes(v []byte) error {
	return s.SetString(string(v))
}
func (s *StringSink) view() (ByteView, error) {
	return s.v, nil
}

// 2
type ByteViewSink struct {
	dst *ByteView
}

func (b *ByteViewSink) ByteViewSink(dst *ByteView) Sink {
	return &ByteViewSink{dst: dst}
}
func (b *ByteViewSink) SetString(v string) error {
	*b.dst = ByteView{s: v}
	return nil
}
func (b *ByteViewSink) SetBytes(v []byte) error {
	*b.dst = ByteView{b: cloneBytes(v)}
	return nil
}
func (b *ByteViewSink) view() (ByteView, error) {
	return *b.dst, nil
}

// 他这还多实现了一个set方法，是setBytes和setString的结合版
func (b *ByteViewSink) SetView(v ByteView) error {
	*b.dst = v
	return nil
}
