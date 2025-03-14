## 1
- pb "groupcache/groupcachepb"
- 相当于define,给groupcache/groupcachepb重新起名pb
## 2
- if bytes,ok:=tt.b.([]byte);ok{
-			if got:=va.EqualBytes(bytes);got!=tt.want{
- if _,ok:=tt.b.([]byte);ok{
-			if got:=va.EqualBytes(tt.b);got!=tt.want{
- 我想的是，要是是ok了，那么tt.b就是[]byte类型，那么第二种也可以，其实不是，因为tt.b要是是interface{}类型，强转成[]byte类型存金bytes，ok也是true,但是tt.b还是interface{}类型
## 3
- *b.dst = []byte(v)和*b.dst = v.([]byte)有什么区别吗
- 第一种
- 这里 v 必须是 string 类型，因为 []byte(v) 只能转换 string。这个操作 会创建 v 的一个新副本，b.dst 指向的是这个新分配的 []byte
- 第二种
- 这里 v 必须是 interface{} 类型，并且其具体值必须是 []byte 类型。不会创建新副本，只是让 b.dst 指向 v 内部的 []byte 数据