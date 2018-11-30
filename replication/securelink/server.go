package securelink

import (
	"crypto/tls"
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/alexandrestein/gotinydb/replication/common"
)

type (
	Server struct {
		// Ctx         context.Context
		AddrStruct  *common.Addr
		TLSListener net.Listener
		Certificate *Certificate
		TLSConfig   *tls.Config
		Handlers    []*Handler
		ErrChan     chan error

		getHostNameFromAddr FuncGetHostNameFromAddr
	}
)

func NewServer(port uint16, tlsConfig *tls.Config, cert *Certificate, getHostNameFromAddr FuncGetHostNameFromAddr) (*Server, error) {
	addr, err := common.NewAddr(port)
	if err != nil {
		return nil, err
	}

	var tlsListener net.Listener
	tlsListener, err = tls.Listen("tcp", addr.ForListenerBroadcast(), tlsConfig)
	if err != nil {
		return nil, err
	}

	s := &Server{
		// Ctx:         ctx,
		AddrStruct:  addr,
		TLSListener: tlsListener,
		Certificate: cert,
		TLSConfig:   tlsConfig,
		Handlers:    []*Handler{},

		getHostNameFromAddr: getHostNameFromAddr,
	}

	go func(s *Server) {
		isListenerClosedReg := regexp.MustCompile(": use of closed network connection$")
		for {
			_, err := s.Accept()
			if err != nil {
				if isListenerClosedReg.MatchString(err.Error()) {
					return
				}

				panic(err)
			}
		}
	}(s)

	return s, nil
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

	tc, _ := newTransportConn(conn, true)

	if tlsConn.ConnectionState().ServerName != "" {
		for _, service := range s.Handlers {
			if service.matchFunction(tc.ConnectionState().ServerName) {
				go service.Handle(tc)
				return nil, nil
			}
		}
	}

	tc.Close()

	return nil, nil
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

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	tc, _ := newTransportConn(conn, false)

	return tc, nil
}
