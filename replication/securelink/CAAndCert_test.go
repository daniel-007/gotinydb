package securelink_test

import (
	"crypto/tls"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestNewCA(t *testing.T) {
	tests := []struct {
		Name   string
		Type   securelink.KeyType
		Length securelink.KeyLength
		Long   bool
		Error  bool
	}{
		{"EC 256", securelink.KeyTypeEc, securelink.KeyLengthEc256, false, false},
		{"EC 384", securelink.KeyTypeEc, securelink.KeyLengthEc384, false, false},
		{"EC 521", securelink.KeyTypeEc, securelink.KeyLengthEc521, false, false},

		{"RSA 2048", securelink.KeyTypeRSA, securelink.KeyLengthRsa2048, false, false},
		{"RSA 3072", securelink.KeyTypeRSA, securelink.KeyLengthRsa3072, true, false},
		{"RSA 4096", securelink.KeyTypeRSA, securelink.KeyLengthRsa4096, true, false},
		{"RSA 8192", securelink.KeyTypeRSA, securelink.KeyLengthRsa8192, true, false},

		{"not valid", securelink.KeyTypeRSA, securelink.KeyLengthEc256, false, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Long && testing.Short() {
				t.SkipNow()
			}

			ca, err := securelink.NewCA(test.Type, test.Length, time.Hour, "ca")
			if err != nil {
				if test.Error {
					return
				}
				t.Fatal(err)
			}

			listen(t, ca)

			runClient(t, ca)
		})

	}
}

func listen(t *testing.T, ca *securelink.Certificate) {
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{ca.GetTLSCertificate()},
		ClientCAs:    ca.GetCertPool(),
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	listener, err := tls.Listen("tcp", ":1323", serverTLSConfig)
	if err != nil {
		t.Fatal(err)
	}

	go func(listener net.Listener) {
		netConn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}

		readBuffer := make([]byte, 1000)
		var n, n2 int
		n, err = netConn.Read(readBuffer)
		if err != nil {
			t.Fatal(err)
		}

		readBuffer = readBuffer[:n]

		n2, err = netConn.Write(readBuffer)
		if err != nil {
			t.Fatal(err)
		}

		if n != n2 {
			t.Fatal("the read and write length are not equal", n, n2)
		}
	}(listener)
}

func runClient(t *testing.T, ca *securelink.Certificate) {
	cert, err := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Minute, "client")
	if err != nil {
		t.Fatal(err)
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      ca.GetCertPool(),
		Certificates: []tls.Certificate{cert.GetTLSCertificate()},
		ServerName:   "ca",
	}

	var netConn *tls.Conn
	netConn, err = tls.Dial("tcp", "localhost:1323", clientTLSConfig)
	if err != nil {
		t.Fatal(err)
	}

	var n, n2 int
	n, err = netConn.Write([]byte("HELLO"))
	if err != nil {
		t.Fatal(err)
	}

	readBuff := make([]byte, n)
	n2, err = netConn.Read(readBuff)
	if err != nil {
		t.Fatal(err)
	}

	if n != n2 {
		t.Fatal("the returned content is not the same length as the sent one", n2, n)
	}

	if string(readBuff) != "HELLO" {
		t.Fatal("the returned content is not the same as the sent one", string(readBuff), "HELLO")
	}
}

func TestCertificateMarshaling(t *testing.T) {
	ca, _ := securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, "ca")

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

			cert, _ := ca.NewCert(test.Type, test.Length, time.Hour, "node1")

			asBytes := cert.Marshal()

			cert2, err := securelink.Unmarshal(asBytes)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(cert, cert2) {
				t.Fatalf("certificates are not equal\n%v\n%v", cert, cert2)
			}
		})
	}

}
