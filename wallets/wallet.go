package wallets

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"simple-blockchain-go/common"
	"simple-blockchain-go/keys"
	"simple-blockchain-go/transactions"

	"golang.org/x/crypto/sha3"
)

const (
	KEYPAIR_FILE = "%s_%skeypair.key"
)

type AccountInfo struct {
	Nonce   uint64
	Balance uint64
}

type Wallet struct {
	keyPair *keys.KeyPair
	AccountInfo
}

func NewWallet(id string, name string) (*Wallet, error) {
	keyFile := fmt.Sprintf(KEYPAIR_FILE, id, name)
	if common.ExistFile(keyFile) {
		f, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		key, err := common.Decode[keys.KeyPair](f)
		if err != nil {
			return nil, err
		}
		return fromKeyPair(key), nil
	}

	key, err := keys.GenerateKey()
	if err != nil {
		return nil, err
	}
	enc, err := common.Encode(key)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(keyFile, enc, 0644)
	if err != nil {
		return nil, err
	}
	return fromKeyPair(key), nil
}

func fromKeyPair(kp *keys.KeyPair) *Wallet {
	return &Wallet{
		keyPair:     kp,
		AccountInfo: AccountInfo{0, 0},
	}
}

func (w *Wallet) PublicKey() ed25519.PublicKey {
	return w.keyPair.PublicKey
}

func (w *Wallet) Sign(tx *transactions.Transaction) error {
	err := tx.ContentsCheck()
	if err != nil {
		return err
	}

	sig := ed25519.Sign(w.keyPair.PrivateKey, tx.InnerData.Data)
	tx.InnerData.Signature = sig

	enc, err := common.Encode(&tx.InnerData)
	if err != nil {
		return err
	}
	hash := sha3.Sum256(enc)
	tx.Hash = hash
	return nil
}

func (w *Wallet) QuickSign(content []byte) []byte {
	return ed25519.Sign(w.keyPair.PrivateKey, content)
}
