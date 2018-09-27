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
	name                 string
	primaryEncryptionKey [32]byte
	indexPrefixID        []byte
	indexPrefixIDLen     int
	db                   *badger.DB
	mo                   store.MergeOperator
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

	primaryEncryptionKey, ok := config["key"].([32]byte)
	if !ok {
		return nil, fmt.Errorf("must specify a key as [32]byte")
	}

	rv := Store{
		name:                 path,
		indexPrefixID:        prefixID,
		indexPrefixIDLen:     len(prefixID),
		primaryEncryptionKey: primaryEncryptionKey,
		db:                   db,
		mo:                   mo,
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
