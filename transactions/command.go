package transactions

import "log"

type CommandKind byte

const (
	AIRDROP_CMD CommandKind = iota + 1
	TRANSFER_CMD
)

func (ck CommandKind) MakePayload(data []byte) []byte {
	bs := make([]byte, 0, len(data)+1)
	bs = append(bs, byte(ck))
	bs = append(bs, data...)
	return bs
}

func (ck CommandKind) ToString() string {
	switch ck {
	case AIRDROP_CMD:
		return "airdrop command"
	case TRANSFER_CMD:
		return "transfer command"
	default:
		log.Panicf("unknown command %d", ck)
	}
	return ""
}

type Airdrop struct {
	PublicKey []byte
	Amount    uint64
}

type Transfer struct {
	From   []byte
	To     []byte
	Amount uint64
}
