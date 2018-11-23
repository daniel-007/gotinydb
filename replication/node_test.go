package replication_test

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

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

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("addrs", addrs)

	var db *gotinydb.DB
	db, err = gotinydb.Open(dbPath, [32]byte{})
	rs := db.GetRaftStore()

	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "node")

	var n *replication.Node
	n, err = replication.NewNode(addrs[0], rs, cert)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("n", n)
}
