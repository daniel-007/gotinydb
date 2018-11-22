package replication

import (
	"time"

	"github.com/hashicorp/raft"
)

type (
	RaftStore interface {
		raft.StableStore
		raft.LogStore
		raft.SnapshotStore
	}
)

func (n *Node) startRaft(raftStore RaftStore) (err error) {
	n.raftChan = make(chan<- bool, 10)
	raftConfig := &raft.Config{
		ProtocolVersion:    raft.ProtocolVersionMax,
		HeartbeatTimeout:   time.Second * 10,
		ElectionTimeout:    time.Second * 10,
		CommitTimeout:      time.Second * 2,
		MaxAppendEntries:   500,
		ShutdownOnRemove:   true,
		TrailingLogs:       1000,
		SnapshotInterval:   time.Minute,
		SnapshotThreshold:  100,
		LeaderLeaseTimeout: time.Second * 10,
		StartAsLeader:      false,
		LocalID:            raft.ServerID(n.GetID().String()),
		NotifyCh:           n.raftChan,
	}

	n.Raft, err = raft.NewRaft(raftConfig, nil, raftStore, raftStore, raftStore, nil)
	return err
}
