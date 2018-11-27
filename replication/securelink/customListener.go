package securelink

import (
	"crypto/tls"
	"fmt"
	"net"
)

type (
	// Listener defines a struct to manage multiple services behind a sign TLS server.
	// It bases the service definition on the tls serverName client is asking for.
	Listener struct {
		tlsListener net.Listener
		services    []*serviceRaw
	}

	// ServiceMatch is a simple function type which based on a string tells if
	// the match is true or not
	ServiceMatch func(serverName string) (match bool)

	// Handler defines a interface to have other services after the TLS handshake.
	// Register a new a new service which implement the Handler interface to mix
	// HTTP with other protocols.
	Handler interface {
		Handle(net.Conn) error
	}

	serviceRaw struct {
		name          string
		handler       Handler
		matchFunction ServiceMatch
	}
)

// NewListener builds a new Listener with the given tls listener
func NewListener(tlsListener net.Listener) *Listener {
	cl := new(Listener)
	cl.tlsListener = tlsListener
	return cl
}

// Accept implements the net.Listener interface
func (l *Listener) Accept() (net.Conn, error) {
	fnErr := func(conn net.Conn, err error) (net.Conn, error) {
		fmt.Println("print error from (l *Listener) Accept()", err, conn)
		if conn != nil {
			conn.Close()
			return conn, nil
		}
		return conn, nil
	}

	conn, err := l.tlsListener.Accept()
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

	for _, service := range l.services {
		if service.matchFunction(tlsConn.ConnectionState().ServerName) {
			err := service.handler.Handle(tlsConn)
			if err != nil {
				return fnErr(conn, err)
			}

			return tlsConn, nil
		}
	}

	return tlsConn, nil
}

// Close implements the net.Listener interface
func (l *Listener) Close() error {
	return l.tlsListener.Close()
}

// Addr implements the net.Listener interface
func (l *Listener) Addr() net.Addr {
	return l.tlsListener.Addr()
}

// RegisterService adds a new service with it's associated math function
func (l *Listener) RegisterService(name string, serviceMatchFunc ServiceMatch, handler Handler) {
	sr := new(serviceRaw)
	sr.name = name
	sr.handler = handler
	sr.matchFunction = serviceMatchFunc

	l.services = append(l.services, sr)

	return
}

// DeregisterService removes a service base on the index
func (l *Listener) DeregisterService(name string) {
	for i, service := range l.services {
		if service.name == name {
			copy(l.services[i:], l.services[i+1:])
			l.services[len(l.services)-1] = nil // or the zero value of T
			l.services = l.services[:len(l.services)-1]
		}
	}
}
