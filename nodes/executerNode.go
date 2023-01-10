package nodes

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"simple-blockchain-go/accounts"
	"simple-blockchain-go/blockchain"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/epoch"
	"simple-blockchain-go/memory"
	"simple-blockchain-go/p2p"
	"simple-blockchain-go/transactions"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

const (
	MINE_THRESHOLD_MAX = 5100
	MINE_THRESHOLD_MIN = 4900
)

type ExecuterNode struct {
	Node
	*blockchain.Blockchain
	txPool         *memory.TxPool
	epoch          *epoch.Epoch
	isSyncing      bool
	offeredTime    int64
	offeredTxHash  []byte
	airdropAccount []byte
}

func NewExecuterNode(port string) (*ExecuterNode, error) {
	bc, err := blockchain.NewBlockchain(port)
	s := ExecuterNode{
		Node: Node{
			id:      p2p.NewNodeId(port, p2p.EXECUTER_NODE),
			version: 1,
		},
		Blockchain:  bc,
		txPool:      memory.NewTransactionPool(),
		epoch:       nil,
		offeredTime: time.Now().UnixMilli(),
	}
	s.AppendPeer(p2p.DefaultKnownNode(port, p2p.EXECUTER_NODE))
	return &s, err
}

func (e *ExecuterNode) Run() error {
	listener, err := net.Listen(p2p.TCP, string(e.id.Ip))
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("executer node is listening at %s", e.id.Ip)

	err = e.broadcastJoin()
	if err != nil {
		return err
	}

	e.epoch = epoch.NewEpoch(e.executionRoutine)
	go e.epoch.StartEpochRoutine()
	if isRendezvous(e.id) {
		e.retry()
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go e.handleConnection(conn)
	}
}

func (e *ExecuterNode) checkHealth() error {
	dbLatesHash, err := e.GetLatest()
	if err != nil {
		return err
	}
	dbHeight, err := e.GetHeight()
	if err != nil {
		return err
	}
	if !bytes.Equal(dbLatesHash, e.PreviousBlockHash) || dbHeight != e.Height {
		return fmt.Errorf(
			"database mismatch\n db: %d, %x\n cached: %d, %x",
			dbHeight, dbLatesHash, e.Height, e.PreviousBlockHash,
		)
	}
	return nil
}

func (e *ExecuterNode) handleConnection(conn net.Conn) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	msgKind := p2p.MessageKind(request[0])
	log.Printf("received msg '%s'\n", msgKind.ToString())

	switch msgKind {
	case p2p.JOIN_MSG:
		err = e.handleJoin(request[1:])
	case p2p.ACCOUNT_MSG:
		err = e.handleAccount(request[1:])
	case p2p.REGISTER_BLOCK_MSG:
		err = e.handleRegisterBlock(request[1:])
	case p2p.TX_MSG:
		err = e.handleTransaction(request[1:])
	case p2p.TX_POOL_MSG:
		err = e.handleTxPool(request[1:])
	case p2p.ADDRESS_MSG:
		err = e.handleAddress(request[1:])
	case p2p.BLOCKCHAIN_INFO_MSG:
		err = e.handleBlockchainInfo(request[1:])
	case p2p.SYNC_BLOCK_REQUEST_MSG:
		err = e.handleSyncBlockRequest(request[1:])
	case p2p.SYNC_BLOCK_RESPONSE_MSG:
		err = e.handleSyncBlockResponse(request[1:])
	case p2p.ACCEPTED_BLOCK_MSG:
		err = e.handleAcceptedBlock(request[1:])
	default:
		log.Println("unknown message skipping...")
	}
	if err != nil {
		log.Panic(err)
	}
}

