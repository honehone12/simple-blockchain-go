package transactions

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"log"
	"simple-blockchain-go/common"

	"golang.org/x/crypto/sha3"
)

type TransactionData struct {
	Data      []byte
	PublicKey ed25519.PublicKey
	Nonce     uint64
	Signature []byte
	Timestamp int64
}

type Transaction struct {
	Hash      [32]byte
	InnerData TransactionData
}

func (tx *Transaction) ContentsCheck() error {
	if tx.InnerData.Timestamp == 0 {
		return errors.New("timestamp is zero")
	}
	if len(tx.InnerData.Data) == 0 {
		return errors.New("data is empty")
	}
	if len(tx.InnerData.PublicKey) == 0 {
		return errors.New("public key is empty")
	}
	return nil
}

func (tx *Transaction) Verify() (bool, error) {
	err := tx.ContentsCheck()
	if err != nil {
		return false, err
	}

	enc, err := common.Encode(&tx.InnerData)
	if err != nil {
		return false, err
	}
	hash := sha3.Sum256(enc)
	if !bytes.Equal(hash[:], tx.Hash[:]) {
		log.Println("transaction hash is broken")
		return false, nil
	}

	return ed25519.Verify(
		tx.InnerData.PublicKey,
		tx.InnerData.Data,
		tx.InnerData.Signature,
	), nil
}
