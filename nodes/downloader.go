package nodes

import (
	"bytes"
	"errors"
	"log"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/p2p"
)

// impl simplest way
// one peer from first to end
// send - response - verify block one to one
func (e *ExecuterNode) startDownloadBlocks(to p2p.NodeId) {
	h := e.Height
	if h != 0 {
		// we want next block, except genesis block
		h++
	}
	log.Printf("start downloading blocks from %d...\n", h)
	e.sendSyncBlockRequest(to, h)
}

func (e *ExecuterNode) sendSyncBlockRequest(to p2p.NodeId, height uint64) error {
	if to.Kind != p2p.EXECUTER_NODE {
		return errors.New("sync block request should be sent to executer node")
	}

	msg := p2p.SyncBlockRequestMsg{
		From:   e.id,
		Height: height,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.SYNC_BLOCK_REQUEST_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) handleSyncBlockRequest(raw []byte) error {
	msg, err := common.Decode[p2p.SyncBlockRequestMsg](raw)
	if err != nil {
		return err
	}

	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}
	log.Printf(
		"node's height: %d, requested height: %d \n",
		e.Height, msg.Height,
	)
	if msg.Height > e.Height {
		// here means there might be higher(longer) forks
		// or just be spam
		log.Println("requested higher blocks, skipping...")
		return nil
	}

	block, err := e.GetBlockByHeight(msg.Height)
	if err != nil {
		return err
	}
	return e.sendSyncBlockResponse(msg.From, block)
}

func (e *ExecuterNode) sendSyncBlockResponse(
	to p2p.NodeId, block *blocks.Block,
) error {
	msg := p2p.SyncBlockResponseMsg{
		From:     e.id,
		Block:    *block,
		IsLatest: e.Height == block.Height,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}
	payload := p2p.SYNC_BLOCK_RESPONSE_MSG.MakePayload(enc)
	return e.send(to, payload)
}

func (e *ExecuterNode) handleSyncBlockResponse(raw []byte) error {
	e.Lock()
	defer e.Unlock()

	msg, err := common.Decode[p2p.SyncBlockResponseMsg](raw)
	if err != nil {
		return err
	}

	if msg.From.Kind != p2p.EXECUTER_NODE {
		return nil
	}
	log.Printf(
		"node's height: %d, received height: %d \n",
		e.Height, msg.Block.Height,
	)

	if msg.Block.Height == 0 {
		err = e.OverwriteGenesis(msg.Block)
		if err != nil {
			return err
		}
	} else {
		if e.Height+1 != msg.Block.Height {
			log.Println("received unexpected block, skipping...")
			return nil
		}

		err = e.syncBlockImpl(&msg.Block)
		if err != nil {
			return err
		}
	}

	if msg.IsLatest {
		e.isSyncing = false
		log.Println("syncing is done...")
		return nil
	}

	return e.sendSyncBlockRequest(msg.From, e.Height+1)
}

func (e *ExecuterNode) syncBlockImpl(block *blocks.Block) error {
	// verify
	ok, err := e.VerifyBlock(block)
	if err != nil {
		return err
	}
	if !ok {
		log.Println("received block is invalid")
		return nil
	}

	// execute
	for _, tx := range block.Bundle.Transactions {
		err = e.executeTransaction(tx)
		if err != nil {
			return err
		}
	}

	// put to db
	err = e.PutBlockWithCheck(block)
	if err != nil {
		return err
	}

	// calc state
	stateHash, err := e.calcState()
	if err != nil {
		return err
	}
	if !bytes.Equal(stateHash, block.StateHash) {
		// there are no way to restore database for now
		return errors.New("state hash does not match")
	}

	return nil
}
