package securelink_test

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

const (
	secret1 = "secret1"
	secret2 = "secret2"
)

var (
	s1, s2 *securelink.Server
	tt     *testing.T

	ca *securelink.Certificate
)

func TestTransportAndServer(t *testing.T) {
	tt = t
	ca, _ = securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "ca", "*.ca")
	cert1, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "1", "*.1")
	cert2, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "2", "*.2")

	getHostNameFunc := func(addr string) (serverID string) {
		return securelink.GetID(addr, ca)
	}

	var err error
	s1, err = securelink.NewServer(3461, securelink.GetBaseTLSConfig("1", cert1), cert1, getHostNameFunc)
	if err != nil {
		t.Fatal(err)
	}
	s2, err = securelink.NewServer(3462, securelink.GetBaseTLSConfig("2", cert2), cert2, getHostNameFunc)
	if err != nil {
		t.Fatal(err)
	}

	testPrefixFn := func(s string) bool {
		if len(s) < 4 {
			return false
		}
		if s[:4] == "test" {
			return true
		}
		return false
	}
	s1.RegisterService(securelink.NewHandler("testGroup", testPrefixFn, handle1))
	s2.RegisterService(securelink.NewHandler("testGroup", testPrefixFn, handle2))

	var conn net.Conn
	conn, err = s2.Dial(":3461", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Connect, send a small message and read the response
	conn, err = s2.Dial(":3461", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var n int
	n, err = conn.Write([]byte(secret1))
	if err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		t.Logf("the client has write %d bytes to server: %s", n, secret1)
	}

	buff := make([]byte, 150)
	n, err = conn.Read(buff)
	if err != nil {
		t.Fatal(err)
	}
	buff = buff[:n]

	if string(buff) != secret2 {
		t.Fatalf("the returned secret is not good")
	}

	if testing.Verbose() {
		t.Logf("the client has read %d bytes from server: %s", n, string(buff))
	}

	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	// t.Run("net.Listener interface", testNetListenerInterface)
	t.Run("deregister", testDeregister)
	t.Run("http fallback", httpFallback)

	err = s1.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = s2.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Accept a connection and contact the other server to get the second secret and return the second secret
// to the first one.
func handle1(connAsServer net.Conn) error {
	buf := make([]byte, 100)
	n, err := connAsServer.Read(buf)
	if err != nil {
		if err == io.EOF {
			return err
		}
		tt.Fatal(err)
	}

	cs := connAsServer.(*securelink.TransportConn).ConnectionState()

	remoteClientServerName := cs.ServerName

	var connAsClient net.Conn
	connAsClient, err = s1.Dial(":3462", "test", time.Millisecond*500)
	if err != nil {
		tt.Fatal(err)
	}
	defer connAsClient.Close()

	remoteServerServerName := cs.ServerName

	if remoteClientServerName != remoteServerServerName {
		tt.Fatalf("the connected client and the corresponding server are not corresponding %s != %s", remoteClientServerName, remoteServerServerName)
	}

	_, err = connAsClient.Write(buf[:n])
	if err != nil {
		tt.Fatal(err)
	}

	buf2 := make([]byte, 100)
	n, err = connAsClient.Read(buf2)
	if err != nil {
		tt.Fatal(err)
	}

	_, err = connAsServer.Write(buf2[:n])
	if err != nil {
		tt.Fatal(err)
	}

	return nil
}

// Check that the client sent secret one and returns secret 2
func handle2(connAsServer net.Conn) error {
	buf := make([]byte, 100)
	n, err := connAsServer.Read(buf)
	if err != nil {
		tt.Fatal(err)
	}

	if string(buf[:n]) != secret1 {
		tt.Fatalf("bad secret %s, %d", buf[:n], n)
	}

	_, err = connAsServer.Write([]byte(secret2))
	if err != nil {
		tt.Fatal(err)
	}

	return nil
}

func httpFallback(t *testing.T) {
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "cli")

	s, err := securelink.NewServer(7777, securelink.GetBaseTLSConfig("1", cert), cert, nil)
	if err != nil {
		t.Fatal(err)
	}

	s.Echo.GET("/", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	t.Fatalf("test with client")
}

func testDeregister(t *testing.T) {
	s1.DeregisterService("testGroup")

	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "cli")
	conn, err := securelink.NewServiceConnector(":3461", "test.1", cert, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 10)
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("the service must be deregister and connection should be close")
	}
}
