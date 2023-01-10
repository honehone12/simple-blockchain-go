package nodes

import (
	"bytes"
	"errors"
	"log"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/geneis"
	"simple-blockchain-go/memory"
	"simple-blockchain-go/merkleTree"
	"simple-blockchain-go/transactions"
	"time"
)

func (e *ExecuterNode) retry() {
	time.AfterFunc(time.Millisecond*10000, func() {
		e.epoch.C() <- true
	})
}

func (e *ExecuterNode) executionRoutine() {
	log.Printf("epoch %d: next: %d\n", e.Height, e.Height+1)
	if e.isSyncing {
		return
	}

	err := e.checkHealth()
	if err != nil {
		log.Panic(err)
	}

	if e.txPool.Len() == 0 {
		log.Println("no transactions to execute")
		if isRendezvous(e.id) {
			e.retry()
		}
		return
	}

	// chose transactions for block
	txsForExecute, err := e.txPool.GetTransactionForBlock()
	if err != nil {
		log.Panic(err)
	}

	// execute transaction and get other tx on fail
	var errorLen int
	var executedTxs []transactions.Transaction
	for {
		for _, tx := range txsForExecute {
			err := e.executeTransaction(tx)
			if err != nil {
				log.Panic(err)
			}
			executedTxs = append(executedTxs, tx)
		}
		if errorLen == 0 {
			break
		}
		txsForExecute = e.txPool.Get(errorLen)
	}

	block := blocks.NewBlock(
		transactions.TxBundle{Transactions: executedTxs},
		e.BlockInfo,
	)
	// increment because this is next block
	block.Height++

	// delete tx pool
	txKeys := memory.GetTxKeys(block)
	e.txPool.BatchRemove(txKeys)

	// calc state hash
	stateHash, err := e.calcState()
	if err != nil {
		log.Panic(err)
	}
	block.StateHash = stateHash

	log.Printf(
		"block at height %d is created, broadcasting offer...\n",
		block.Height,
	)
	log.Printf("including %d tx\n", len(block.Bundle.Transactions))
	err = e.broadcastOfferBlock(block)
	if err != nil {
		log.Panic(err)
	}
}

// this is very shortcut(rough or actually crazy) implementation...
func (e *ExecuterNode) calcState() ([]byte, error) {
	states, err := e.GetAllStates()
	if err != nil {
		return nil, err
	}
	l := len(states)
	targetLen := common.NextPowerOf2(l)
	for i := 0; i < targetLen-l; i++ {
		states = append(states, states[len(states)-1])
	}
	mTree, err := merkleTree.NewMerkleTree(states)
	if err != nil {
		return nil, err
	}
	return mTree.RootNode.Data, nil
}

func (e *ExecuterNode) executeTransaction(tx transactions.Transaction) error {
	// check again
	ok, err := tx.Verify()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid transaction")
	}

	raw := tx.InnerData.Data
	cmdKind := transactions.CommandKind(raw[0])
	switch cmdKind {
	case transactions.AIRDROP_CMD:
		err = e.executeAirdrop(raw[1:], tx.InnerData.Nonce)
	case transactions.TRANSFER_CMD:
		err = e.executeTransfer(raw[1:], tx.InnerData.Nonce)
	default:
		log.Println("unknown command skipping...")
	}
	return err
}

func (e *ExecuterNode) executeAirdrop(raw []byte, nonce uint64) error {
	cmd, err := common.Decode[transactions.Airdrop](raw)
	if err != nil {
		return err
	}

	log.Printf("airdropping %d...\n", cmd.Amount)
	if e.airdropAccount == nil {
		generator, err := geneis.GetGenerator()
		if err != nil {
			return err
		}
		genesisState, err := e.GetAccountState(generator)
		if err != nil {
			return err
		}
		log.Printf("genesis balance: %d\n", genesisState.Balance)

		e.airdropAccount = generator
	}

	return e.transferImpl(
		cmd.PublicKey, nonce,
		e.airdropAccount, cmd.PublicKey, cmd.Amount,
	)
}

func (e *ExecuterNode) executeTransfer(raw []byte, nonce uint64) error {
	cmd, err := common.Decode[transactions.Transfer](raw)
	if err != nil {
		return err
	}

	log.Printf("transfering %d...\n", cmd.Amount)
	return e.transferImpl(
		cmd.From, nonce,
		cmd.From, cmd.To, cmd.Amount,
	)
}

func (e *ExecuterNode) transferImpl(
	caller []byte, nonce uint64,
	from []byte, to []byte, amount uint64,
) error {
	if bytes.Equal(from, to) {
		return errors.New("invalid public keys")
	}

	// decrease from's balance
	fromState, err := e.GetAccountState(from)
	if err != nil {
		return err
	}
	if bytes.Equal(caller, from) {
		if !fromState.CheckNonce(nonce) {
			return errors.New("nonce is not expected")
		}
	}
	ok := fromState.Subtract(amount)
	if !ok {
		return errors.New("underflow")
	}
	err = e.PutAccountState(from, fromState)
	if err != nil {
		return err
	}

	// increase to's balance
	toState, err := e.GetAccountStateSafe(to)
	if err != nil {
		return err
	}
	if bytes.Equal(caller, to) {
		if !toState.CheckNonce(nonce) {
			return errors.New("nonce is not expected")
		}
	}
	ok = toState.Add(amount)
	if !ok {
		return errors.New("overflow")
	}
	return e.PutAccountState(to, toState)
}
