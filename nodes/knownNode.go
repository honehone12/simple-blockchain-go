package nodes

import (
	"simple-blockchain-go/p2p"
	"sync"

	"golang.org/x/exp/slices"
)

type KnownNodes struct {
	sync.Mutex
	peers []p2p.NodeId
}

func (kn *KnownNodes) AppendPeer(id ...p2p.NodeId) {
	kn.Lock()
	defer kn.Unlock()
	kn.peers = append(kn.peers, id...)
}

func (kn *KnownNodes) RemovePeer(idx int) {
	if idx < 0 || idx > len(kn.peers) {
		return
	}
	kn.Lock()
	defer kn.Unlock()
	kn.peers = slices.Delete(kn.peers, idx, idx+1)
}

func (kn *KnownNodes) GetPeer(idx int) (p2p.NodeId, bool) {
	if idx < 0 || idx > len(kn.peers) {
		return p2p.NodeId{}, false
	}
	return kn.peers[idx], true
}

func (kn *KnownNodes) PeerLen() int {
	return len(kn.peers)
}

func (kn *KnownNodes) PeerIndex(id p2p.NodeId) int {
	return slices.Index(kn.peers, id)
}
