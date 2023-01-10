package accounts

import (
	"log"
	"simple-blockchain-go/geneis"
)

type AccountState struct {
	Nonce   uint64
	Balance uint64
}

func (as *AccountState) Subtract(amount uint64) bool {
	if amount > as.Balance {
		return false
	}

	as.Balance -= amount
	return true
}

func (as *AccountState) Add(amount uint64) bool {
	max := geneis.GENESIS_BALANCE - as.Balance
	if amount > max {
		return false
	}

	as.Balance += amount
	return true
}

func (as *AccountState) CheckNonce(nonce uint64) bool {
	log.Printf("checking nonce, received: %d expected: %d", nonce, as.Nonce)
	ok := as.Nonce == nonce
	if ok {
		as.Nonce++
	}
	return ok
}
