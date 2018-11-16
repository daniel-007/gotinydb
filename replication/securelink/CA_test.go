package securelink_test

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestNewCA(t *testing.T) {
	ca, err := securelink.NewCA(time.Hour, "ca")
	if err != nil {
		t.Fatal(err)
	}

	listen(t, ca)

	runClient(t, ca)
}

func listen(t *testing.T, ca *securelink.CA) {
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

func runClient(t *testing.T, ca *securelink.CA) {
	cert, err := ca.NewCert(time.Minute, "client")
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
