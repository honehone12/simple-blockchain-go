package keys

import (
	"crypto/ed25519"
	"crypto/rand"
)

type KeyPair struct {
	ed25519.PrivateKey
	ed25519.PublicKey
}

func GenerateKey() (*KeyPair, error) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		sk, pk,
	}, nil
}
