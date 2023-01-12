package transactions

import (
	"simple-blockchain-go/common"
	"simple-blockchain-go/merkleTree"

	"golang.org/x/exp/slices"
)

type TxBundle struct {
	Transactions []Transaction
}

func (b *TxBundle) HashTransactions() ([]byte, error) {
	var transactions [][]byte
	for _, tx := range b.Transactions {
		enc, err := common.Encode(tx)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, enc)
	}

	mTree, err := merkleTree.NewMerkleTree(transactions)
	if err != nil {
		return nil, err
	}
	return mTree.RootNode.Data, nil
}

func (b *TxBundle) SortTransactions() {
	slices.SortFunc(b.Transactions, func(a, b Transaction) bool {
		return a.InnerData.Nonce < b.InnerData.Nonce
	})
}
