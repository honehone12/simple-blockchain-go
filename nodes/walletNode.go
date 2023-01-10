package nodes

import (
	"io"
	"log"
	"net"
	"simple-blockchain-go/common"
	"simple-blockchain-go/p2p"
	"simple-blockchain-go/transactions"
	"simple-blockchain-go/wallets"
	"strconv"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

const (
	NUM_ACCOUNTS = 10
)

type WalletNode struct {
	Node
	accounts map[string]*wallets.Wallet
}

func NewWalletNode(port string) (*WalletNode, error) {
	w := WalletNode{
		Node: Node{
			id:      p2p.NewNodeId(port, p2p.WALLET_NODE),
			version: 1,
		},
		accounts: make(map[string]*wallets.Wallet),
	}
	for i := 0; i < NUM_ACCOUNTS; i++ {
		wallet, err := wallets.NewWallet(port, strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		key := base58.Encode(wallet.PublicKey())
		w.accounts[key] = wallet
	}
	w.AppendPeer(p2p.DefaultKnownNode(port, p2p.WALLET_NODE))
	return &w, nil
}

func (w *WalletNode) Run() error {
	listener, err := net.Listen(p2p.TCP, string(w.id.Ip))
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Printf("wallet node is listening at %s", w.id.Ip)

	err = w.broadcastJoin()
	if err != nil {
		return err
	}

	p, _ := w.GetPeer(0)
	err = w.sendAccount(p)
	if err != nil {
		return err
	}

	go w.startSendingAirdropTransactions()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go w.handleConnection(conn)
	}
}

func (w *WalletNode) handleConnection(conn net.Conn) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	msgKind := p2p.MessageKind(request[0])
	log.Printf("received msg '%s'\n", msgKind.ToString())

	switch msgKind {
	case p2p.ADDRESS_MSG:
		err = w.handleAddress(request[1:])
	case p2p.ACCOUNT_INFO_MSG:
		err = w.handleAccountInfo(request[1:])
	default:
		log.Println("unknown message skipping...")
	}
	if err != nil {
		log.Panic(err)
	}
}

func (w *WalletNode) handleAccountInfo(raw []byte) error {
	msg, err := common.Decode[p2p.AccountInfoMsg](raw)
	if err != nil {
		return err
	}
	key := base58.Encode(msg.PublicKey)
	log.Printf(
		"account %s balance: %d, nonce:%d\n",
		key, msg.Balance, msg.Nance,
	)
	w.accounts[key].Nonce = msg.Nance
	w.accounts[key].Balance = msg.Balance
	return nil
}

func (w *WalletNode) startSendingAirdropTransactions() {
	ticker := time.NewTicker(time.Millisecond * 1000)
	defer ticker.Stop()
	for range ticker.C {
		if w.PeerLen() <= 1 {
			continue
		}

		for _, a := range w.accounts {
			cmd := transactions.Airdrop{
				PublicKey: a.PublicKey(),
				Amount:    1,
			}
			cmdEnc, err := common.Encode(cmd)
			if err != nil {
				log.Panic(err)
			}
			payload := transactions.AIRDROP_CMD.MakePayload(cmdEnc)
			tx := transactions.Transaction{
				InnerData: transactions.TransactionData{
					Data:      payload,
					PublicKey: a.PublicKey(),
					Nonce:     a.Nonce,
					Timestamp: time.Now().UnixMilli(),
				},
			}
			err = a.Sign(&tx)
			if err != nil {
				log.Panic(err)
			}

			p, _ := w.GetPeer(0)
			log.Printf("sending airdrop transaction to %s\n", p.Ip)
			err = w.sendTxMessage(p, &tx)
			if err != nil {
				log.Panic(err)
			}
			a.Nonce++
		}
	}
}

func (w *WalletNode) sendTxMessage(
	to p2p.NodeId, tx *transactions.Transaction,
) error {
	msg := p2p.TransactionMsg{
		From:        w.id,
		Transaction: *tx,
	}
	enc, err := common.Encode(msg)
	if err != nil {
		return err
	}

	payload := p2p.TX_MSG.MakePayload(enc)
	err = w.send(to, payload)
	if err != nil {
		return err
	}
	return nil
}

func (w *WalletNode) sendAccount(to p2p.NodeId) error {
	for _, a := range w.accounts {
		msg := p2p.AccountMsg{
			From:      w.id,
			PublicKey: a.PublicKey(),
			Signature: nil,
		}
		content, err := common.Encode(msg.From)
		if err != nil {
			return err
		}
		msg.Signature = a.QuickSign(content)
		enc, err := common.Encode(msg)
		if err != nil {
			return err
		}
		payload := p2p.ACCOUNT_MSG.MakePayload(enc)
		err = w.send(to, payload)
		if err != nil {
			return err
		}
	}
	return nil
}
