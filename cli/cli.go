package cli

import (
	"flag"
	"fmt"
	"os"
)

func printUsage() {
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println(" miner -p PORT (start miner on PORT)")
	fmt.Println(" executer -p PORT (start storage node on PORT)")
	fmt.Println(" wallet -p PORT (start wallet on PORT)")
	fmt.Println()
}

func validateArgs() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
}

func Run() error {
	validateArgs()
	executerCmd := flag.NewFlagSet("executer", flag.ExitOnError)
	minerCmd := flag.NewFlagSet("miner", flag.ExitOnError)
	walletCmd := flag.NewFlagSet("wallet", flag.ExitOnError)

	executerPort := executerCmd.String("p", "3000", "port number to use")
	minerPort := minerCmd.String("p", "3001", "port number to use")
	walletPort := walletCmd.String("p", "3002", "port number to use")

	var err error
	switch os.Args[1] {
	case "executer":
		err = executerCmd.Parse(os.Args[2:])
	case "miner":
		err = minerCmd.Parse(os.Args[2:])
	case "wallet":
		err = walletCmd.Parse(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		return err
	}

	fmt.Println()
	if executerCmd.Parsed() {
		err = startExecuterNode(*executerPort)
	} else if minerCmd.Parsed() {
		err = startMinerNode(*minerPort)
	} else if walletCmd.Parsed() {
		err = startWalletNode(*walletPort)
	}
	return err
}
