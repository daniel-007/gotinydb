package replication

import (
	"fmt"
	"time"

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
		WriteHardState(raftpb.HardState)
		WriteSnapShot(raftpb.Snapshot)
		WriteEntry(raftpb.Entry)
	}
)

func NewRaft(id uint64, peers []raft.Peer) *raftNode {
	storage := raft.NewMemoryStorage()

	rNode := raft.StartNode(&raft.Config{
		ID:              id,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         storage,
		MaxSizePerMsg:   4096,
		MaxInflightMsgs: 256,
	}, peers)

	r := &raftNode{
		Node:          rNode,
		MemoryStorage: storage,
	}

	// go r.raftLoop()
	go r.raftLoop()

	return r
}

// func (r *raftNode) raftLoop() {
// 	var prev raftpb.HardState
// 	for {
// 		// Ready blocks until there is new state ready.
// 		rd := <-r.Node.Ready()
// 		if !r.isHardStateEqual(prev, rd.HardState) {
// 			r.saveStateToDisk(rd.HardState)
// 			prev = rd.HardState
// 		}

// 		r.saveToDisk(rd.Entries)
// 		go r.applyToStore(rd.CommittedEntries)
// 		r.sendMessages(rd.Messages)
// 	}
// }

func (r *raftNode) raftLoop() {
	for {
		select {
		case <-time.Tick(time.Millisecond * 100):
			r.Node.Tick()
		case rd := <-r.Node.Ready():
			r.saveToStorage(rd.HardState, rd.Entries, rd.Snapshot)
			r.sendMessages(rd.Messages)
			if !raft.IsEmptySnap(rd.Snapshot) {
				r.processSnapshot(rd.Snapshot)
			}
			for _, entry := range rd.CommittedEntries {
				r.processEntry(entry)
				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					cc.Unmarshal(entry.Data)
					r.Node.ApplyConfChange(cc)
				}
			}
			r.Node.Advance()
		}
	}
}

func (r *raftNode) Close() {
	r.Node.Stop()
}

func (r *raftNode) saveToStorage(hardState raftpb.HardState, entries []raftpb.Entry, snapshot raftpb.Snapshot) {
	fmt.Println("saveToStorage")
}

func (r *raftNode) sendMessages(messages []raftpb.Message) {
	fmt.Println("sendMessages")
	for i, message := range messages {
		fmt.Println("m", i, message)
	}
}

func (r *raftNode) processSnapshot(snapshot raftpb.Snapshot) {
	fmt.Println("processSnapshot")
}

func (r *raftNode) processEntry(entry raftpb.Entry) {
	fmt.Println("processEntry")
}

// func (r *raftNode) isHardStateEqual(previous, actual raftpb.HardState) bool {
// 	fmt.Println("isHardStateEqual")
// 	return false
// }

// func (r *raftNode) saveStateToDisk(raftpb.HardState) {
// 	fmt.Println("saveStateToDisk")
// }

// func (r *raftNode) saveToDisk([]raftpb.Entry) {
// 	fmt.Println("saveToDisk")
// }

// func (r *raftNode) applyToStore([]raftpb.Entry) {
// 	fmt.Println("applyToStore")
// }

// func (r *raftNode) sendMessages([]raftpb.Message) {
// 	fmt.Println("sendMessages")
// }
