package main

import (
	"log"
	"simple-blockchain-go/cli"
)

func main() {
	err := cli.Run()
	if err != nil {
		log.Panic(err)
	}
}
