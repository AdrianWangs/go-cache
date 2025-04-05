// Package peers defines interfaces for peer selection and communication
package peers

import (
	pb "github.com/AdrianWangs/go-cache/proto/cache_server"
)

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	// PickPeer returns the peer that owns the specific key and a boolean
	// indicating whether a peer was found.
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	// Get returns the value for the specified group and key.
	Get(group string, key string) ([]byte, error)

	// GetByProto returns the value for the specified request using protobuf.
	GetByProto(req *pb.Request, resp *pb.Response) error
}
