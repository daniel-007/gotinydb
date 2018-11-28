package securelink

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type (
	// Handler provides a way to use multiple handlers inside a sign TLS listener.
	// You specify the TLS certificate for server but the same certificate is used in case
	// of Dial.
	Handler struct {
		// addr net.Addr
		name string

		// Certificate    *Certificate
		handleFunction HandlerFunction

		matchFunction ServiceMatch

		// listener net.Listener
	}

	transportConn struct {
		Error  error
		wg     sync.WaitGroup
		conn   net.Conn
		closed bool
	}
)

func NewHandler(name string, serviceMatchFunc ServiceMatch, handlerFunction HandlerFunction) *Handler {
	// func NewServiceHandler(listener net.Listener, certificate *Certificate, getRemoteIDFromAddressFunc func(raft.ServerAddress) (serverID raft.ServerID)) *ServiceHandler {
	return &Handler{
		name:           name,
		handleFunction: handlerFunction,
		matchFunction:  serviceMatchFunc,
	}
}

func (t *Handler) Handle(conn net.Conn) (err error) {
	if t.handleFunction == nil {
		return fmt.Errorf("no handler registered")
	}

	tc := newTransportConn(conn)

	return t.handleFunction(tc)
}

// func (t *ServiceHandler) Accept() (net.Conn, error) {
// 	conn, err := t.listener.Accept()
// 	if err != nil {
// 		return nil, err
// 	}

// 	tc := newTransportConn(conn)

// 	return tc, tc.Error
// }

// func (t *ServiceHandler) Addr() net.Addr {
// 	return t.listener.Addr()
// }

// func (t *ServiceHandler) Close() error {
// 	return t.listener.Close()
// }

// func (t *Server) Dial(addr string, timeout time.Duration) (net.Conn, error) {
// 	hostName := t.getCertHostNameFromAddr(addr)

// 	tlsConfig := GetBaseTLSConfig(string(hostName), t.Certificate)

// 	conn, err := tls.Dial("tcp", string(addr), tlsConfig)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = conn.SetDeadline(time.Now().Add(timeout))
// 	if err != nil {
// 		return nil, err
// 	}

// 	tc := newTransportConn(conn)

// 	return tc, err
// }

func newTransportConn(conn net.Conn) *transportConn {
	tc := &transportConn{
		wg:   sync.WaitGroup{},
		conn: conn,
	}

	tc.wg.Add(1)

	return tc
}

func (tc *transportConn) Read(b []byte) (n int, err error) {
	n, err = tc.conn.Read(b)
	tc.Error = err

	return
}

func (tc *transportConn) Write(b []byte) (n int, err error) {
	n, err = tc.conn.Write(b)
	tc.Error = err

	return
}

func (tc *transportConn) Close() (err error) {
	if tc.closed {
		return tc.Error
	}

	tc.closed = true

	err = tc.conn.Close()
	tc.Error = err
	tc.wg.Done()
	return
}

func (tc *transportConn) LocalAddr() net.Addr {
	return tc.conn.LocalAddr()
}

func (tc *transportConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

func (tc *transportConn) SetDeadline(t time.Time) (err error) {
	err = tc.conn.SetDeadline(t)
	tc.Error = err
	return
}

func (tc *transportConn) SetReadDeadline(t time.Time) (err error) {
	err = tc.conn.SetReadDeadline(t)
	tc.Error = err
	return
}

func (tc *transportConn) SetWriteDeadline(t time.Time) (err error) {
	err = tc.conn.SetWriteDeadline(t)
	tc.Error = err
	return
}
