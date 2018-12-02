package echohandler

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"

	"github.com/labstack/echo"
)

type (
	Handler struct {
		name       string
		Echo       *echo.Echo
		Listener   *Listener
		httpServer *http.Server
		matchReg   *regexp.Regexp
	}

	Listener struct {
		addr       net.Addr
		acceptChan chan net.Conn
	}
)

func New(addr net.Addr, name string, tlsConfig *tls.Config) (*Handler, error) {
	rg, err := regexp.Compile(
		fmt.Sprintf("^%s\\.", name),
	)
	if err != nil {
		return nil, err
	}

	li := &Listener{
		acceptChan: make(chan net.Conn),
		addr:       addr,
	}

	httpServer := new(http.Server)
	httpServer.TLSConfig = tlsConfig
	httpServer.Addr = addr.String()

	e := echo.New()
	e.TLSListener = li
	// e.Listener = li

	return &Handler{
		name:       name,
		Echo:       e,
		Listener:   li,
		matchReg:   rg,
		httpServer: httpServer,
	}, nil
}

func (h *Handler) Start() error {
	// err := http2.ConfigureServer(h.httpServer, nil)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println("sss", h.Echo.Server.)
	return h.Echo.StartServer(h.httpServer)
	// return h.Echo.StartServer(h.httpServer)
	// return h.Echo.Server.ServeTLS(h.Echo.TLSListener, "", "")
}

func (h *Handler) Handle(conn net.Conn) error {
	fmt.Println("handle")
	h.Listener.acceptChan <- conn
	// fmt.Println("close")
	// conn.Close()
	return nil
}

func (h *Handler) Match(serverName string) bool {
	return h.matchReg.MatchString(serverName)
}

func (h *Handler) Name() string {
	return h.name
}

func (l *Listener) Accept() (net.Conn, error) {
	fmt.Println("accept")
	conn := <-l.acceptChan
	return conn, nil
}
func (l *Listener) Close() error {
	return nil
}
func (l *Listener) Addr() net.Addr {
	return l.addr
}
