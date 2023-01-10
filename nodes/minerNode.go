package nodes

import (
	"io"
	"log"
	"net"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/p2p"
	"simple-blockchain-go/pow"
)

type MinerNode struct {
	Node
	latestInfo blocks.BlockInfo
	offerer    p2p.NodeId
}

func NewMinerNode(port string) *MinerNode {
	m := MinerNode{
		Node: Node{
			id:      p2p.NewNodeId(port, p2p.MINER_NODE),
			version: 1,
		},
		latestInfo: blocks.BlockInfo{},
	}
	m.AppendPeer(p2p.DefaultKnownNode(port, p2p.MINER_NODE))
	return &m
}

func (m *MinerNode) Run() error {
	listener, err := net.Listen(p2p.TCP, string(m.id.Ip))
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("miner node is listening at %s", m.id.Ip)

	err = m.broadcastJoin()
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go m.handleConnection(conn)
	}
}

func (m *MinerNode) mine(block *blocks.Block) error {
	miner := pow.NewProofOfWork(block)
	nonce, hash, err := miner.Run()
	if err != nil {
		return err
	}

	block.Hash = hash[:]
	block.Nonce = nonce

	log.Printf("broadcasting new block...")
	return m.sendRegisterBlock(block)
}

func (m *MinerNode) handleConnection(conn net.Conn) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	msgKind := p2p.MessageKind(request[0])
	log.Printf("received msg '%s'\n", msgKind.ToString())

	switch msgKind {
	case p2p.ADDRESS_MSG:
		err = m.handleAddress(request[1:])
	case p2p.BLOCKCHAIN_INFO_MSG:
		err = m.handleBlockchainInfo(request[1:])
	case p2p.OFFER_BLOCK_MSG:
		err = m.handleOfferBlock(request[1:])
	case p2p.ACCEPTED_BLOCK_MSG:
		err = m.handleAcceptedBlock(request[1:])
	case p2p.REWARD_MSG:
		log.Printf("\n\n    this is the miner (^_^)    \n\n")
	default:
		log.Println("unknown message skipping...")
	}
	if err != nil {
		log.Panic(err)
	}
}

func (m *MinerNode) handleOfferBlock(raw []byte) error {
	msg, err := common.Decode[p2p.OfferBlockMsg](raw)
	if err != nil {
		return err
	}
	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}

	m.offerer = msg.From
	return m.mine(&msg.Block)
}

func (m *MinerNode) handleBlockchainInfo(raw []byte) error {
	msg, err := common.Decode[p2p.BlockchainInfoMsg](raw)
	if err != nil {
		return err
	}
	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}

	m.latestInfo.Height = msg.Height + 1
	m.latestInfo.Difficulty = msg.Difficulty
	m.latestInfo.PreviousBlockHash = msg.PreviousBlockHash
	log.Printf(
		"received blockchain info\n next height: %d\n difficulty: %d\n latest: %x\n",
		m.latestInfo.Height, m.latestInfo.Difficulty, m.latestInfo.PreviousBlockHash,
	)
	return nil
}

func (m *MinerNode) handleAcceptedBlock(raw []byte) error {
	msg, err := common.Decode[p2p.AcceptedBlockMsg](raw)
	if err != nil {
		return err
	}
	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}

	m.latestInfo.Height = msg.Block.Height + 1
	m.latestInfo.Difficulty = msg.Difficulty
	m.latestInfo.PreviousBlockHash = msg.Block.PreviousBlockHash
	log.Printf(
		"received accepted block\n next height: %d\n difficulty: %d\n latest: %x\n",
		m.latestInfo.Height, m.latestInfo.Difficulty, m.latestInfo.PreviousBlockHash,
	)
	return nil
}

func (m *MinerNode) sendRegisterBlock(block *blocks.Block) error {
	msg := p2p.RegisterBlockMsg{
		From:  m.id,
		Block: *block,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}

	payload := p2p.REGISTER_BLOCK_MSG.MakePayload(enc)
	return m.send(m.offerer, payload)
}
