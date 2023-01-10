package pow

import (
	"bytes"
	"log"
	"math"
	"math/big"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/transactions"

	"golang.org/x/crypto/sha3"
)

const (
	MAX_NONCE = math.MaxUint64
)

type ProofOfWork struct {
	block      *blocks.Block
	difficulty byte
	target     *big.Int
}

func MineBlock(
	transactions transactions.TxBundle,
	info blocks.BlockInfo,
) (*blocks.Block, error) {
	block := blocks.NewBlock(transactions, info)
	pow := NewProofOfWork(block)
	nonce, hash, err := pow.Run()
	if err != nil {
		return nil, err
	}

	block.Hash = hash[:]
	block.Nonce = nonce
	return block, nil
}

func NewProofOfWork(b *blocks.Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(math.MaxUint8-b.Difficulty))
	pow := ProofOfWork{
		block:      b,
		difficulty: b.Difficulty,
		target:     target,
	}
	return &pow
}

func (pow *ProofOfWork) Run() (uint64, []byte, error) {
	timestampHex, err := common.ToHex(pow.block.Timestamp)
	if err != nil {
		return 0, nil, err
	}
	targetBitsHex, err := common.ToHex(pow.difficulty)
	if err != nil {
		return 0, nil, err
	}

	transactionHash, err := pow.block.Bundle.HashTransactions()
	if err != nil {
		return 0, nil, err
	}

	var hashInt big.Int
	var hash [32]byte
	var nonce uint64 = 0

	log.Println("mining a new block")
	for nonce < MAX_NONCE {
		nonceHex, err := common.ToHex(nonce)
		if err != nil {
			return 0, nil, err
		}

		data := bytes.Join(
			[][]byte{
				pow.block.PreviousBlockHash,
				transactionHash,
				timestampHex,
				targetBitsHex,
				nonceHex,
			},
			nil,
		)
		hash = sha3.Sum256(data)
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	log.Printf("mined hash:\n%x\n", hash)
	return nonce, hash[:], nil
}

func (pow *ProofOfWork) Validate() (bool, error) {
	timestampHex, err := common.ToHex(pow.block.Timestamp)
	if err != nil {
		return false, err
	}
	targetBitsHex, err := common.ToHex(pow.difficulty)
	if err != nil {
		return false, err
	}
	nonceHex, err := common.ToHex(pow.block.Nonce)
	if err != nil {
		return false, err
	}
	transactionHash, err := pow.block.Bundle.HashTransactions()
	if err != nil {
		return false, err
	}

	data := bytes.Join(
		[][]byte{
			pow.block.PreviousBlockHash,
			transactionHash,
			timestampHex,
			targetBitsHex,
			nonceHex,
		},
		nil,
	)
	var hashInt big.Int
	hash := sha3.Sum256(data)
	hashInt.SetBytes(hash[:])

	result := hashInt.Cmp(pow.target)
	return result == -1, nil
}