func (e *ExecuterNode) handleBlockchainInfo(raw []byte) error {
	msg, err := common.Decode[p2p.BlockchainInfoMsg](raw)
	if err != nil {
		return err
	}
	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}

	log.Printf(
		"received blockchain info\n next height: %d\n difficulty: %d\n latest: %x\n",
		msg.Height, msg.Difficulty, msg.PreviousBlockHash,
	)
	if msg.Height > e.Height {
		e.isSyncing = true
		// now here means always download from RENDEZVOUS
		// this should be changed
		e.startDownloadBlocks(msg.From)
	}

	return nil
}

func (e *ExecuterNode) handleAccount(raw []byte) error {
	msg, err := common.Decode[p2p.AccountMsg](raw)
	if err != nil {
		return nil
	}
	content, err := common.Encode(msg.From)
	if err != nil {
		return err
	}
	ok := common.QuickVerify(msg.Signature, msg.PublicKey, content)
	if !ok {
		log.Println("received signature in msg is invalid")
		return nil
	}

	state, err := e.GetAccountStateSafe(msg.PublicKey)
	if err != nil {
		return err
	}
	return e.sendAccountInfo(msg.From, state, msg.PublicKey)
}

func (e *ExecuterNode) handleRegisterBlock(raw []byte) error {
	e.Lock()
	defer e.Unlock()

	msg, err := common.Decode[p2p.RegisterBlockMsg](raw)
	if err != nil {
		return err
	}

	hash, err := msg.Block.Bundle.HashTransactions()
	if err != nil {
		return err
	}
	if !bytes.Equal(e.offeredTxHash, hash) {
		log.Println("received block's transactions are not expected")
		return nil
	}

	if msg.Block.Difficulty != e.Difficulty {
		log.Printf(
			"received bloks's difficulty %d is invalid, expected: %d\n",
			msg.Block.Difficulty, e.Difficulty,
		)
		return nil
	}

	ok, err := e.VerifyBlock(&msg.Block)
	if err != nil {
		return err
	}
	if !ok {
		log.Println("received block is invalid")
		return nil
	}

	err = e.PutBlockWithCheck(&msg.Block)
	if err != nil {
		return err
	}

	accepetdTime := time.Now().UnixMilli()
	if accepetdTime-e.offeredTime > MINE_THRESHOLD_MAX {
		e.Difficulty--
	} else if accepetdTime-e.offeredTime < MINE_THRESHOLD_MIN {
		e.Difficulty++
	}

	// send reward only to accepted miner
	err = e.sendReward(msg.From)
	if err != nil {
		return err
	}

	// start new epoch routine
	e.epoch.C() <- true

	return e.broadcastAcceptedBlock(&msg.Block)
}

