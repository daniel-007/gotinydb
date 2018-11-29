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
		handleFunction FuncHandler

		matchFunction FuncServiceMatch

		// listener net.Listener
	}

	TransportConn struct {
		err    error
		wg     sync.WaitGroup
		conn   net.Conn
		closed bool
	}
)

// NewHandler builds a new Hanlder pointer to use in a server object
func NewHandler(name string, serviceMatchFunc FuncServiceMatch, handlerFunction FuncHandler) *Handler {
	return &Handler{
		name:           name,
		handleFunction: handlerFunction,
		matchFunction:  serviceMatchFunc,
	}
}

// Handle is called when a client connect to the server and the client point to the service.
func (t *Handler) Handle(conn net.Conn) (err error) {
	if t.handleFunction == nil {
		return fmt.Errorf("no handler registered")
	}

	tc := newTransportConn(conn)

	return t.handleFunction(tc)
}

// func (t *Handler) Accept() (net.Conn, error) {
// 	// conn, err := t.listener.Accept()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// tc := newTransportConn(conn)

// 	// return tc, tc.Error

// 	return nil, fmt.Errorf("(t *Handler) Accept() (net.Conn, error) accept not implemented")
// }

// func (t *Handler) Addr() net.Addr {
// 	// return t.listener.Addr()
// 	fmt.Println("(t *Handler) Addr() net.Addr accept not implemented")
// 	return nil
// }

// func (t *Handler) Close() error {
// 	// return t.listener.Close()
// 	return fmt.Errorf("(t *Handler) Close() error accept not implemented")
// }

// func (s *Server) Dial(addr string, timeout time.Duration) (net.Conn, error) {
// 	hostName := s.getHostNameFromAddr(addr)

// 	tlsConfig := GetBaseTLSConfig(string(hostName), s.Certificate)

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

func newTransportConn(conn net.Conn) *TransportConn {
	tc := &TransportConn{
		wg:   sync.WaitGroup{},
		conn: conn,
	}

	tc.wg.Add(1)

	return tc
}

func (tc *TransportConn) Read(b []byte) (n int, err error) {
	n, err = tc.conn.Read(b)
	tc.err = err

	fmt.Printf("read %p %d %d %v\n", tc, len(b), n, err)
	// fmt.Printf("read %p %d %d %v\n\t%s\n\n", tc, len(b), n, err, string(b))

	return
}

func (tc *TransportConn) Write(b []byte) (n int, err error) {
	n, err = tc.conn.Write(b)
	tc.err = err

	fmt.Printf("write %p %d %d %v\n", tc, len(b), n, err)
	// fmt.Printf("write %p %d %d %v\n\t%s\n\n", tc, len(b), n, err, string(b))

	return
}

func (tc *TransportConn) Close() (err error) {
	fmt.Println("close")

	if tc.closed {
		return tc.err
	}

	tc.closed = true

	err = tc.conn.Close()
	tc.err = err
	tc.wg.Done()
	return
}

func (tc *TransportConn) Wait() {
	tc.wg.Wait()
}

// func (tc *TransportConn) Done() {
// 	tc.wg.Done()
// }

func (tc *TransportConn) LocalAddr() net.Addr {
	return tc.conn.LocalAddr()
}

func (tc *TransportConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

func (tc *TransportConn) SetDeadline(t time.Time) (err error) {
	err = tc.conn.SetDeadline(t)
	tc.err = err
	return
}

func (tc *TransportConn) SetReadDeadline(t time.Time) (err error) {
	err = tc.conn.SetReadDeadline(t)
	tc.err = err
	return
}

func (tc *TransportConn) SetWriteDeadline(t time.Time) (err error) {
	err = tc.conn.SetWriteDeadline(t)
	tc.err = err
	return
}

func (tc *TransportConn) Error() error {
	return tc.err
}
