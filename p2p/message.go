package p2p

import (
	"log"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/transactions"
)

type MessageKind byte

const (
	ADDRESS_MSG MessageKind = iota + 1
	OFFER_BLOCK_MSG
	REGISTER_BLOCK_MSG
	ACCEPTED_BLOCK_MSG
	REWARD_MSG
	SYNC_BLOCK_REQUEST_MSG
	SYNC_BLOCK_RESPONSE_MSG
	BLOCKCHAIN_INFO_MSG
	ACCOUNT_MSG
	ACCOUNT_INFO_MSG
	INV_MSG
	TX_MSG
	TX_POOL_MSG
	JOIN_MSG
)

func (mk MessageKind) MakePayload(data []byte) []byte {
	bs := make([]byte, 0, len(data)+1)
	bs = append(bs, byte(mk))
	bs = append(bs, data...)
	return bs
}

func (mk MessageKind) ToString() string {
	switch mk {
	case ADDRESS_MSG:
		return "address message"
	case OFFER_BLOCK_MSG:
		return "offer block message"
	case REGISTER_BLOCK_MSG:
		return "register block message"
	case ACCEPTED_BLOCK_MSG:
		return "accepted block message"
	case SYNC_BLOCK_REQUEST_MSG:
		return "sync block request"
	case SYNC_BLOCK_RESPONSE_MSG:
		return "sync block response"
	case REWARD_MSG:
		return "reward message"
	case BLOCKCHAIN_INFO_MSG:
		return "blockchain info message"
	case ACCOUNT_MSG:
		return "account message"
	case ACCOUNT_INFO_MSG:
		return "account info message"
	case INV_MSG:
		return "inv message"
	case TX_MSG:
		return "tx message"
	case TX_POOL_MSG:
		return "tx pool message"
	case JOIN_MSG:
		return "join message"
	default:
		log.Panicf("unknown value %d", mk)
	}
	return ""
}

type AddressMsg struct {
	From     NodeId
	NodeList []NodeId
}

type BlockchainInfoMsg struct {
	From              NodeId
	Height            uint64
	Difficulty        byte
	PreviousBlockHash []byte
}

type OfferBlockMsg struct {
	From  NodeId
	Block blocks.Block
}

type RegisterBlockMsg struct {
	From  NodeId
	Block blocks.Block
}

type AcceptedBlockMsg struct {
	From       NodeId
	Block      blocks.Block
	Difficulty byte
}

// this is just notification
// do nothing actually
type RewardMsg struct {
	From NodeId
}

type SyncBlockRequestMsg struct {
	From   NodeId
	Height uint64
}

type SyncBlockResponseMsg struct {
	From     NodeId
	IsLatest bool
	Block    blocks.Block
}

type AccountMsg struct {
	From      NodeId
	PublicKey []byte
	Signature []byte
}

type AccountInfoMsg struct {
	From      NodeId
	PublicKey []byte
	Balance   uint64
	Nance     uint64
}

type TransactionMsg struct {
	From        NodeId
	Transaction transactions.Transaction
}

type TxPoolMsg struct {
	From         NodeId
	Transactions []transactions.Transaction
}

type JoinMsg struct {
	From    string
	Version byte
	Kind    NodeKind
}
