package groupcache

import (
	"bytes"
	"io"
	"strings"
)

// 在这个文件中，这个结构体中都是b的优先级高
type ByteView struct {
	b []byte //先用这个，如果这个没有，就用下面这个
	s string
}

// 之前用的都是*ByteView，现在用ByteView,为啥
// Len()
func (v ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}
	return len(v.s)
}

// ByteSlice()
func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		//return v.b为啥不这样
		//如果直接返回 v.b，调用者可以修改返回的 []byte，从而影响 ByteView 内部数据，破坏不可变性
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

// String()
func (v ByteView) String() string {
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
func (v ByteView) At(i int) byte {
	if v.b != nil {
		return v.b[i]
	}
	//v.s时string，但是这样就直接时byte类型吗
	return v.s[i]
}

// Slice()
func (v ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:to]}
	}
	return ByteView{s: v.s[from:to]}
}

// SliceFrom()
func (v ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:]}
	}
	return ByteView{s: v.s[from:]}
}
func (v ByteView) Equal(b ByteView) bool {
	if b.b == nil {
		return v.EqualString(b.s)
	}
	return v.EqualBytes(b.b)
}
func (v ByteView) EqualString(s string) bool {
	if v.b == nil {
		return v.s == s
	}
	l := v.Len()
	if l != len(s) {
		return false
	}
	for i, vb := range v.b {
		if vb != s[i] {
			return false
		}
	}
	return true
}
func (v ByteView) EqualBytes(b []byte) bool {
	if v.b != nil {
		return bytes.Equal(v.b, b)
	}
	l := v.Len()
	if l != len(b) {
		return false
	}
	for i, bb := range b {
		if bb != v.s[i] {
			return false
		}
	}
	return true
}

// 下面这里还需思考
func (v ByteView) Reader() io.ReadSeeker {
	if v.b != nil {
		return bytes.NewReader(v.b)
	}
	return strings.NewReader((v.s))
}

// Copy(dest []byte)int方法的作用是将ByteView内部存储的数据拷贝到dest数组中，并返回实际拷贝的字节数
// copy是go语言中的内值函数，不需要包含在任何包中就可以用
func (v ByteView) Copy(dest []byte) int {
	if v.b != nil {
		return copy(dest, v.b)
	}
	return copy(dest, v.s)
}

// 将ByteView内部存储的数据拷贝到dest数组中，并返回实际拷贝的字节数,这是从off开始
// 这里必须将off 转成int64类型嗯的int类型不行；原因test测试中：
// r, err := ioutil.ReadAll(io.NewSectionReader(v, 0, int64(len(s))))，第一个参数是io.ReaderAt接口，所以ByteView需要实现这个接口
func (v ByteView) ReadAt(p []byte, off int64) (int, error) {
	n := v.SliceFrom(int(off)).Copy(p)
	return n, nil
}

// 将ByteView内部存储的数据拷贝到dest数组中，并返回实际拷贝的字节数
func (v ByteView) WriteTo(w io.Writer) (int, error) {
	if v.b != nil {
		return w.Write(v.b)
	}
	return io.WriteString(w, v.s)
}
