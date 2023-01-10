package memory

import (
	"log"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/transactions"
	"sync"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/exp/maps"
)

type TxPool struct {
	sync.Mutex
	pool map[string]transactions.Transaction
}

func NewTransactionPool() *TxPool {
	return &TxPool{
		pool: map[string]transactions.Transaction{},
	}
}

func GetTxKeys(block *blocks.Block) []string {
	var keys []string
	for _, tx := range block.Bundle.Transactions {
		k := base58.Encode(tx.Hash[:])
		keys = append(keys, k)
	}
	return keys
}

func (p *TxPool) Len() int {
	return len(p.pool)
}

func (p *TxPool) Append(tx *transactions.Transaction) {
	p.Lock()
	defer p.Unlock()
	key := base58.Encode(tx.Hash[:])
	_, ok := p.pool[key]
	if !ok {
		p.pool[key] = *tx
	}
}

func (p *TxPool) AppendOrOverwrite(tx *transactions.Transaction) {
	p.Lock()
	defer p.Unlock()
	key := base58.Encode(tx.Hash[:])
	_, ok := p.pool[key]
	if ok {
		log.Printf("key is already exists, overwritten:\n%s\n", key)
	}
	p.pool[key] = *tx
}

func (p *TxPool) GetAll() []transactions.Transaction {
	p.Lock()
	defer p.Unlock()
	bundle := transactions.TxBundle{
		Transactions: maps.Values(p.pool),
	}
	bundle.SortTransactions()
	return bundle.Transactions
}

func (p *TxPool) Get(n int) []transactions.Transaction {
	p.Lock()
	defer p.Unlock()
	bundle := transactions.TxBundle{
		Transactions: maps.Values(p.pool),
	}
	bundle.SortTransactions()
	return bundle.Transactions[:n]
}

func (p *TxPool) GetTransactionForBlock() ([]transactions.Transaction, error) {
	txLen := len(p.pool)
	if txLen == 0 {
		return nil, nil
	}
	if txLen == 1 {
		return p.GetAll(), nil
	}

	len := common.LastPowerOf2(txLen)
	return p.Get(len), nil
}

func (p *TxPool) BatchRemove(keys []string) {
	p.Lock()
	defer p.Unlock()
	for _, k := range keys {
		delete(p.pool, k)
	}
}
