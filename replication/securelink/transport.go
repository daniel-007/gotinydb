package securelink

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

type (
	Transport struct {
		AcceptChan chan *transportConn
		addr       net.Addr

		Certificate    *Certificate
		HandleFunction func(conn net.Conn) (err error)

		getIDFromAddr func(addr raft.ServerAddress) (serverID raft.ServerID)

		listener net.Listener
	}

	transportConn struct {
		Error  error
		wg     sync.WaitGroup
		conn   net.Conn
		closed bool
	}
)

func NewTransport(listener net.Listener, certificate *Certificate, getRemoteIDFromAddressFunc func(raft.ServerAddress) (serverID raft.ServerID)) *Transport {
	return &Transport{
		AcceptChan:    make(chan *transportConn),
		Certificate:   certificate,
		getIDFromAddr: getRemoteIDFromAddressFunc,
		listener:      listener,
	}
}

func (t *Transport) Handle(conn net.Conn) (err error) {
	if t.HandleFunction == nil {
		return fmt.Errorf("no handler registered")
	}

	tc := newTransportConn(conn)

	return t.HandleFunction(tc)
	// fmt.Println("handle")

	// err = fmt.Errorf("channel looks close from sender")

	// if t.AcceptChan == nil {
	// 	return err
	// }

	// tc := newTransportConn(conn)

	// t.AcceptChan <- tc

	// // fmt.Println("handler wait for close")

	// tc.wg.Wait()

	// // fmt.Println("handler connection closed")

	// return tc.Error
}

func (t *Transport) accept(conn net.Conn) (net.Conn, error) {
	tc, ok := <-t.AcceptChan
	if !ok {
		return nil, fmt.Errorf("channel looks closed from receiver")
	}

	return tc, tc.Error
}

func (t *Transport) Accept() (net.Conn, error) {
	conn, err := t.listener.Accept()
	if err != nil {
		return nil, err
	}

	tc := newTransportConn(conn)

	// tc, ok := <-t.AcceptChan
	// if !ok {
	// 	return nil, fmt.Errorf("channel looks closed from receiver")
	// }

	return tc, tc.Error
}

func (t *Transport) Addr() net.Addr {
	return t.listener.Addr()
}

func (t *Transport) Close() error {
	if t.AcceptChan != nil {
		close(t.AcceptChan)
		t.AcceptChan = nil
	}

	return t.listener.Close()
}

func (t *Transport) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	hostName := t.getIDFromAddr(addr)

	// host, _, err := net.SplitHostPort(string(addr))
	// if err != nil {
	// 	return nil, err
	// }

	// t.

	// hostWithPort := fmt.Sprintf("%s:%s", host, port)
	tlsConfig := GetBaseTLSConfig(string(hostName), t.Certificate)

	// var conn *tls.Conn
	conn, err := tls.Dial("tcp", string(addr), tlsConfig)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	// err = conn.Handshake()
	// if err != nil {
	// 	return nil, err
	// }

	tc := newTransportConn(conn)

	return tc, err

	// location := &url.URL{
	// 	Scheme: "https",
	// 	Host:   string(addr),
	// 	Path:   fmt.Sprintf("/%s/%s", APIVersion, GetRaftStreamerPATH),
	// }
	// origin := &url.URL{
	// 	Scheme: "https",
	// 	Host:   string(addr),
	// 	Path:   "/",
	// }

	// wsConfig := &websocket.Config{
	// 	// A WebSocket server address.
	// 	Location: location,

	// 	// A Websocket client origin.
	// 	Origin: origin,

	// 	// WebSocket subprotocols.
	// 	Protocol: []string{""},

	// 	//  // WebSocket protocol version.
	// 	//  Version int

	// 	// TLS config for secure WebSocket (wss).
	// 	TlsConfig: tlsConfig,

	// 	//  // Additional header fields to be sent in WebSocket opening handshake.
	// 	//  Header http.Header

	// 	//  // Dialer used when opening websocket connections.
	// 	//  Dialer *net.Dialer
	// }

	// ws, err := websocket.DialConfig(wsConfig)
	// if err != nil {
	// 	return nil, err
	// }
	// return ws, nil
}

func newTransportConn(conn net.Conn) *transportConn {
	tc := &transportConn{
		wg:   sync.WaitGroup{},
		conn: conn,
	}

	tc.wg.Add(1)

	return tc
}

func (tc *transportConn) Read(b []byte) (n int, err error) {
	// tlsConn := tc.conn.(*tls.Conn)
	// fmt.Println("tlsConn", tlsConn.ConnectionState().HandshakeComplete)

	// fmt.Println("tc read -1", len(b))

	// givenBufferSize := 0
	// if len(b) < 4096 {
	// 	givenBufferSize = len(b)
	// 	b = make([]byte, 4096)
	// }

	// fmt.Println("tc read 0", len(b))
	n, err = tc.conn.Read(b)

	// if givenBufferSize != 0 {
	// 	b = b[:givenBufferSize]
	// }

	tc.Error = err
	// fmt.Println("tc read 1", n, len(b), err)
	// fmt.Println("tc read 2", string(b))
	return
}

func (tc *transportConn) Write(b []byte) (n int, err error) {
	// fmt.Println("tc write 0", len(b))

	// givenBufferSize := 0
	// if len(b) < 4096 {
	// 	givenBufferSize = len(b)
	// 	b = make([]byte, 4096)
	// }

	n, err = tc.conn.Write(b)
	tc.Error = err

	// if givenBufferSize != 0 {
	// 	b = b[:givenBufferSize]
	// 	n = givenBufferSize
	// }

	// if err != nil {
	// 	fmt.Println("Write err", err)
	// } else {
	// 	fmt.Println("written ", n)
	// }

	// fmt.Println("Write")
	// fmt.Println("tc Write", len(b), string(b))
	return
}

func (tc *transportConn) Close() (err error) {
	if tc.closed {
		return tc.Error
	}

	tc.closed = true

	// fmt.Println("close")
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
