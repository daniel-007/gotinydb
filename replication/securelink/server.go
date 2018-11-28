package securelink

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo"
)

type (
	Server struct {
		TLSListener net.Listener
		Certificate *Certificate
		TLSConfig   *tls.Config
		Echo        *echo.Echo
		Handlers    []*Handler

		getHostNameFromAddr func(addr string) (hostName string)
	}
)

func NewServer(addr string, tlsConfig *tls.Config, cert *Certificate, getHostNameFromAddr func(string) string) (*Server, error) {
	tlsListener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	// getRemoteAddressFunc := func(addr raft.ServerAddress) (serverID raft.ServerID) {
	// 	return raft.ServerID("ca")
	// }

	// cl := NewListener(tlsListener)
	// cl.RegisterService("direct", func(serverName string) bool {
	// 	if serverName == "ca" {
	// 		return true
	// 	}

	// 	return false
	// }, tr)

	s := &Server{
		TLSListener: tlsListener,
		Certificate: cert,
		TLSConfig:   tlsConfig,
		Handlers:    []*Handler{},

		getHostNameFromAddr: getHostNameFromAddr,
	}

	httpServer := &http.Server{}

	e := echo.New()
	e.Listener = s
	e.TLSListener = s
	e.Server = httpServer
	e.TLSServer = httpServer
	e.HideBanner = true
	e.HidePort = true

	s.Echo = e

	return s, nil
}

func (s *Server) Start() error {
	return s.Echo.StartServer(s.Echo.TLSServer)
}

// Accept implements the net.Listener interface
func (s *Server) Accept() (net.Conn, error) {
	fnErr := func(conn net.Conn, err error) (net.Conn, error) {
		fmt.Println("print error from (s *Server) Accept()", err)
		if conn != nil {
			conn.Close()
			return conn, nil
		}
		return conn, nil
	}

	conn, err := s.TLSListener.Accept()
	if err != nil {
		return nil, err
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return fnErr(conn, fmt.Errorf("the connection is not TLS"))
	}

	err = tlsConn.Handshake()
	if err != nil {
		return fnErr(conn, err)
	}

	for _, service := range s.Handlers {
		if service.matchFunction(tlsConn.ConnectionState().ServerName) {
			err = service.Handle(tlsConn)
			if err != nil {
				return fnErr(conn, fmt.Errorf("during handle function: %s", err.Error()))
			}

			return tlsConn, nil
		}
	}

	return tlsConn, nil
}

// Close implements the net.Listener interface
func (s *Server) Close() error {
	return s.TLSListener.Close()
}

// Addr implements the net.Listener interface
func (s *Server) Addr() net.Addr {
	return s.TLSListener.Addr()
}

// RegisterService adds a new service with it's associated math function
func (s *Server) RegisterService(handler *Handler) {
	// sr := new(serviceRaw)
	// sr.name = name
	// sr.handler = handler
	// sr.matchFunction = serviceMatchFunc

	s.Handlers = append(s.Handlers, handler)

	return
}

// DeregisterService removes a service base on the index
func (s *Server) DeregisterService(name string) {
	for i, service := range s.Handlers {
		if service.name == name {
			copy(s.Handlers[i:], s.Handlers[i+1:])
			s.Handlers[len(s.Handlers)-1] = nil // or the zero value of T
			s.Handlers = s.Handlers[:len(s.Handlers)-1]
		}
	}
}

func (t *Server) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	hostName := t.getHostNameFromAddr(addr)

	tlsConfig := GetBaseTLSConfig(hostName, t.Certificate)

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	tc := newTransportConn(conn)

	return tc, err
}
