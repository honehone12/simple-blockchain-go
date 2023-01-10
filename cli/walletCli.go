package cli

import (
	"simple-blockchain-go/nodes"
)

func startWalletNode(port string) error {
	w, err := nodes.NewWalletNode(port)
	if err != nil {
		return err
	}
	return w.Run()
}
