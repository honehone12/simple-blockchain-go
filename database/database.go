package database

import (
	"fmt"
	"log"
	"simple-blockchain-go/accounts"
	"simple-blockchain-go/blocks"
	"simple-blockchain-go/common"
	"simple-blockchain-go/geneis"

	bolt "go.etcd.io/bbolt"
)

const (
	DATABASE_FILE = "%s_database.db"
	BLOCKS_BUCKET = "blocks"
	STATE_BUCKET  = "state"
	LATEST_TAG    = "latest"
	HEIGHT_TAG    = "height"
)

type Database struct {
	innerDb *bolt.DB
}

func DatabaseFileName(id string) string {
	return fmt.Sprintf(DATABASE_FILE, id)
}

func ExistsDatabaseFile(id string) bool {
	return common.ExistFile(DatabaseFileName(id))
}

func Open(id string) (Database, error) {
	if ExistsDatabaseFile(id) {
		log.Printf("found existing database for id: %s\n", id)
		db, err := bolt.Open(DatabaseFileName(id), 0600, nil)
		return Database{db}, err
	}

	// create new
	db, err := bolt.Open(DatabaseFileName(id), 0600, nil)
	if err != nil {
		return Database{}, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		// bucket for blocks
		b, err := tx.CreateBucket([]byte(BLOCKS_BUCKET))
		if err != nil {
			return err
		}

		// bucket for state
		s, err := tx.CreateBucket([]byte(STATE_BUCKET))
		if err != nil {
			return err
		}

		// genesis block
		genesis, err := geneis.GenerateGenesis()
		if err != nil {
			return err
		}
		err = b.Put(genesis.Hash, genesis.EncodedBlock)
		if err != nil {
			return err
		}
		h, err := common.ToHex(uint64(0))
		if err != nil {
			return err
		}
		err = b.Put(h, genesis.Hash)
		if err != nil {
			return err
		}

		// generator account
		genesisState := accounts.AccountState{
			Nonce: 0, Balance: geneis.GENESIS_BALANCE,
		}
		enc, err := common.Encode(genesisState)
		if err != nil {
			return err
		}
		err = s.Put(genesis.PublicKey, enc)
		if err != nil {
			return err
		}

		// shortcut for latest
		err = b.Put([]byte(LATEST_TAG), genesis.Hash)
		if err != nil {
			return err
		}

		// shortcut for height
		err = b.Put([]byte(HEIGHT_TAG), h)
		return err
	})

	log.Printf("database for id: %s is created\n", id)
	return Database{db}, err
}

func (db *Database) GetHeight() (uint64, error) {
	var hex []byte
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKS_BUCKET))
		hex = b.Get([]byte(HEIGHT_TAG))
		return nil
	})
	if err != nil {
		return 0, err
	}
	return common.FromHex[uint64](hex)
}

func (db *Database) GetLatest() ([]byte, error) {
	var hash []byte
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKS_BUCKET))
		hash = b.Get([]byte(LATEST_TAG))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (db *Database) GetBlockByHash(blockHash []byte) (*blocks.Block, error) {
	var enc []byte
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKS_BUCKET))
		enc = b.Get(blockHash)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return common.Decode[blocks.Block](enc)
}

func (db *Database) GetBlockByHeight(height uint64) (*blocks.Block, error) {
	var enc []byte
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKS_BUCKET))
		h, err := common.ToHex(height)
		if err != nil {
			return err
		}
		hash := b.Get(h)
		enc = b.Get(hash)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return common.Decode[blocks.Block](enc)
}

func (db *Database) PutBlock(block *blocks.Block) error {
	return db.innerDb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKS_BUCKET))
		enc, err := common.Encode(block)
		if err != nil {
			return err
		}
		err = b.Put(block.Hash, enc)
		if err != nil {
			return err
		}
		h, err := common.ToHex(block.Height)
		if err != nil {
			return err
		}
		err = b.Put(h, block.Hash)
		if err != nil {
			return err
		}

		err = b.Put([]byte(HEIGHT_TAG), h)
		if err != nil {
			return err
		}
		return b.Put([]byte(LATEST_TAG), block.Hash)
	})
}

func (db *Database) GetAccountState(pubKey []byte) (*accounts.AccountState, error) {
	var enc []byte
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(STATE_BUCKET))
		enc = b.Get(pubKey)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if enc == nil {
		return nil, nil
	}
	return common.Decode[accounts.AccountState](enc)
}

func (db *Database) PutAccountState(
	pubKey []byte, state *accounts.AccountState,
) error {
	return db.innerDb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(STATE_BUCKET))
		enc, err := common.Encode(state)
		if err != nil {
			return err
		}
		return b.Put(pubKey, enc)
	})
}

func (db *Database) GetAllStates() ([][]byte, error) {
	raw := [][]byte{}
	err := db.innerDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(STATE_BUCKET))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			raw = append(raw, append(k, v...))
		}
		return nil
	})
	return raw, err
}
