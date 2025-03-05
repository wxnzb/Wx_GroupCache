package groupcache

type ByteView struct {
	b []byte
	s string
}

// Len()
func (v *ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}
	return len(v.s)
}

// ByteSlice()
func (v *ByteView) ByteSlice() []byte {
	if v.b != nil {
		//return v.b为啥不这样
		//如果直接返回 v.b，调用者可以修改返回的 []byte，从而影响 ByteView 内部数据，破坏不可变性
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

// String()
func (v *ByteView) String() string {
	if v.b != nil {
		return string(v.b)
	}
	return v.s
}

// 为啥我感觉这个这样实现更好
//
//	func (v *ByteView)String()string{
//		if v.s!=""{
//			return v.s
//		}
//		return string(v.b)
//	}
//
// At()
func (v *ByteView) At(i int) byte {
	if v.b != nil {
		return v.b[i]
	}
	//v.s时string
	return v.s[i]
}

// Slice()
func (v *ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:to]}
	}
	return ByteView{s: v.s[from:to]}
}

// SliceFrom()
func (v *ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:]}
	}
	return ByteView{s: v.s[from:]}
}
