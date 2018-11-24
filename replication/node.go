package replication

import (
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/hashicorp/raft"
	"github.com/labstack/echo"
	"golang.org/x/net/websocket"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

type (
	Node struct {
		Echo        *echo.Echo
		Certificate *securelink.Certificate

		Raft                  *raft.Raft
		raftChan              chan<- bool
		raftTransport         *Transport
		raftFileSnapshotStore *raft.FileSnapshotStore

		Path string

		Addr net.Addr
	}

	Transport struct {
		acceptChan chan net.Conn
		addr       net.Addr

		cert *securelink.Certificate
	}
)

func NewNode(addr net.Addr, raftStore RaftStore, path string, cert *securelink.Certificate, bootstrap bool) (_ *Node, err error) {
	n := new(Node)

	err = os.MkdirAll(path, 1740)
	if err != nil {
		return nil, err
	}

	n.raftFileSnapshotStore, err = raft.NewFileSnapshotStore(path, 10, nil)
	if err != nil {
		return nil, err
	}

	n.Addr = addr
	n.Certificate = cert

	err = n.startHTTP()
	if err != nil {
		return nil, err
	}

	n.buildRaftTransport()

	err = n.startRaft(raftStore, bootstrap)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (n *Node) startHTTP() error {
	n.Echo = echo.New()

	n.settupHandlers()

	return nil
}

func (n *Node) GetToken() (string, error) {
	_, port, err := net.SplitHostPort(n.Addr.String())
	if err != nil {
		return "", err
	}
	return n.Certificate.GetToken(port)
}

func (n *Node) GetID() *big.Int {
	return n.Certificate.Cert.SerialNumber
}

func (n *Node) buildRaftTransport() {
	ch := make(chan net.Conn)
	n.raftTransport = &Transport{
		acceptChan: ch,
		addr:       n.Addr,
		cert:       n.Certificate,
	}
}

func (n *Node) GetRaftTransport() *Transport {
	return n.raftTransport
}

func (t *Transport) Accept() (net.Conn, error) {
	err := fmt.Errorf("connection looks closed")

	if t.acceptChan == nil {
		return nil, err
	}

	conn, ok := <-t.acceptChan
	if !ok {
		return nil, err
	}
	return conn, nil
}

func (t *Transport) Close() error {
	if t.acceptChan != nil {
		close(t.acceptChan)
		t.acceptChan = nil
	}
	return nil
}

func (t *Transport) Addr() net.Addr {
	return t.addr
}

func (t *Transport) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {

	location := &url.URL{
		Scheme: "https",
		Host:   string(addr),
		Path:   fmt.Sprintf("/%s/%s", APIVersion, GetRaftStreamerPATH),
	}
	origin := &url.URL{
		Scheme: "https",
		Host:   string(addr),
		Path:   "/",
	}

	tlsConfig := securelink.GetBaseTLSConfig(string(addr), t.cert)

	wsConfig := &websocket.Config{
		// A WebSocket server address.
		Location: location,

		// A Websocket client origin.
		Origin: origin,

		// WebSocket subprotocols.
		Protocol: []string{""},

		//  // WebSocket protocol version.
		//  Version int

		// TLS config for secure WebSocket (wss).
		TlsConfig: tlsConfig,

		//  // Additional header fields to be sent in WebSocket opening handshake.
		//  Header http.Header

		//  // Dialer used when opening websocket connections.
		//  Dialer *net.Dialer
	}

	ws, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return nil, err
	}
	return ws, nil
}
