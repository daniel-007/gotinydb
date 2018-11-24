package replication

import (
	"time"

	"github.com/hashicorp/raft"
)

type (
	RaftStore interface {
		raft.StableStore
		raft.LogStore
		// raft.SnapshotStore
	}
)

func (n *Node) startRaft(raftStore RaftStore, bootstrap bool) (err error) {
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

	err = raft.ValidateConfig(raftConfig)
	if err != nil {
		return err
	}

	tr := raft.NewNetworkTransport(n.raftTransport, 10, time.Second*2, nil)

	if bootstrap {
		servers := raft.Configuration{
			Servers: []raft.Server{
				n.raftConfig,
			},
		}

		raftConfig.StartAsLeader = true

		err = raft.BootstrapCluster(raftConfig, raftStore, raftStore, n.raftFileSnapshotStore, tr, servers)
		if err != nil {
			return err
		}

	}

	n.Raft, err = raft.NewRaft(raftConfig, nil, raftStore, raftStore, n.raftFileSnapshotStore, tr)
	if err != nil {
		return err
	}

	return err
}
