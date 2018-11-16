package replication_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication"
	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestNodes(t *testing.T) {
	ca, _ := securelink.NewCA(time.Hour, "ca")
	// server1Cert, _ := ca.NewCert(time.Hour, "server1")

	masterNode, err := replication.NewMasterNode(ca, ":1323")
	if err != nil {
		t.Fatal(err)
	}

	// var node1 replication.Node
	// node1, err = replication.NewNode(server1Cert)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	fmt.Println("show addresses", masterNode.GetAddresses())
	fmt.Println(masterNode.GetPort())
}
