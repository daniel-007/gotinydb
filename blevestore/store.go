package blevestore

import (
	"fmt"
	"os"

	"github.com/dgraph-io/badger"

	"github.com/blevesearch/bleve/index/store"
	"github.com/blevesearch/bleve/registry"
)

const (
	Name = "internal"
)

type Store struct {
	// name is defined by the path
	name             string
	writeTxn         *badger.Txn
	encrypt          func(dbID, clearContent []byte) (encryptedContent []byte)
	decrypt          func(dbID, encryptedContent []byte) (decryptedContent []byte, _ error)
	indexPrefixID    []byte
	indexPrefixIDLen int
	db               *badger.DB
	mo               store.MergeOperator
}

func New(mo store.MergeOperator, config map[string]interface{}) (store.KVStore, error) {
	path, ok := config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("must specify path")
	}
	if path == "" {
		return nil, os.ErrInvalid
	}

	prefixID, ok := config["prefix"].([]byte)
	if !ok {
		return nil, fmt.Errorf("must specify a prefix")
	}

	db, ok := config["db"].(*badger.DB)
	if !ok {
		return nil, fmt.Errorf("must specify a db")
	}

	encrypt, ok := config["encrypt"].(func(dbID, clearContent []byte) []byte)
	if !ok {
		return nil, fmt.Errorf("the encrypt function must be provided")
	}

	decrypt, ok := config["decrypt"].(func(dbID, encryptedContent []byte) (decryptedContent []byte, _ error))
	if !ok {
		return nil, fmt.Errorf("the decrypt function must be provided")
	}

	writeTxn, ok := config["writeTxn"].(*badger.Txn)
	if !ok {
		return nil, fmt.Errorf("the write transaction pointer must be initialized")
	}

	rv := Store{
		name:             path,
		indexPrefixID:    prefixID,
		indexPrefixIDLen: len(prefixID),
		writeTxn:         writeTxn,
		encrypt:          encrypt,
		decrypt:          decrypt,
		db:               db,
		mo:               mo,
	}
	return &rv, nil
}

func (bs *Store) Close() error { return nil }

// Reader open a new transaction but it needs to be closed
func (bs *Store) Reader() (store.KVReader, error) {
	return &Reader{
		store:         bs,
		txn:           bs.db.NewTransaction(false),
		indexPrefixID: bs.indexPrefixID,
	}, nil
}

func (bs *Store) Writer() (store.KVWriter, error) {
	return &Writer{
		store: bs,
	}, nil
}

func init() {
	registry.RegisterKVStore(Name, New)
}

func (bs *Store) buildID(key []byte) []byte {
	return append(bs.indexPrefixID, key...)
}
