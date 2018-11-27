package replication

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/hashicorp/raft"
)

type (
	Transport struct {
		acceptChan chan *transportConn
		addr       net.Addr

		cert *securelink.Certificate

		getIDFromAddr func(addr raft.ServerAddress) (serverID raft.ServerID)
	}

	transportConn struct {
		wg   sync.WaitGroup
		conn net.Conn
	}
)

func (t *Transport) Handle(conn net.Conn) (err error) {
	err = fmt.Errorf("channel looks close from sender")

	if t.acceptChan == nil {
		return err
	}

	tc := newTransportConn(conn)

	tc.wg.Add(1)
	t.acceptChan <- tc

	fmt.Println("handler wait")

	tc.wg.Wait()

	fmt.Println("handler free")

	return nil
}

func (t *Transport) Accept() (net.Conn, error) {
	tc, ok := <-t.acceptChan
	if !ok {
		return nil, fmt.Errorf("channel looks closed from receiver")
	}

	return tc, nil
}

func (t *Transport) Addr() net.Addr {
	return t.addr
}

func (t *Transport) Close() error {
	if t.acceptChan != nil {
		close(t.acceptChan)
		t.acceptChan = nil
	}
	return nil
}

func (t *Transport) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	hostName := t.getIDFromAddr(addr)

	// host, _, err := net.SplitHostPort(string(addr))
	// if err != nil {
	// 	return nil, err
	// }

	// t.

	// hostWithPort := fmt.Sprintf("%s:%s", host, port)
	requestedHostName := fmt.Sprintf("%s.%s", "raft", hostName)
	tlsConfig := securelink.GetBaseTLSConfig(requestedHostName, t.cert)

	// var conn *tls.Conn
	conn, err := tls.Dial("tcp", string(addr), tlsConfig)
	if err != nil {
		return nil, err
	}
	err = conn.Handshake()
	if err != nil {
		return nil, err
	}

	return conn, err

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
	return &transportConn{
		wg:   sync.WaitGroup{},
		conn: conn,
	}
}

func (tc *transportConn) Read(b []byte) (n int, err error) {
	n, err = tc.conn.Read(b)
	// fmt.Println("tc read", n, len(b), string(b[:n]))
	return n, err
}

func (tc *transportConn) Write(b []byte) (n int, err error) {
	// fmt.Println("tc Write", len(b), string(b))
	return tc.conn.Write(b)
}

func (tc *transportConn) Close() error {
	fmt.Println("close")
	tc.wg.Done()
	return tc.conn.Close()
}

func (tc *transportConn) LocalAddr() net.Addr {
	return tc.conn.LocalAddr()
}

func (tc *transportConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

func (tc *transportConn) SetDeadline(t time.Time) error {
	return tc.conn.SetDeadline(t)
}

func (tc *transportConn) SetReadDeadline(t time.Time) error {
	return tc.conn.SetReadDeadline(t)
}

func (tc *transportConn) SetWriteDeadline(t time.Time) error {
	return tc.conn.SetWriteDeadline(t)
}
