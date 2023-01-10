package blocks

import (
	"simple-blockchain-go/transactions"
	"time"
)

type BlockInfo struct {
	Height            uint64
	Difficulty        byte
	PreviousBlockHash []byte
}

type Block struct {
	BlockInfo
	Timestamp int64
	Bundle    transactions.TxBundle
	Hash      []byte
	Nonce     uint64
	StateHash []byte
}

func NewBlock(
	transactions transactions.TxBundle,
	info BlockInfo,
) *Block {
	block := Block{
		BlockInfo: info,
		Timestamp: time.Now().Unix(),
		Bundle:    transactions,
		Hash:      nil,
		Nonce:     0,
	}
	return &block
}
