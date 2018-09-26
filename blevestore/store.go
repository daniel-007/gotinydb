package blevestore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dgraph-io/badger"

	"github.com/blevesearch/bleve/index/store"
	"github.com/blevesearch/bleve/registry"
)

const (
	Name                    = "internal"
	defaultCompactBatchSize = 100
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
	fmt.Println("called")
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

func (bs *Store) Stats() json.Marshaler {
	return &stats{
		s: bs,
	}
}

// CompactWithBatchSize removes DictionaryTerm entries with a count of zero (in batchSize batches)
// Removing entries is a workaround for github issue #374.
func (bs *Store) CompactWithBatchSize(batchSize int) error {
	for {
		cnt := 0
		err := bs.db.Update(func(txn *badger.Txn) error {
			iter := txn.NewIterator(badger.DefaultIteratorOptions)
			// c := tx.Bucket([]byte(bs.bucket)).Cursor()
			prefix := []byte("d")

			// for k, v := iter.Seek(bs.buildID(prefix)); iter.ValidForPrefix(prefix); k, v = iter.Next() {
			for iter.Seek(bs.buildID(prefix)); iter.ValidForPrefix(prefix); iter.Next() {
				item := iter.Item()

				var k, v []byte
				item.KeyCopy(k)
				_, err := item.ValueCopy(v)
				if err != nil {
					return err
				}

				if bytes.Equal(v, []byte{0}) {
					cnt++
					if err := txn.Delete(bs.buildID(prefix)); err != nil {
						return err
					}
					if cnt == batchSize {
						break
					}
				}

			}
			return nil
		})
		if err != nil {
			return err
		}

		if cnt == 0 {
			break
		}
	}
	return nil
}

// Compact calls CompactWithBatchSize with a default batch size of 100.  This is a workaround
// for github issue #374.
func (bs *Store) Compact() error {
	return bs.CompactWithBatchSize(defaultCompactBatchSize)
}

func init() {
	registry.RegisterKVStore(Name, New)
}

func (bs *Store) buildID(key []byte) []byte {
	return append(bs.indexPrefixID, key...)
}
