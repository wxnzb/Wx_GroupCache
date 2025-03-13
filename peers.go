package groupcache

import (
	pb "groupcache/groupcachepb"
)

type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}
type PeerGetter interface {
	Get(*pb.GetRequest, *pb.GetResponse) error
}
type NoPeers struct{}

func (NoPeers) PickPeer(key string) (peer PeerGetter, ok bool) {
	return
}

var PortPicker func(groupname string) PeerPicker

// 两种方式
// 节点选择器
// 现在我想不通第一种不传入groupname是怎样选的
func RegisterPeerPicker(fn func() PeerPicker) {
	if PortPicker != nil {
		panic("RegisterPeerPicker called more than once")
	}
	//_ string这是啥
	PortPicker = func(_ string) PeerPicker {
		return fn()
	}
}
func RegisterGroupPeerPicker(fn func(groupname string) PeerPicker) {
	if PortPicker != nil {
		panic("RegisterGroupPeerPicker called more than once")
	}
	PortPicker = fn
}
func GetPeers(nameGroup string) PeerPicker {
	if PortPicker == nil {
		return NoPeers{}
	}
	fn := PortPicker(nameGroup)

	//return fn好像还不能这样返回
	if fn == nil {
		fn = NoPeers{}
	}
	return fn
}
