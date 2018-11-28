package securelink_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/hashicorp/raft"
	"github.com/labstack/echo"
)

func TestTransport(t *testing.T) {
	ca, _ := securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "ca", "*.ca")
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "node", "*.node")

	tlsListener, err := tls.Listen("tcp", ":3468", securelink.GetBaseTLSConfig("ca", ca))
	if err != nil {
		t.Fatal(err)
	}

	getRemoteAddressFunc := func(addr raft.ServerAddress) (serverID raft.ServerID) {
		return raft.ServerID("ca")
	}
	tr := securelink.NewTransport(tlsListener, cert, getRemoteAddressFunc)
	tr.HandleFunction = handle

	cl := securelink.NewListener(tlsListener)
	cl.RegisterService("direct", func(serverName string) bool {
		if serverName == "ca" {
			return true
		}

		return false
	}, tr)

	echo := echo.New()
	echo.Listener = cl
	echo.TLSListener = cl
	echo.HideBanner = true
	echo.HidePort = true

	httpServer := &http.Server{}

	go func() {
		err := echo.StartServer(httpServer)
		t.Fatal(err)
	}()

	// Connect and close directly
	var conn net.Conn
	conn, err = tr.Dial(":3468", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Connect, send a small message and read the response
	conn, err = tr.Dial(":3468", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Is writing
	buff := []byte("Hello Server!!!")

	if testing.Verbose() {
		t.Logf("the client is writing %d bytes: %q", len(buff), string(buff))
	}

	var n int
	n, err = conn.Write(buff)
	if err != nil {
		t.Fatal(err)
	}

	buff = make([]byte, 150)
	n, err = conn.Read(buff)
	if err != nil {
		t.Fatal(err)
	}
	buff = buff[:n]

	if testing.Verbose() {
		t.Logf("the client has read %d bytes from server: %s", n, string(buff))
	}

	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func handle(conn net.Conn) error {
	buff := make([]byte, 4096)
	n, err := conn.Read(buff)
	if err != nil && err != io.EOF {
		return err
	}

	buff = buff[:n]

	_, err = conn.Write(
		[]byte(
			fmt.Sprintf(
				"Hi client, nice to se you.\nYour previous message was %q",
				string(buff),
			),
		),
	)

	return err
}
