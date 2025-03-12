package groupcache

import (
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

func Test_ByteView(t *testing.T) {
	for _, s := range []string{"", "wx", "WWXx"} {
		for _, v := range []ByteView{of([]byte(s)), of(s)} {
			name := fmt.Sprintf("string:%s,view:%v", s, v)
			//Len()
			if v.Len() != len(s) {
				t.Errorf("%s:Len=%d,want %d", name, v.Len(), len(s))
			}
			//String()
			if v.String() != s {
				t.Errorf("%s:String=%s,want %s", name, v.String(), s)
			}
			//Copy()
			var longBuf [3]byte
			if n := v.Copy(longBuf[:]); n != len(s) {
				t.Errorf("%s:Copy=%d;want %d", name, n, len(s))
			}
			var shortBuf [1]byte
			if n := v.Copy(shortBuf[:]); n != min(len(s), 1) {
				t.Errorf("%s:Copy=%d;want %d", name, n, min(len(s), 1))
			}
			//Reader()
			if r, err := ioutil.ReadAll(v.Reader()); err != nil || string(r) != s {
				t.Errorf("%s:Read=%v;want %s", name, r, s)
			}
			//ReadAt()
			if r, err := ioutil.ReadAll(io.NewSectionReader(v, 0, len(s))); err != nil || string(s) != len(s) {
				t.Errorf("%s:ReadAt=%v;want %s", name, r, s)
			}
		}
	}
}

//	func of( x interface{})ByteView{
//	     if bytes,ok:=x.([]byte),ok{
//	       return ByteView{b:bytes}
//		 }
//		 return ByteView{s:x.(string)}
//	}
func of(x interface{}) ByteView {
	if bytes, ok := x.([]byte); ok {
		return ByteView{b: bytes}
	}
	return ByteView{s: x.(string)}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
