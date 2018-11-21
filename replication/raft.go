package replication

import (
	"context"
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

	peer struct {
		Addresses []string
		Port      string
		RaftPeer  *raft.Peer
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
	i := 0
	for {
		i++
		fmt.Println("new loop", i)
		fmt.Println("ID", r.Node.Status().ID)
		select {
		// case <-time.Tick(time.Millisecond * 100):
		// 	r.Node.Tick()
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
	fmt.Println("\thardState", hardState)
	fmt.Println("\tentries", entries)
	fmt.Println("\tsnapshot", snapshot)
}

func (r *raftNode) sendMessages(messages []raftpb.Message) {
	fmt.Println("sendMessages", messages)
	for i, message := range messages {
		fmt.Println("m", i, message)
	}
}

func (r *raftNode) processSnapshot(snapshot raftpb.Snapshot) {
	fmt.Println("processSnapshot", snapshot)
}

func (r *raftNode) processEntry(entry raftpb.Entry) {
	fmt.Println("processEntry", entry)
}

func (r *raftNode) AddNode(newNode *nodeExport) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	configChange := raftpb.ConfChange{
		Type:   raftpb.ConfChangeAddNode,
		NodeID: newNode.ID.Uint64(),
	}
	// go func() {
	// 	err := r.Node.ProposeConfChange(ctx, configChange)
	// 	fmt.Println("eresdvsdf545", err)
	// 	}()

	err := r.Node.ProposeConfChange(ctx, configChange)
	return err
}
