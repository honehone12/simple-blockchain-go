package geneis

import (
	"crypto/rand"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/pow"
	"simple-blockchain-go/transactions"
	"simple-blockchain-go/wallets"
	"time"
)

const (
	GENESIS_DIFFICULTY        = 10
	GENERATOR_KEY             = "generator"
	GENESIS_BALANCE    uint64 = 10_000_000_000_000_000_000
)

type Genesis struct {
	Hash         []byte
	EncodedBlock []byte
	PublicKey    []byte
}

func getGenerator() (*wallets.Wallet, error) {
	return wallets.NewWallet(GENERATOR_KEY, "")
}

func mineGenesisBlock(
	generator *wallets.Wallet, info blocks.BlockInfo,
) (*blocks.Block, error) {
	data := make([]byte, 32)
	_, err := rand.Read(data)
	if err != nil {
		return nil, err
	}
	tx := transactions.Transaction{
		InnerData: transactions.TransactionData{
			Data:      data,
			PublicKey: generator.PublicKey(),
			Timestamp: time.Now().Unix(),
		},
	}
	err = generator.Sign(&tx)
	if err != nil {
		return nil, err
	}

	return pow.MineBlock(
		transactions.TxBundle{
			Transactions: []transactions.Transaction{tx},
		},
		info,
	)
}

func GetGenerator() ([]byte, error) {
	w, err := getGenerator()
	if err != nil {
		return nil, err
	}
	return w.PublicKey(), nil
}

func GenerateGenesis() (*Genesis, error) {
	generator, err := getGenerator()
	if err != nil {
		return nil, err
	}
	genesis, err := mineGenesisBlock(
		generator,
		blocks.BlockInfo{
			Difficulty: GENESIS_DIFFICULTY,
		},
	)
	if err != nil {
		return nil, err
	}
	enc, err := common.Encode(genesis)
	if err != nil {
		return nil, err
	}
	return &Genesis{
		Hash:         genesis.Hash,
		EncodedBlock: enc,
		PublicKey:    generator.PublicKey(),
	}, nil
}
