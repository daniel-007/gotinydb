package replication_test

import (
	"fmt"
	"testing"

	"github.com/alexandrestein/gotinydb/replication"
)

func TestNode(t *testing.T) {
	n, err := replication.NewNode()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("n", n)
}
