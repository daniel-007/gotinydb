package echohandler_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"

	"github.com/alexandrestein/gotinydb/replication/securelink/handlers/echohandler"
)

func TestMain(t *testing.T) {
	ca, _ := securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "ca", "*.ca")
	cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc384, time.Hour, securelink.GetCertTemplate(nil, nil), "1", "*.1")

	addr := new(addr)
	echoService, err := echohandler.New(addr, "echo", securelink.GetBaseTLSConfig("1", cert))
	if err != nil {
		t.Fatal(err)
	}

	echoService.Echo.GET("/", func(c echo.Context) error {
		c.String(200, "OK")
		return nil
	})

	go echoService.Start()

	getNameFn := func(s string) string {
		return securelink.GetID(s, cert)
	}
	s, _ := securelink.NewServer(1364, securelink.GetBaseTLSConfig("1", cert), cert, getNameFn)

	s.RegisterService(echoService)

	time.Sleep(time.Minute)

	cli := securelink.NewHTTPSConnector("1", cert)
	resp, err := cli.Get("https://echo.localhost:1364/")
	if err != nil {
		t.Fatal(err)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		t.Logf("the server respond %q", string(buf))
	}
}

type (
	addr struct{}
)

func (a *addr) Network() string {
	return ""
}

func (a *addr) String() string {
	return ":1364"
}
