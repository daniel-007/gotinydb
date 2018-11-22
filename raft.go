package gotinydb

import (
	"context"

	"github.com/alexandrestein/gotinydb/transaction"
)

type (
	raftSore struct {
		*DB
	}
)

func (rs *raftSore) buildStoreKey(prefix byte, key []byte) []byte {
	return append(
		[]byte{prefixRaftStore, prefix},
		key...,
	)
}
func (rs *raftSore) buildStableStoreKey(key []byte) []byte {
	return rs.buildStoreKey(prefixRaftStableStore, key)
}
func (rs *raftSore) buildLogStoreKey(key []byte) []byte {
	return rs.buildStoreKey(prefixRaftLogStore, key)
}
func (rs *raftSore) buildSnapshotStoreKey(key []byte) []byte {
	return rs.buildStoreKey(prefixRaftSnapshotStore, key)
}

func (rs *raftSore) blockForWrite(tx *transaction.Transaction) (err error) {
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
func (rs *raftSore) Set(key []byte, val []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx := transaction.New(ctx)
	tx.AddOperation(
		transaction.NewOperation("", nil, rs.buildStableStoreKey(key), val, false, true),
	)

	return rs.blockForWrite(tx)
}

// // Get returns the value for key, or an empty byte slice if key was not found.
// // StableStore interface
// func (rs *raftSore) Get(key []byte) ([]byte, error) {
// 	storeKey := rs.buildStableStoreKey(key)

// 	// rs.badger.View(func)
// }

// // StableStore interface
// func (rs *raftSore) SetUint64(key []byte, val uint64) error {

// }

// // GetUint64 returns the uint64 value for key, or 0 if key was not found.
// // StableStore interface
// func (rs *raftSore) GetUint64(key []byte) (uint64, error) {

// }

// // FirstIndex returns the first index written. 0 for no entries.
// // LogStore
// func (rs *raftSore) FirstIndex() (uint64, error) {

// }

// // LastIndex returns the last index written. 0 for no entries.
// // LogStore
// func (rs *raftSore) LastIndex() (uint64, error) {

// }

// // GetLog gets a log entry at a given index.
// // LogStore
// func (rs *raftSore) GetLog(index uint64, log *raft.Log) error {

// }

// // StoreLog stores a log entry.
// // LogStore
// func (rs *raftSore) StoreLog(log *raft.Log) error {

// }

// // StoreLogs stores multiple log entries.
// // LogStore
// func (rs *raftSore) StoreLogs(logs []*raft.Log) error {

// }

// // DeleteRange deletes a range of log entries. The range is inclusive.
// // LogStore
// func (rs *raftSore) DeleteRange(min, max uint64) error {

// }

// // Create is used to begin a snapshot at a given index and term, and with
// // the given committed configuration. The version parameter controls
// // which snapshot version to create.
// // SnapshotStore INTERFACE
// func (rs *raftSore) Create(version raft.SnapshotVersion, index, term uint64, configuration raft.Configuration, configurationIndex uint64, trans raft.Transport) (raft.SnapshotSink, error) {

// }

// // List is used to list the available snapshots in the store.
// // It should return then in descending order, with the highest index first.
// // SnapshotStore INTERFACE
// func (rs *raftSore) List() ([]*raft.SnapshotMeta, error) {

// }

// // Open takes a snapshot ID and provides a ReadCloser. Once close is
// // called it is assumed the snapshot is no longer needed.
// // SnapshotStore INTERFACE
// func (rs *raftSore) Open(id string) (*raft.SnapshotMeta, io.ReadCloser, error) {

// }
