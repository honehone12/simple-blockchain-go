package p2p

import (
	"fmt"
	"log"
	"strings"
)

const (
	TCP         = "tcp"
	ADDRESS_FMT = "localhost:%s"
	RENDEZVOUS  = "localhost:3000"
)

type NodeKind byte

const (
	EXECUTER_NODE NodeKind = iota + 1
	MINER_NODE
	WALLET_NODE
)

func (nk NodeKind) ToString() string {
	switch nk {
	case EXECUTER_NODE:
		return "executer node"
	case MINER_NODE:
		return "miner node"
	case WALLET_NODE:
		return "wallet node"
	default:
		log.Panicf("unknown value %d", nk)
		return ""
	}
}

type NodeId struct {
	Ip   string
	Kind NodeKind
}

func NewNodeId(port string, kind NodeKind) NodeId {
	return NodeId{
		Ip:   fmt.Sprintf(ADDRESS_FMT, port),
		Kind: kind,
	}
}

func DefaultKnownNode(port string, kind NodeKind) NodeId {
	return NodeId{
		Ip:   RENDEZVOUS,
		Kind: EXECUTER_NODE,
	}
}

func IsSameIp(nodeA NodeId, nodeB NodeId) bool {
	return strings.Compare(nodeA.Ip, nodeB.Ip) == 0
}