func (e *ExecuterNode) handleJoin(raw []byte) error {
	msg, err := common.Decode[p2p.JoinMsg](raw)
	if err != nil {
		return err
	}

	if !slices.ContainsFunc(e.peers, func(node p2p.NodeId) bool {
		return strings.Compare(node.Ip, msg.From) == 0
	}) {
		newFound := p2p.NodeId{
			Ip:   msg.From,
			Kind: msg.Kind,
		}
		e.AppendPeer(newFound)

		log.Printf(
			"found new peer at %s : %s\n",
			newFound.Ip,
			newFound.Kind.ToString(),
		)

		err = e.sendKnownPeer(newFound)
		if err != nil {
			return err
		}

		if msg.Kind == p2p.EXECUTER_NODE || msg.Kind == p2p.MINER_NODE {
			err = e.sendBlockchainInfo(newFound)
			if err != nil {
				return err
			}
		}

		if msg.Kind == p2p.EXECUTER_NODE {
			err = e.sendTxPool(newFound)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *ExecuterNode) handleAcceptedBlock(raw []byte) error {
	msg, err := common.Decode[p2p.AcceptedBlockMsg](raw)
	if err != nil {
		return err
	}

	if p2p.IsSameIp(msg.From, e.id) {
		return nil
	}

	log.Println("received new accepted block")
	log.Printf("including %d tx\n", len(msg.Block.Bundle.Transactions))
	err = e.syncBlockImpl(&msg.Block)
	if err != nil {
		return err
	}

	// should check all transactions are included in mempool
	// delete tx pool
	txKeys := memory.GetTxKeys(&msg.Block)
	e.txPool.BatchRemove(txKeys)

	// need not inform others ??
	return nil // e.broadcastAcceptedBlock(&msg.Block)
}

func (e *ExecuterNode) handleTxPool(raw []byte) error {
	msg, err := common.Decode[p2p.TxPoolMsg](raw)
	if err != nil {
		return err
	}

	for _, tx := range msg.Transactions {
		e.txPool.Append(&tx)
	}
	return nil
}

func (e *ExecuterNode) handleTransaction(raw []byte) error {
	msg, err := common.Decode[p2p.TransactionMsg](raw)
	if err != nil {
		return err
	}

	ok, err := msg.Transaction.Verify()
	if err != nil {
		return err
	}
	if !ok {
		log.Println("received transaction is invalid")
		return nil
	}

	executers := common.FindAll(e.peers, func(id p2p.NodeId) bool {
		return id.Kind == p2p.EXECUTER_NODE &&
			!e.isSelf(id) && !p2p.IsSameIp(msg.From, id)
	})
	for _, exc := range executers {
		e.sendTransaction(exc, &msg.Transaction)
	}

	if msg.From.Kind == p2p.WALLET_NODE {
		e.txPool.AppendOrOverwrite(&msg.Transaction)
	} else if msg.From.Kind == p2p.EXECUTER_NODE {
		e.txPool.Append(&msg.Transaction)
	}

	log.Printf(
		"received transaction, current pool size: %d\n",
		e.txPool.Len(),
	)
	return nil
}

func (e *ExecuterNode) sendTxPool(to p2p.NodeId) error {
	msg := p2p.TxPoolMsg{
		From:         e.id,
		Transactions: e.txPool.GetAll(),
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.TX_POOL_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) sendAccountInfo(
	to p2p.NodeId, info *accounts.AccountState, pubKey []byte,
) error {
	msg := p2p.AccountInfoMsg{
		From:      e.id,
		PublicKey: pubKey,
		Balance:   info.Balance,
		Nance:     info.Nonce,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.ACCOUNT_INFO_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) sendKnownPeer(to p2p.NodeId) error {
	msg := p2p.AddressMsg{
		From:     e.id,
		NodeList: e.peers,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.ADDRESS_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) sendTransaction(to p2p.NodeId, tx *transactions.Transaction) error {
	msg := p2p.TransactionMsg{
		From:        e.id,
		Transaction: *tx,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.TX_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) sendBlockchainInfo(to p2p.NodeId) error {
	msg := p2p.BlockchainInfoMsg{
		From:              e.id,
		Height:            e.Height,
		Difficulty:        e.Difficulty,
		PreviousBlockHash: e.PreviousBlockHash,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.BLOCKCHAIN_INFO_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) broadcastAcceptedBlock(
	block *blocks.Block,
) error {
	msg := p2p.AcceptedBlockMsg{
		From:       e.id,
		Block:      *block,
		Difficulty: e.Difficulty,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}

	payload := p2p.ACCEPTED_BLOCK_MSG.MakePayload(enc)
	return e.broadcast(payload)
}

func (e *ExecuterNode) sendReward(to p2p.NodeId) error {
	msg := p2p.RewardMsg{From: e.id}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}

	payload := p2p.REWARD_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) broadcastOfferBlock(block *blocks.Block) error {
	msg := p2p.OfferBlockMsg{
		From:  e.id,
		Block: *block,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}

	payload := p2p.OFFER_BLOCK_MSG.MakePayload(enc)
	err = e.broadcast(payload)
	if err != nil {
		return err
	}

	hash, err := block.Bundle.HashTransactions()
	if err != nil {
		return err
	}
	e.offeredTxHash = hash
	e.offeredTime = time.Now().UnixMilli()
	return nil
}
