package replication

import (
	"fmt"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
)

type (
	raftNode struct {
		Node          raft.Node
		MemoryStorage *raft.MemoryStorage

		PersistantStorage raftStorage
	}

	raftStorage interface {
		Save()
	}
)

func NewRaft(id uint64, peers []raft.Peer) *raftNode {
	storage := raft.NewMemoryStorage()

	raft := raft.StartNode(&raft.Config{
		ID:              id,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         storage,
		MaxSizePerMsg:   4096,
		MaxInflightMsgs: 256,
	}, peers)

	r := &raftNode{
		Node:          raft,
		MemoryStorage: storage,
	}

	go r.raftLoop()

	return r
}

func (r *raftNode) raftLoop() {
	// var prev raftpb.HardState
	for {
		// Ready blocks until there is new state ready.
		rd := <-r.Node.Ready()
		// if !isHardStateEqual(prev, rd.HardState) {
		// 	saveStateToDisk(rd.HardState)
		// 	prev = rd.HardState
		// }

		r.saveToDisk(rd.Entries)
		go r.applyToStore(rd.CommittedEntries)
		r.sendMessages(rd.Messages)
	}
}

func (r *raftNode) Close() {
	r.Node.Stop()
}

func (r *raftNode) saveToDisk([]raftpb.Entry) {
	fmt.Println("saveToDisk")
}

func (r *raftNode) applyToStore([]raftpb.Entry) {
	fmt.Println("applyToStore")
}

func (r *raftNode) sendMessages([]raftpb.Message) {
	fmt.Println("sendMessages")
}
