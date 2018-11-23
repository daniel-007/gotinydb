package securelink_test

import (
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
	jwt "github.com/dgrijalva/jwt-go"
)

func TestToken(t *testing.T) {
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

			certTemplate := securelink.GetCertTemplate(nil, nil)
			ca, err := securelink.NewCA(test.Type, test.Length, time.Hour, certTemplate, "ca")
			if err != nil {
				t.Fatal(err)
			}

			var token string
			token, err = ca.GetToken(":2313")
			if err != nil {
				t.Fatal(err)
			}

			var tokenObject *jwt.Token
			tokenObject, err = ca.ReadToken(token, false)
			if err != nil {
				t.Fatal(err)
			}

			if !ca.VerifyToken(token) {
				t.Fatalf("the token must be valid but it's not")
			}

			if ca.VerifyToken(token) {
				t.Fatalf("the token must be expired but it's not")
			}

			if ca.VerifyToken(token[:16] + "e" + token[17:]) {
				t.Fatalf("the token must be invalid but it is valid")
			}

			// {"alg":"none","typ":"JWT"}
			token = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0" + token[36:]
			if ca.VerifyToken(token) {
				t.Fatalf("the token must be invalid but it is valid")
			}
			// {"alg":none,"typ":"JWT"}
			token = "eyJhbGciOm5vbmUsInR5cCI6IkpXVCJ9" + token[35:]
			if ca.VerifyToken(token) {
				t.Fatalf("the token must be invalid but it is valid")
			}

			if issPort := tokenObject.Claims.(jwt.MapClaims)["issPort"].(string); issPort != ":2313" {
				t.Fatalf("base value must be equal to restored value but not\n\t%s\n\t%s", issPort, ":2313")
			}
		})
	}
}
