package peers

// PeerPicker 用于选择一个节点，并返回从该节点获取数据的PeerGetter
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 用于从节点获取数据
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
