package blockchain

import (
	"bytes"
	"log"
	"simple-blockchain-go/accounts"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/database"
	"simple-blockchain-go/pow"
	"sync"
)

const (
	DEFAULT_DIFFICULTY byte = 20
)

type Blockchain struct {
	sync.Mutex
	blocks.BlockInfo
	database.Database
}

func NewBlockchain(id string) (*Blockchain, error) {
	bc := Blockchain{}
	db, err := database.Open(id)
	if err != nil {
		return &bc, err
	}

	height, err := db.GetHeight()
	if err != nil {
		return &bc, err
	}

	latestHash, err := db.GetLatest()
	if err != nil {
		return &bc, err
	}

	bc.Height = height
	bc.PreviousBlockHash = latestHash
	bc.Database = db
	bc.Difficulty = DEFAULT_DIFFICULTY
	log.Printf(
		"blockchain starts at\n height: %d\n difficulty: %d\n letest: %x",
		bc.Height, bc.Difficulty, bc.PreviousBlockHash,
	)
	return &bc, nil
}

func (bc *Blockchain) GetAccountStateSafe(pubKey []byte) (*accounts.AccountState, error) {
	state, err := bc.GetAccountState(pubKey)
	if err != nil {
		return nil, err
	}
	if state != nil {
		return state, nil
	}

	state = &accounts.AccountState{
		Nonce:   0,
		Balance: 0,
	}
	err = bc.PutAccountState(pubKey, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (bc *Blockchain) VerifyBlock(block *blocks.Block) (bool, error) {
	receivedHeight := block.Height
	expectedHeight := bc.Height + 1

	if receivedHeight != expectedHeight {
		log.Printf(
			"received block height is %d, expected %d\n",
			receivedHeight, expectedHeight,
		)
		return false, nil
	}

	if !bytes.Equal(bc.PreviousBlockHash, block.PreviousBlockHash) {
		log.Printf(
			"received previous hash:\n %x\nexpected:\n %x\n",
			block.PreviousBlockHash, bc.PreviousBlockHash,
		)
		return false, nil
	}

	validator := pow.NewProofOfWork(block)
	ok, err := validator.Validate()
	if err != nil {
		return false, err
	}
	if !ok {
		log.Println("pow block validation failed...")
		return false, nil
	}

	log.Printf("verified block at height: %d\n", expectedHeight)
	return true, nil
}

func (bc *Blockchain) PutBlockWithCheck(block *blocks.Block) error {
	currentHeight, err := bc.GetHeight()
	if err != nil {
		return err
	}
	expectedHeight := currentHeight + 1
	if block.Height != expectedHeight {
		log.Printf(
			"height conflict\n current: %d\n expected: %d\n received: %d\n",
			currentHeight, expectedHeight, block.Height,
		)
		return nil
	}

	bc.Height = block.Height
	bc.PreviousBlockHash = block.Hash

	return bc.PutBlock(block)
}

// need better way to open database
// this func should be removed
func (bc *Blockchain) OverwriteGenesis(block blocks.Block) error {
	if block.Height != 0 {
		block.Height = 0
	}

	bc.Height = block.Height
	bc.PreviousBlockHash = block.Hash
	return bc.PutBlock(&block)
}
