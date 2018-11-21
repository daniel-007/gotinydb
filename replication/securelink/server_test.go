package securelink_test

import (
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestToken(t *testing.T) {
	ca, err := securelink.NewCA(time.Hour, "ca")
	if err != nil {
		t.Fatal(err)
	}

	var token string
	token, err = ca.GetToken("issuer ID", ":3654")
	if err != nil {
		t.Fatal(err)
	}

	if !ca.VerifyToken(token) {
		t.Fatalf("the token must be valid but it's not")
	}

	if ca.VerifyToken(token) {
		t.Fatalf("the token must be expired but it's not")
	}

	token = token[:16] + "e" + token[17:]
	if ca.VerifyToken(token) {
		t.Fatalf("the token must be invalid but it is valid")
	}
}
