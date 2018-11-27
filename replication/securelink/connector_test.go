package securelink_test

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/blake2b"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

var (
	tlsListener net.Listener
	httpServer  *http.Server
	cl          *securelink.Listener
)

type (
	handler struct{}
)

func (h *handler) Handle(conn net.Conn) error {
	buffer := make([]byte, 1000*10)
	n, err := conn.Read(buffer)
	if err != nil {
		return err
	}

	buffer = buffer[:n]

	var n2 int
	n2, err = conn.Write(buffer)
	if err != nil {
		return err
	}

	if n != n2 {
		return fmt.Errorf("the read and write length are different: %d %d", n, n2)
	}

	return nil
}

func TestConnector(t *testing.T) {
	ca, _ := securelink.NewCA(
		securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"ca",
	)

	t.Run("base", func(t *testing.T) {
		testBase(t, ca)
	})
	t.Run("wildcard", func(t *testing.T) {
		testWildcardCertAndDifferentHandler(t, ca)
	})
	t.Run("concurrent", func(t *testing.T) {
		testConcurantDirectAccess(t, ca)
	})
}

func runSever(t *testing.T, cert *securelink.Certificate, wg *sync.WaitGroup) {
	tlsConfig := securelink.GetBaseTLSConfig("node", cert)

	var err error
	tlsListener, err = tls.Listen("tcp", ":6246", tlsConfig)
	if err != nil {
		t.Fatal(err)
	}

	httpHandler := http.NewServeMux()
	httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	httpServer = &http.Server{
		TLSConfig: tlsConfig,
		Handler:   httpHandler,
	}

	cl = securelink.NewListener(tlsListener)
	cl.RegisterService("directTLS", func(serverName string) bool {
		if serverName == "test.node" {
			cl.Addr()
			return true
		}

		return false
	}, new(handler))

	wg.Done()
	httpServer.Serve(cl)
}

func testBase(t *testing.T, ca *securelink.Certificate) {
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"node",
	)
	cliCert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"cli",
	)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go runSever(t, cert, &wg)
	wg.Wait()

	cli := securelink.NewConnector("node", cliCert)
	_, err := cli.Get("https://127.0.0.1:6246/")
	if err != nil {
		t.Fatal(err)
	}

	cli = securelink.NewConnector("node", cliCert)
	_, err = cli.Get("https://127.0.0.1:6246/")
	if err != nil {
		t.Fatal(err)
	}

	err = httpServer.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func testWildcardCertAndDifferentHandler(t *testing.T, ca *securelink.Certificate) {
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"*.node",
	)
	cliCert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"cli",
	)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go runSever(t, cert, &wg)
	wg.Wait()

	// Access directly to the TLS stream with the hander for the sub domain test.node
	conn, err := tls.Dial("tcp", "127.0.0.1:6246", securelink.GetBaseTLSConfig("test.node", cliCert))
	if err != nil {
		t.Fatal(err)
	}

	writeBuff := []byte("HELLO Wold!!!")
	var n int
	n, err = conn.Write(writeBuff)
	if err != nil {
		t.Fatal(err)
	}

	buff := make([]byte, n)
	var n2 int
	n2, err = conn.Read(buff)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	if n != n2 {
		t.Fatalf("the write and read length are different: %d %d", n, n2)
	}

	if s1, s2 := string(writeBuff), string(buff); s1 != s2 {
		t.Fatalf("the write and read are different: %s %s", s1, s2)
	}

	// Deregister the direct TLS handler
	cl.DeregisterService("directTLS")

	// use the same sub domain to access default HTTP interface
	cli := securelink.NewConnector("test.node", cliCert)
	_, err = cli.Get("https://127.0.0.1:6246/")
	if err != nil {
		t.Fatal(err)
	}
	err = httpServer.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func testConcurantDirectAccess(t *testing.T, ca *securelink.Certificate) {
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"*.node",
	)
	cliCert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"cli",
	)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go runSever(t, cert, &wg)
	wg.Wait()

	randBuffers := make([][]byte, 20)
	for i, buffer := range randBuffers {
		buffer = make([]byte, 1000)
		n, err := rand.Read(buffer)
		if err != nil {
			t.Fatal(err)
		}

		buffer = buffer[:n]
		randBuffers[i] = buffer
	}

	nTest := 500
	for index := 0; index < nTest; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Access directly to the TLS stream with the hander for the sub domain test.node
			conn, err := tls.Dial("tcp", "127.0.0.1:6246", securelink.GetBaseTLSConfig("test.node", cliCert))
			if err != nil {
				t.Fatal(err)
			}

			writeBuff := make([]byte, 1000)
			copy(writeBuff, randBuffers[index%10])

			var n int
			n, err = conn.Write(writeBuff)
			if err != nil {
				t.Fatal(err)
			}

			buff := make([]byte, n)
			var n2 int
			n2, err = conn.Read(buff)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
			err = conn.Close()
			if err != nil {
				t.Fatal(err)
			}

			if n != n2 {
				t.Fatalf("the write and read length are different: %d %d", n, n2)
			}

			if s1, s2 := blake2b.Sum256(writeBuff), blake2b.Sum256(buff); !reflect.DeepEqual(s1, s2) {
				t.Fatalf("the write and read are different: %x %x", s1, s2)
			}
		}(index)
	}

	wg.Wait()
}
