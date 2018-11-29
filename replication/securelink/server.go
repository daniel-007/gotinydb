package securelink

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/labstack/echo"
)

type (
	Server struct {
		AddrStruct  *common.Addr
		TLSListener net.Listener
		Certificate *Certificate
		TLSConfig   *tls.Config
		Echo        *echo.Echo
		Handlers    []*Handler

		getHostNameFromAddr FuncGetHostNameFromAddr
	}
)

func NewServer(port uint16, tlsConfig *tls.Config, cert *Certificate, getHostNameFromAddr FuncGetHostNameFromAddr) (*Server, error) {
	addr, err := common.NewAddr(port)
	if err != nil {
		return nil, err
	}

	var tlsListener net.Listener
	tlsListener, err = tls.Listen("tcp", addr.String(), tlsConfig)
	if err != nil {
		return nil, err
	}

	s := &Server{
		AddrStruct:  addr,
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
	// return s.TLSListener.
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

	if tlsConn.ConnectionState().ServerName != "" {
		for _, service := range s.Handlers {
			if service.matchFunction(tlsConn.ConnectionState().ServerName) {
				fmt.Println("handle", s.Certificate.ID().String())
				err = service.Handle(tlsConn)
				if err != nil {
					return fnErr(conn, fmt.Errorf("during handle function: %s", err.Error()))
				}

				return tlsConn, nil
			}
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
	return s.AddrStruct
}

// RegisterService adds a new service with it's associated math function
func (s *Server) RegisterService(handler *Handler) {
	s.Handlers = append(s.Handlers, handler)
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

func (s *Server) Dial(addr, hostNamePrefix string, timeout time.Duration) (net.Conn, error) {
	hostName := s.getHostNameFromAddr(addr)

	if hostNamePrefix != "" {
		hostName = fmt.Sprintf("%s.%s", hostNamePrefix, hostName)
	}

	tlsConfig := GetBaseTLSConfig(hostName, s.Certificate)

	fmt.Println("dial from", s.Certificate.ID().String(), "to", hostName)
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	tc := newTransportConn(conn)

	return tc, nil
}
