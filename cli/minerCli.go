package cli

import (
	"simple-blockchain-go/nodes"
)

func startMinerNode(port string) error {
	m := nodes.NewMinerNode(port)
	return m.Run()
}
