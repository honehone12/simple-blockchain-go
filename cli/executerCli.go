package cli

import (
	"simple-blockchain-go/nodes"
)

func startExecuterNode(port string) error {
	s, err := nodes.NewExecuterNode(port)
	if err != nil {
		return err
	}
	return s.Run()
}
