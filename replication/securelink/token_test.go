package securelink_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestToken(t *testing.T) {
	tokenValues := url.Values{}
	tokenValues.Set("port", ":1323")

	tests := []struct {
		Name   string
		Type   securelink.KeyType
		Length securelink.KeyLength
		Long   bool
	}{
		{"EC 256", securelink.KeyTypeEc, securelink.KeyLengthEc256, false},
		{"EC 384", securelink.KeyTypeEc, securelink.KeyLengthEc384, false},
		{"EC 521", securelink.KeyTypeEc, securelink.KeyLengthEc521, false},

		{"RSA 2048", securelink.KeyTypeRSA, securelink.KeyLengthRsa2048, false},
		{"RSA 3072", securelink.KeyTypeRSA, securelink.KeyLengthRsa3072, true},
		{"RSA 4096", securelink.KeyTypeRSA, securelink.KeyLengthRsa4096, true},
		{"RSA 8192", securelink.KeyTypeRSA, securelink.KeyLengthRsa8192, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Long && testing.Short() {
				t.SkipNow()
			}

			ca, err := securelink.NewCA(test.Type, test.Length, time.Hour, "ca")
			if err != nil {
				t.Fatal(err)
			}

			var token string
			token, err = ca.GetToken(tokenValues)
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

			tokenObject, _, _ := ca.ReadToken(token, false)
			if tokenObject.Values.Get("port") != tokenValues.Get("port") {
				t.Fatalf("base value must be equal to restored value but not\n\t%s\n\t%s", tokenObject.Values.Get("port"), tokenValues.Get("port"))
			}

		})
	}
}
