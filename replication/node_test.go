package replication_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/common"

	"github.com/alexandrestein/gotinydb"

	"github.com/alexandrestein/gotinydb/replication"
	"github.com/alexandrestein/gotinydb/replication/securelink"
)

var (
	ca *securelink.Certificate
)

func init() {
	ca, _ = securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "ca")
}

func TestNode(t *testing.T) {
	dbPath := os.TempDir() + "/testRaftNode"
	defer os.RemoveAll(dbPath)

	addrs, err := common.NewAddr(1254)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("addrs", addrs.String())
	fmt.Println("addrs", addrs.SwitchMain(2))
	fmt.Println("addrs", addrs.Addrs)

	var db *gotinydb.DB
	db, err = gotinydb.Open(dbPath, [32]byte{})
	rs := db.GetRaftStore()

	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "node")

	var n *replication.Node
	n, err = replication.NewNode(addrs, rs, dbPath+"/raftStore", cert, true)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("n", n)
}
