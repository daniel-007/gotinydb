package replication_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb"
	"github.com/alexandrestein/gotinydb/replication"
	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/hashicorp/raft"
)

var (
	ca *securelink.Certificate
)

func init() {
	ca, _ = securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "ca")
}

func TestOneNode(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	dbPath := os.TempDir() + "/testRaftNode"
	defer os.RemoveAll(dbPath)

	addrs, err := common.NewAddr(1254)
	if err != nil {
		t.Fatal(err)
	}

	var db *gotinydb.DB
	db, err = gotinydb.Open(dbPath, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	rs := db.GetRaftStore()

	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "node")

	// var n *replication.Node
	_, err = replication.NewNode(addrs, rs, dbPath+"/raftStore", cert, true)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println("n", n)
}

func TestThreeNodes(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	dbPath1 := os.TempDir() + "/testRaftNode1"
	dbPath2 := os.TempDir() + "/testRaftNode2"
	// dbPath3 := os.TempDir() + "/testRaftNode3"
	defer os.RemoveAll(dbPath1)
	defer os.RemoveAll(dbPath2)
	// defer os.RemoveAll(dbPath3)

	addrs1, _ := common.NewAddr(1251)
	addrs2, _ := common.NewAddr(1252)
	// addrs3, _ := common.NewAddr(1253)

	db1, _ := gotinydb.Open(dbPath1, [32]byte{})
	rs1 := db1.GetRaftStore()
	db2, _ := gotinydb.Open(dbPath2, [32]byte{})
	rs2 := db2.GetRaftStore()
	// db3, _ := gotinydb.Open(dbPath3, [32]byte{})
	// rs3 := db3.GetRaftStore()

	cert1, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil))
	cert2, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil))
	// cert3, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil))

	node1, err := replication.NewNode(addrs1, rs1, dbPath1+"/raftStore", cert1, true)
	if err != nil {
		t.Fatal(err)
	}

	var node2 *replication.Node
	node2, err = replication.NewNode(addrs2, rs2, dbPath2+"/raftStore", cert2, false)
	if err != nil {
		t.Fatal(err)
	}
	// node2.Raft.BootstrapCluster(
	// 	replication.GetRaftConfig(node2.GetID().String(), node2.RaftChan),
	// )

	f := node1.AddNonvoter(raft.ServerID(node2.GetID().String()), raft.ServerAddress(node2.Addr.String()))
	if err := f.Error(); err != nil {
		t.Fatal(err)
	}

	// var node3 *replication.Node
	// node3, err = replication.NewNode(addrs3, rs3, dbPath3+"/raftStore", cert3, false)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// node3.GetID().String()
	// f = node1.AddVoter(raft.ServerID(node3.GetID().String()), raft.ServerAddress(node3.Addr.String()))
	// if err := f.Error(); err != nil {
	// 	t.Fatal(err)
	// }

	fmt.Println("node2 conf", node2.Raft.GetConfiguration().Configuration().Servers)

	time.Sleep(time.Second * 25)

	fmt.Println("node2 conf", node2.Raft.GetConfiguration().Configuration().Servers)
}
