package nodes

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"simple-blockchain-go/common"
	"simple-blockchain-go/p2p"
	"strings"
)

type Node struct {
	id      p2p.NodeId
	version byte
	KnownNodes
}

func isRendezvous(node p2p.NodeId) bool {
	return strings.Compare(node.Ip, p2p.RENDEZVOUS) == 0
}

func (n *Node) isSelf(node p2p.NodeId) bool {
	return strings.Compare(node.Ip, n.id.Ip) == 0
}

func (n *Node) broadcastJoin() error {
	if isRendezvous(n.id) {
		log.Println("listening rendezvous point...")
		return nil
	}

	msg := p2p.JoinMsg{
		From:    n.id.Ip,
		Version: n.version,
		Kind:    n.id.Kind,
	}
	ser, err := common.Encode(msg)
	if err != nil {
		return err
	}

	log.Println("broadcasting join...")
	payload := p2p.JOIN_MSG.MakePayload(ser)
	return n.broadcast(payload)
}

func (n *Node) send(to p2p.NodeId, data []byte) error {
	idx := n.PeerIndex(to)
	if idx < 0 {
		return errors.New("node is not known")
	}

	conn, err := net.Dial(p2p.TCP, string(to.Ip))
	if err != nil {
		log.Printf("%s is not available\n", to.Ip)
		n.RemovePeer(idx)
		return nil
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	return err
}

func (n *Node) broadcast(data []byte) error {
	for _, node := range n.peers {
		if n.isSelf(node) {
			continue
		}

		err := n.send(node, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) handleAddress(raw []byte) error {
	msg, err := common.Decode[p2p.AddressMsg](raw)
	if err != nil {
		return err
	}

	if msg.From.Kind == p2p.WALLET_NODE {
		return nil
	}

	peer := common.FindAll(msg.NodeList, func(node p2p.NodeId) bool {
		return !isRendezvous(node) && !n.isSelf(node) &&
			(node.Kind == p2p.EXECUTER_NODE ||
				node.Kind == p2p.MINER_NODE)

	})
	log.Printf("recieved %d peer\n", len(peer))
	n.AppendPeer(peer...)
	return nil
}
