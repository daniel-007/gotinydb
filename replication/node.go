package replication

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

type (
	Node struct {
		Echo        *echo.Echo
		Certificate *securelink.Certificate

		Raft                  *raft.Raft
		RaftChan              chan bool
		Server                *securelink.Server
		raftFileSnapshotStore *raft.FileSnapshotStore
		raftConfig            raft.Server

		Path string

		Addr net.Addr
		// raftTransport         *securelink.Handler
	}
)

func NewNode(addr net.Addr, raftStore RaftStore, path string, cert *securelink.Certificate, bootstrap bool) (_ *Node, err error) {
	n := new(Node)

	err = os.MkdirAll(path, 1740)
	if err != nil {
		return nil, err
	}

	// n.waitingNodes = []raft.Server{}
	n.Addr = addr
	n.Certificate = cert

	n.raftFileSnapshotStore, err = raft.NewFileSnapshotStore(path, 10, nil)
	if err != nil {
		return nil, err
	}
	n.raftConfig = raft.Server{
		Suffrage: raft.Voter,
		ID:       raft.ServerID(n.GetID().String()),
		Address:  raft.ServerAddress(n.Addr.String()),
	}

	// n.buildRaftTransport()

	err = n.startServer()
	if err != nil {
		return nil, err
	}

	err = n.startRaft(raftStore, bootstrap)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (n *Node) startServer() error {
	tlsConfig := securelink.GetBaseTLSConfig(n.GetID().String(), n.Certificate)

	// tlsListener, err := tls.Listen("tcp", fmt.Sprintf(":%s", n.GetPort()), tlsConfig)
	// if err != nil {
	// 	return err
	// }

	s, err := securelink.NewServer(n.Addr.String(), tlsConfig, n.Certificate, n.GetIDFromAddr)
	if err != nil {
		return err
	}

	isItRaftConn := func(serverName string) bool {
		if CheckRaftHostRequestReg.MatchString(serverName) {
			return true
		}

		return false
	}
	rt := n.getRaftTransport()
	handler := securelink.NewHandler("raft", isItRaftConn, rt.Handle)
	s.RegisterService(handler)

	n.Server = s

	// cl := securelink.NewListener(tlsListener)
	// cl.RegisterService("raft", , n.raftTransport)

	// n.Echo = echo.New()
	// n.Echo.Server = &http.Server{
	// 	TLSConfig: tlsConfig,
	// }
	// n.Echo.TLSListener = cl

	// n.settupHandlers()

	// httpServer := &http.Server{
	// 	TLSConfig: tlsConfig,
	// 	Handler:   nil,
	// }

	go func() {
		// err := n.Echo.StartServer(httpServer)
		err = s.Start()
		fmt.Println("merde avec le sever", err)
		log.Fatal(err)
	}()

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

// func (n *Node) buildRaftTransport() {
// 	n.raftTransport = &Transport{
// 		acceptChan:    make(chan *transportConn),
// 		addr:          n.Addr,
// 		cert:          n.Certificate,
// 		getIDFromAddr: n.getIDFromAddr,
// 	}
// }

func (n *Node) GetIDFromAddr(addr string) (serverID string) {
	for i, server := range n.Raft.GetConfiguration().Configuration().Servers {
		fmt.Println("GetIDFromAddr server i", i, server)
		if server.Address == raft.ServerAddress(addr) {
			return string(server.ID)
		}
	}

	return n.getIDFromAddrByConnecting(addr)
}

func (n *Node) getIDFromAddrByConnecting(addr string) (serverID string) {
	tlsConfig := securelink.GetBaseTLSConfig("", n.Certificate)
	tlsConfig.InsecureSkipVerify = true
	conn, err := tls.Dial("tcp", string(addr), tlsConfig)
	if err != nil {
		fmt.Println("err -1", err)
		return ""
	}

	err = conn.Handshake()
	if err != nil {
		fmt.Println("err 0", err)
		return ""
	}

	remoteCert := conn.ConnectionState().PeerCertificates[0]
	err = remoteCert.CheckSignatureFrom(n.Certificate.CACert)
	if err != nil {
		fmt.Println("err 1", err)
		return ""
	}

	return remoteCert.SerialNumber.String()
}

func (n *Node) getRaftTransport() *raftTransport {
	acceptChan := make(chan *securelink.TransportConn, 0)
	rt := &raftTransport{
		Server:     n.Server,
		acceptChan: acceptChan,
	}
	return rt
}

func (n *Node) AddVoter(serverID raft.ServerID, serverAddress raft.ServerAddress) raft.IndexFuture {
	// lastIndex := n.Raft.LastIndex()
	// return n.Raft.AddVoter(serverID, serverAddress, lastIndex, time.Second*5)
	return n.Raft.AddVoter(serverID, serverAddress, 0, time.Second*2)
}

func (n *Node) AddNonvoter(serverID raft.ServerID, serverAddress raft.ServerAddress) raft.IndexFuture {
	// lastIndex := n.Raft.LastIndex()
	// return n.Raft.AddVoter(serverID, serverAddress, lastIndex, time.Second*5)
	return n.Raft.AddNonvoter(serverID, serverAddress, 0, time.Second*2)
}

// GetPort return a string representation of the port from the address.
// It has the form of "3169".
func (n *Node) GetPort() string {
	_, port := n.getHostAndPort()
	return port
}

func (n *Node) GetHost() string {
	host, _ := n.getHostAndPort()
	return host
}

func (n *Node) getHostAndPort() (string, string) {
	host, port, err := net.SplitHostPort(n.Addr.String())
	if err != nil {
		return "", ""
	}
	return host, port
}
