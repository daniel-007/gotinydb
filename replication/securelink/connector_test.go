package securelink_test

import (
	"crypto/tls"
	"fmt"
	"html"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestConnector(t *testing.T) {
	ca, _ := securelink.NewCA(
		securelink.KeyTypeEc, securelink.KeyLengthEc256,
		time.Hour,
		securelink.GetCertTemplate(nil, nil),
		"ca",
	)

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
	go func() {
		listener, err := tls.Listen("tcp", ":6246", securelink.GetBaseTLSConfig("node", cert))
		if err != nil {
			t.Fatal(err)
		}
		wg.Done()

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
			defer listener.Close()
		})
		http.Serve(listener, nil)
	}()
	wg.Wait()

	cli := securelink.NewConnector("node", cliCert)
	_, err := cli.Get("https://127.0.0.1:6246/")
	if err != nil {
		t.Fatal(err)
	}
}
