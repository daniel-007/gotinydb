package securelink_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

func TestServer(t *testing.T) {
	ca, _ := securelink.NewCA(time.Hour*24, "ca")
	serverCert, _ := ca.NewCert(time.Hour, "server")
	clientCert, _ := ca.NewCert(time.Hour, "client")

	s, err := securelink.NewServer(":1323", serverCert)
	if err != nil {
		t.Fatal(err)
	}

	s.Echo.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "HELLO")
	})

	go func(s *securelink.Server) {
		err = s.Start()
		if err != nil {
			t.Fatal(err)
		}
	}(s)

	// Wait for the server to start
	time.Sleep(time.Microsecond * 100)

	cli := securelink.NewConnector("server", clientCert)
	var resp *http.Response
	resp, err = cli.Get("https://localhost:1323/")
	if err != nil {
		t.Fatal(err)
	}

	buff := make([]byte, 1000)

	var n int
	n, err = io.ReadFull(resp.Body, buff)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatal(err)
	}

	if testing.Verbose() {
		t.Logf("%d -> %s", n, string(buff[:n]))
	}
}
