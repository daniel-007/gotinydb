package replication

import (
	"fmt"
	"net"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/hashicorp/raft"
)

type (
	RaftStore interface {
		raft.StableStore
		raft.LogStore
		// raft.SnapshotStore
	}

	raftTransport struct {
		*securelink.Server
		acceptChan chan *securelink.TransportConn
	}
)

func GetRaftConfig(id string, notifyChan chan bool) *raft.Config {
	return &raft.Config{
		ProtocolVersion:    raft.ProtocolVersionMax,
		HeartbeatTimeout:   time.Second * 10,
		ElectionTimeout:    time.Second * 10,
		CommitTimeout:      time.Second * 5,
		MaxAppendEntries:   500,
		ShutdownOnRemove:   true,
		TrailingLogs:       1000,
		SnapshotInterval:   time.Second * 20,
		SnapshotThreshold:  100,
		LeaderLeaseTimeout: time.Second * 5,
		StartAsLeader:      false,
		LocalID:            raft.ServerID(id),
		NotifyCh:           notifyChan,
	}
}

func (n *Node) startRaft(raftStore RaftStore, bootstrap bool) (err error) {
	n.RaftChan = make(chan bool, 10)
	raftConfig := GetRaftConfig(n.GetID().String(), n.RaftChan)

	err = raft.ValidateConfig(raftConfig)
	if err != nil {
		return err
	}

	tr := raft.NewNetworkTransport(n.getRaftTransport(), 10, time.Second*2, nil)

	if bootstrap {
		servers := raft.Configuration{
			Servers: []raft.Server{
				n.raftConfig,
			},
		}

		// if hosts == nil || len(hosts) <= 0 {
		raftConfig.StartAsLeader = true
		// } else {
		// 	servers.Servers = hosts
		// }

		// fmt.Println("init raft", servers.Servers)
		// fmt.Println("hosts", hosts)

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

func (rt *raftTransport) Accept() (net.Conn, error) {
	conn, ok := <-rt.acceptChan
	if !ok {
		return nil, fmt.Errorf("server looks closes")
	}
	return conn, conn.Error()
}

func (rt *raftTransport) Handle(conn *securelink.TransportConn) error {
	rt.acceptChan <- conn

	conn.Wait()

	fmt.Println("handle DONE")

	return conn.Error()
}

func (rt *raftTransport) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	// var retAddr string
	// addr := string(address)

	// host, _, err := net.SplitHostPort(string(address))
	// if err != nil {
	// 	fmt.Println("err", err)
	// 	return nil, err
	// }

	// fmt.Println("host", host)
	// if addrType := net.ParseIP(host); addrType == nil {
	// 	fmt.Println("addrType", addrType, addr)
	// 	retAddr = fmt.Sprintf("raft.%s", addr)
	// } else {
	// 	fmt.Println("addrType bad", addrType, addr)
	// 	retAddr = addr
	// }

	// fmt.Println("ret", retAddr)

	// return rt.Server.Dial(
	// 	retAddr,
	// 	timeout,
	// )

	return rt.Server.Dial(
		string(address),
		"raft",
		timeout,
	)
}
