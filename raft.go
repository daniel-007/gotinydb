package gotinydb

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"github.com/alexandrestein/gotinydb/transaction"
	"github.com/dgraph-io/badger"
	"github.com/hashicorp/raft"
)

type (
	raftStore struct {
		*DB
	}
)

func (d *DB) startRaft() error {
	return nil
}

func (rs *raftStore) buildStoreKey(prefix byte, key []byte) []byte {
	return append(
		[]byte{prefixRaftStore, prefix},
		key...,
	)
}
func (rs *raftStore) buildStableStoreKey(key []byte) []byte {
	return rs.buildStoreKey(prefixRaftStableStore, key)
}
func (rs *raftStore) buildLogStoreKey(index uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, index)

	return rs.buildStoreKey(prefixRaftLogStore, b)
}
func (rs *raftStore) buildSnapshotStoreKey(key []byte) []byte {
	return rs.buildStoreKey(prefixRaftSnapshotStore, key)
}

func (rs *raftStore) waitForWriteIsDone(tx *transaction.Transaction) (err error) {
	// Run the insertion
	select {
	case rs.writeChan <- tx:
	case <-rs.ctx.Done():
		return rs.ctx.Err()
	}

	// And wait for the end of the insertion
	select {
	case err = <-tx.ResponseChan:
	case <-tx.Ctx.Done():
		err = tx.Ctx.Err()
	}

	return
}

// StableStore interface
func (rs *raftStore) Set(key []byte, val []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx := transaction.New(ctx)
	tx.AddOperation(
		transaction.NewOperation("", nil, rs.buildStableStoreKey(key), val, false, true),
	)

	return rs.waitForWriteIsDone(tx)
}

// Get returns the value for key, or an empty byte slice if key was not found.
// StableStore interface
func (rs *raftStore) Get(key []byte) ([]byte, error) {
	storeKey := rs.buildStableStoreKey(key)

	caller, err := rs.DB.Get(storeKey)
	if err != nil {
		return nil, err
	}

	return caller.Bytes, caller.Error
}

// StableStore interface
func (rs *raftStore) SetUint64(key []byte, val uint64) error {
	bytesAsUint64 := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesAsUint64, val)

	return rs.Set(key, bytesAsUint64)
}

// GetUint64 returns the uint64 value for key, or 0 if key was not found.
// StableStore interface
func (rs *raftStore) GetUint64(key []byte) (uint64, error) {
	bytesAsUint64, err := rs.Get(key)

	return binary.BigEndian.Uint64(bytesAsUint64), err
}

func (rs *raftStore) firstOrLastLogIndex(first bool) (i uint64, _ error) {
	baseI := uint64(0)
	reverse := false
	if !first {
		baseI = math.MaxUint64
		reverse = true
	}

	prefix := rs.buildStoreKey(prefixRaftLogStore, nil)
	firstOrLastPossibleID := rs.buildLogStoreKey(baseI)

	return i, rs.badger.View(func(txn *badger.Txn) error {
		itOptions := badger.DefaultIteratorOptions
		itOptions.PrefetchValues = false
		itOptions.AllVersions = false
		itOptions.Reverse = reverse

		it := txn.NewIterator(itOptions)
		defer it.Close()

		it.Seek(firstOrLastPossibleID)
		if !it.ValidForPrefix(prefix) {
			i = 0
			return fmt.Errorf("looks like there is no existing log")
		}

		item := it.Item()
		i = binary.BigEndian.Uint64(item.Key()[len(prefix):])

		return nil
	})
}

// FirstIndex returns the first index written. 0 for no entries.
// LogStore interface
func (rs *raftStore) FirstIndex() (i uint64, _ error) {
	return rs.firstOrLastLogIndex(true)
}

// LastIndex returns the last index written. 0 for no entries.
// LogStore interface
func (rs *raftStore) LastIndex() (uint64, error) {
	return rs.firstOrLastLogIndex(false)
}

// GetLog gets a log entry at a given index.
// LogStore interface
func (rs *raftStore) GetLog(index uint64, log *raft.Log) error {
	caller, err := rs.DB.Get(rs.buildLogStoreKey(index))
	if err != nil {
		return err
	}

	return json.Unmarshal(caller.Bytes, log)
}

// StoreLog stores a log entry.
// LogStore interface
func (rs *raftStore) StoreLog(log *raft.Log) error {
	encoded, err := json.Marshal(log)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx := transaction.New(ctx)
	tx.AddOperation(
		transaction.NewOperation("", nil, rs.buildLogStoreKey(log.Index), encoded, false, true),
	)

	return rs.waitForWriteIsDone(tx)
}

// StoreLogs stores multiple log entries.
// LogStore interface
func (rs *raftStore) StoreLogs(logs []*raft.Log) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx := transaction.New(ctx)

	for _, log := range logs {
		encoded, err := json.Marshal(log)
		if err != nil {
			return err
		}
		tx.AddOperation(
			transaction.NewOperation("", nil, rs.buildLogStoreKey(log.Index), encoded, false, true),
		)
	}

	return rs.waitForWriteIsDone(tx)
}

// DeleteRange deletes a range of log entries. The range is inclusive.
// LogStore interface
func (rs *raftStore) DeleteRange(min, max uint64) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prefix := rs.buildStoreKey(prefixRaftLogStore, nil)

	tx := transaction.New(ctx)

	err := rs.badger.View(func(txn *badger.Txn) error {
		itOptions := badger.DefaultIteratorOptions
		itOptions.PrefetchValues = false

		it := txn.NewIterator(itOptions)
		defer it.Close()

		for it.Seek(rs.buildLogStoreKey(min)); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			i := binary.BigEndian.Uint64(item.Key()[len(prefix):])
			if min <= i || i <= max {
				var keyCopy []byte
				keyCopy = item.KeyCopy(keyCopy)

				tx.AddOperation(
					transaction.NewOperation("", nil, keyCopy, nil, true, true),
				)
			} else if max < i {
				break
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return rs.waitForWriteIsDone(tx)
}

// // Create is used to begin a snapshot at a given index and term, and with
// // the given committed configuration. The version parameter controls
// // which snapshot version to create.
// // SnapshotStore interface
// func (rs *raftStore) Create(version raft.SnapshotVersion, index, term uint64, configuration raft.Configuration, configurationIndex uint64, trans raft.Transport) (raft.SnapshotSink, error) {

// }

// // List is used to list the available snapshots in the store.
// // It should return then in descending order, with the highest index first.
// // SnapshotStore interface
// func (rs *raftStore) List() ([]*raft.SnapshotMeta, error) {

// }

// // Open takes a snapshot ID and provides a ReadCloser. Once close is
// // called it is assumed the snapshot is no longer needed.
// // SnapshotStore interface
// func (rs *raftStore) Open(id string) (*raft.SnapshotMeta, io.ReadCloser, error) {

// }
