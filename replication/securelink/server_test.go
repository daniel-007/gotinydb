package securelink_test

import (
	"testing"
)

func TestServer(t *testing.T) {
	// ca, _ := securelink.NewCA(time.Hour*24, "ca")
	// clientCert, _ := ca.NewCert(time.Hour, "client")

	// s, err := securelink.NewServer(ca, ":1323")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// s.Echo.GET("/", func(c echo.Context) error {
	// 	return c.String(http.StatusOK, "HELLO")
	// })

	// go func(s *securelink.Server) {
	// 	err = s.Start()
	// }(s)

	// // Wait for the server to start
	// time.Sleep(time.Microsecond * 250)

	// cli := securelink.NewConnector(ca.Cert.SerialNumber.String(), clientCert)
	// var resp *http.Response
	// resp, err = cli.Get("https://localhost:1323/")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// buff := make([]byte, 1000)

	// var n int
	// n, err = io.ReadFull(resp.Body, buff)
	// if err != nil && err != io.ErrUnexpectedEOF {
	// 	t.Fatal(err)
	// }

	// if testing.Verbose() {
	// 	t.Logf("%d -> %s", n, string(buff[:n]))
	// }

	// s.Close()
}
