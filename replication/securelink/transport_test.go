package securelink_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestTransport(t *testing.T) {
	ca, _ := securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "ca", "*.ca")

	getHostNameFunc := func(addr string) (serverID string) {
		return "ca"
	}
	s, err := securelink.NewServer(":3468", securelink.GetBaseTLSConfig("ca", ca), ca, getHostNameFunc)
	if err != nil {
		t.Fatal(err)
	}

	handler := securelink.NewHandler("direct", func(s string) bool { return true }, handle)
	s.RegisterService(handler)

	go func() {
		err := s.Start()
		t.Fatal(err)
	}()

	// Connect and close directly
	var conn net.Conn
	conn, err = s.Dial(":3468", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Connect, send a small message and read the response
	conn, err = s.Dial(":3468", time.Second)
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
