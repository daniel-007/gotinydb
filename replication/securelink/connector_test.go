package securelink_test

// import (
// 	"crypto/tls"
// 	"fmt"
// 	"html"
// 	"net"
// 	"net/http"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/alexandrestein/gotinydb/replication/securelink"
// )

// var (
// 	tlsListener net.Listener
// 	httpServer  *http.Server
// )

// type (
// 	handler struct{}
// )

// func (h *handler) Handle(conn net.Conn) error {
// 	buffer := make([]byte, 1000*10)
// 	n, err := conn.Read(buffer)
// 	if err != nil {
// 		return err
// 	}

// 	buffer = buffer[:n]

// 	var n2 int
// 	n2, err = conn.Write(buffer)
// 	if err != nil {
// 		return err
// 	}

// 	if n != n2 {
// 		return fmt.Errorf("the read and write length are different: %d %d", n, n2)
// 	}

// 	return nil
// }

// func TestConnector(t *testing.T) {
// 	ca, _ := securelink.NewCA(
// 		securelink.KeyTypeEc, securelink.KeyLengthEc256,
// 		time.Hour,
// 		securelink.GetCertTemplate(nil, nil),
// 		"ca",
// 	)

// 	t.Run("base", func(t *testing.T) {
// 		testBase(t, ca)
// 	})
// }

// func runSever(t *testing.T, cert *securelink.Certificate, wg *sync.WaitGroup) {
// 	tlsConfig := securelink.GetBaseTLSConfig("node", cert)

// 	getHostNameFunc := func(addr string) (serverID string) {
// 		return securelink.GetID(addr, ca)
// 	}

// 	var err error
// 	s1, err = securelink.NewServer(3461, securelink.GetBaseTLSConfig("1", cert1), cert1, getHostNameFunc)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	httpHandler := http.NewServeMux()
// 	httpHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
// 	})

// 	httpServer = &http.Server{
// 		TLSConfig: tlsConfig,
// 		Handler:   httpHandler,
// 	}

// 	wg.Done()
// 	httpServer.Serve(tlsListener)
// }

// func testBase(t *testing.T, ca *securelink.Certificate) {
// 	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
// 		time.Hour,
// 		securelink.GetCertTemplate(nil, nil),
// 		"node",
// 	)
// 	cliCert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256,
// 		time.Hour,
// 		securelink.GetCertTemplate(nil, nil),
// 		"cli",
// 	)

// 	wg := sync.WaitGroup{}
// 	wg.Add(1)
// 	go runSever(t, cert, &wg)
// 	wg.Wait()

// 	cli := securelink.NewHTTPSConnector("node", cliCert)
// 	_, err := cli.Get("https://127.0.0.1:6246/")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	cli = securelink.NewHTTPSConnector("node", cliCert)
// 	_, err = cli.Get("https://127.0.0.1:6246/")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = httpServer.Close()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
