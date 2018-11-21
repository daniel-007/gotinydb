package replication

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"

	"github.com/coreos/etcd/raft"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/muesli/cache2go"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"

	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/alexandrestein/gotinydb/replication/securelink/securecache"
)

type (
	// Replication define the replication environment
	Replication interface {
		GetMaster() MasterNode
		GetNodes() []Node

		ChangeMaster(id string)
	}

	replication struct {
		Master Node
		Nodes  []Node
	}

	// Node defines the interface used to manage nodes
	Node interface {
		GetID() *big.Int
		GetAddresses() []string
		GetPort() string

		GetCert() *securelink.Certificate
		UpdateCert(*securelink.Certificate)

		// GetServer() *securelink.Server
		Start() error

		Close() error

		addToRaftCluster(cli *http.Client, path string) error
	}

	// MasterNode is almost equal to Node but specific to master
	MasterNode interface {
		Node

		GetCA() *securelink.Certificate

		GetToken() (string, error)
		VerifyToken(serialized string) bool
	}

	node struct {
		*nodeExport
		Echo        *echo.Echo
		Certificate *securelink.Certificate

		// Server *securelink.Server

		Raft *raftNode
		// waitingRequest     []string
		// outGoingConnection map[string]*http.Client
	}

	nodeExport struct {
		ID        *big.Int
		Addresses []string
		Port      string
		IsMaster  bool

		// // Those parametiers are used for new token
		// RequestID     string `json:",omitempty"`
		// CertSignature []byte `json:",omitempty"`
	}

	NewConnectionRequest struct {
		ID              string
		IssuerID        string
		IssuerAddresses []string
		IssuerPort      string
		CACertSignature []byte
	}
)

func newNode(certificate *securelink.Certificate, port string) (*node, error) {
	id := certificate.Cert.SerialNumber

	// server, err := securelink.NewServer(certificate, port)
	// if err != nil {
	// 	return nil, err
	// }

	// var addresses []string
	addresses, err := common.GetAddresses()
	// addresses, err = server.GetAddresses()
	if err != nil {
		return nil, err
	}

	// link := &securecache.SavedPeer{
	// 	Addrs: addresses,
	// 	Port:  port,
	// }
	// securecache.PeersTable.Add(id.Int64(), time.Hour*24*365*10, link)

	// peers := securecache.GetPeers()

	e := echo.New()
	e.Logger.SetLevel(log.OFF)

	// peers := []raft.Peer{raft.Peer{ID: id.Uint64()}}
	// fmt.Println("peers", peers)

	n := &node{
		nodeExport: &nodeExport{
			ID:        id,
			Addresses: addresses,
			Port:      port,
		},
		Certificate: certificate,
		// Server:      server,
		Echo: e,

		// Raft: NewRaft(id.Uint64(), peers),
		// waitingRequest:     []string{},
		// outGoingConnection: map[string]*http.Client{},
	}

	return n, nil
}

func NewNode(certificate *securelink.Certificate, port string) (Node, error) {
	n, err := newNode(certificate, port)
	if err != nil {
		return nil, err
	}
	n.IsMaster = false
	n.Raft = NewRaft(n.ID.Uint64(), nil)

	return Node(n), nil
}

func NewMasterNode(certificate *securelink.Certificate, port string) (MasterNode, error) {
	n, err := newNode(certificate, port)
	if err != nil {
		return nil, err
	}
	n.IsMaster = true

	peers := []raft.Peer{raft.Peer{ID: n.ID.Uint64()}}
	n.Raft = NewRaft(n.ID.Uint64(), peers)

	return MasterNode(n), nil
}

// func getAddresses() ([]string, error) {
// 	interfaces, err := net.Interfaces()
// 	if err != nil {
// 		return nil, err
// 	}

// 	ret := []string{}

// 	for _, nic := range interfaces {
// 		var addrs []net.Addr
// 		addrs, err = nic.Addrs()
// 		if err != nil {
// 			return nil, err
// 		}

// 		for _, addr := range addrs {
// 			ipAsString := addr.String()
// 			ip, _, err := net.ParseCIDR(ipAsString)
// 			if err != nil {
// 				continue
// 			}

// 			// If ip accessible from outside
// 			if ip.IsGlobalUnicast() {
// 				ret = append(ret, ip.String())
// 			}
// 		}
// 	}

// 	return ret, nil
// }

func (n *node) GetID() *big.Int {
	return n.ID
}

func (n *node) GetAddresses() []string {
	addrs, _ := common.GetAddresses()
	return addrs
}

func (n *node) GetCert() *securelink.Certificate {
	return n.Certificate
}

func (n *node) UpdateCert(newCert *securelink.Certificate) {
	n.Certificate = newCert
}

func (n *node) GetCA() *securelink.Certificate {
	if !n.IsMaster {
		return nil
	}

	if !n.Certificate.IsCA {
		return nil
	}

	return n.Certificate
}

func (n *node) GetPort() string {
	return n.Port
}

func (n *node) Close() error {
	n.Raft.Close()
	return n.Echo.Close()
}

// func (n *node) GetToken() (string, error) {
// 	return n.GetToken()
// }

// func (n *node) VerifyToken(token string) bool {
// 	return n.VerifyToken(token)
// }

// Start starts the HTTP and TLS servers
func (n *node) Start() error {
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{n.Certificate.GetTLSCertificate()},
		ClientCAs:    n.Certificate.CertPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}
	n.Echo.TLSServer.TLSConfig = serverTLSConfig

	tlsListener, err := tls.Listen("tcp", n.Port, serverTLSConfig)
	if err != nil {
		return err
	}

	n.settupHandlers()

	return n.Echo.TLSServer.Serve(tlsListener)
}

func (n *node) GetToken() (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES384, Key: n.Certificate.PrivateKey}, nil)
	if err != nil {
		return "", err
	}

	reqID, token := n.buildNewConnectionRequest()

	object, err := signer.Sign(token)
	if err != nil {
		return "", err
	}

	serialized, err := object.CompactSerialize()
	if err != nil {
		return "", err
	}

	securecache.WaitingRequestTable.Add(reqID, securecache.CacheValueWaitingRequestsTimeOut, object.Signatures[0].Signature)

	return serialized, nil
}

func (n *node) ReadToken(token string, verify bool) (_ *NewConnectionRequest, signature []byte, _ error) {
	object, err := jose.ParseSigned(token)
	if err != nil {
		return nil, nil, err
	}

	var output []byte
	if verify {
		output, err = object.Verify(n.Certificate.PrivateKey.Public())
		if err != nil {
			return nil, nil, err
		}
	} else {
		output = object.UnsafePayloadWithoutVerification()
	}

	signature = object.Signatures[0].Signature

	values := new(NewConnectionRequest)
	err = json.Unmarshal(output, values)
	if err != nil {
		return nil, nil, err
	}

	if values.ID == "" {
		return nil, nil, fmt.Errorf("the request token does not containe any ID")
	}

	return values, signature, nil
}

func (n *node) VerifyToken(token string) bool {
	values, signature, err := n.ReadToken(token, true)
	if err != nil {
		return false
	}

	// cache := cache2go.Cache(CacheValueWaitingRequestsTable)
	var res *cache2go.CacheItem
	res, err = securecache.WaitingRequestTable.Value(values.ID)
	if err != nil {
		return false
	}

	if fmt.Sprintf("%x", res.Data().([]byte)) == fmt.Sprintf("%x", signature) {
		securecache.WaitingRequestTable.Delete(values.ID)
		return true
	}

	return false
}

func Connect(token, localPort string) (Node, error) {
	n := new(node)
	values, _, err := n.ReadToken(token, false)
	if err != nil {
		return nil, err
	}

	ok := false
	var cert *securelink.Certificate
	var usedAddress string
	for _, add := range values.IssuerAddresses {
		ct, retAdd, err := GetClientCertificate(add, values.IssuerPort, token, values)
		// ct, retAdd, err := getClientCertificate(add, values.IssuerPort, token, values)
		if err != nil {
			continue
		}

		ok = true
		cert = ct
		usedAddress = retAdd
		break
	}
	if !ok {
		return nil, fmt.Errorf("can't get certificate")
	}

	var n2 Node
	n2, err = NewNode(cert, localPort)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("https://%s%s/%s/%s", usedAddress, values.IssuerPort, APIVersion, PostConnectNodePATH)
	connector := securelink.NewConnector(values.IssuerID, cert)
	err = n2.addToRaftCluster(connector, path)
	if err != nil {
		return nil, err
	}

	// cache := cache2go.Cache(CacheValueConnectionsTable)
	// var resp *http.Response

	// cache.Add(n2.GetID(), time.hou)

	return n2, err
}

func (n *node) addToRaftCluster(cli *http.Client, path string) error {
	fmt.Println("path", path)

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(n.nodeExport)
	if err != nil {
		return err
	}

	var resp *http.Response
	resp, err = cli.Post(path, "application/json", buffer)
	if err != nil {
		return err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("body", string(body))

	return nil
}

// func addPeerToList(cli *http.Client, path string) error {
// 	// fmt.Println("path", path)
// 	resp, err := cli.Get(path)
// 	if err != nil {
// 		return err
// 	}

// 	var body []byte
// 	body, err = ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return err
// 	}

// 	link := &securecache.SavedPeer{}
// 	err = json.Unmarshal(body, link)
// 	if err != nil {
// 		return err
// 	}

// 	securecache.PeersTable.Add(resp.TLS.PeerCertificates[0].SerialNumber.Int64(), time.Hour*24*365*10, link)

// 	return nil
// }

// func getClientCertificate(address, port, tokenString string, token *newConnectionRequest) (cert *securelink.Certificate, usedAddress string, _ error) {
// 	ip := net.ParseIP(address)
// 	if to4 := ip.To4(); to4 == nil {
// 		address = "[" + address + "]"
// 	}

// 	path := fmt.Sprintf("https://%s%s/%s/%s", address, port, APIVersion, PostCertificatePATH)
// 	// fmt.Println("path", path)

// 	insecureClient := &http.Client{
// 		Transport: &http.Transport{
// 			TLSClientConfig: &tls.Config{
// 				ServerName:         token.ID,
// 				InsecureSkipVerify: true,
// 			},
// 		},
// 	}

// 	data := url.Values{}
// 	data.Set("token", tokenString)

// 	resp, err := insecureClient.PostForm(path, data)
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	if caSign, tokenSign := fmt.Sprintf("%x", resp.TLS.PeerCertificates[0].Signature), fmt.Sprintf("%x", token.CACertSignature); caSign != tokenSign {
// 		return nil, "", fmt.Errorf("the signature from the token is not equal to the server certificate \n\t%q \n\t%q", caSign, tokenSign)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, "", fmt.Errorf("respond status is not 200 but %d: %q", resp.StatusCode, resp.Status)
// 	}

// 	buffer := make([]byte, 1000*1000) //1MB
// 	var nb int
// 	nb, err = io.ReadFull(resp.Body, buffer)
// 	if err != nil {
// 		if err != io.EOF && err != io.ErrUnexpectedEOF {
// 			return nil, "", err
// 		}
// 	}
// 	buffer = buffer[:nb]

// 	var certificate *securelink.Certificate
// 	certificate, err = securelink.Unmarshal(buffer)
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	return certificate, address, nil
// }

func (n *node) buildNewConnectionRequest() (requestID string, reqAsJSON []byte) {
	certSignature := make([]byte, len(n.Certificate.Cert.Signature))
	copy(certSignature, n.Certificate.Cert.Signature)

	addresses, _ := common.GetAddresses()

	req := &NewConnectionRequest{
		ID:              uuid.NewV4().String(),
		IssuerID:        n.ID.String(),
		IssuerPort:      n.Port,
		IssuerAddresses: addresses,
		CACertSignature: certSignature,
	}

	reqAsJSON, _ = json.Marshal(req)
	return req.ID, reqAsJSON
}

func GetClientCertificate(address, port, tokenString string, token *NewConnectionRequest) (cert *securelink.Certificate, usedAddress string, _ error) {
	// ip := net.ParseIP(address)
	// if to4 := ip.To4(); to4 == nil {
	// 	address = "[" + address + "]"
	// }

	path := fmt.Sprintf("https://%s%s/%s/%s", address, port, APIVersion, PostCertificatePATH)
	// fmt.Println("path", path)

	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:         token.ID,
				InsecureSkipVerify: true,
			},
		},
	}

	data := url.Values{}
	data.Set("token", tokenString)

	resp, err := insecureClient.PostForm(path, data)
	if err != nil {
		return nil, "", err
	}

	if caSign, tokenSign := fmt.Sprintf("%x", resp.TLS.PeerCertificates[0].Signature), fmt.Sprintf("%x", token.CACertSignature); caSign != tokenSign {
		return nil, "", fmt.Errorf("the signature from the token is not equal to the server certificate \n\t%q \n\t%q", caSign, tokenSign)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("respond status is not 200 but %d: %q", resp.StatusCode, resp.Status)
	}

	buffer := make([]byte, 1000*1000) //1MB
	var nb int
	nb, err = io.ReadFull(resp.Body, buffer)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, "", err
		}
	}
	buffer = buffer[:nb]

	var certificate *securelink.Certificate
	certificate, err = securelink.Unmarshal(buffer)
	if err != nil {
		return nil, "", err
	}

	return certificate, address, nil
}

// // Start starts the HTTP and TLS servers
// func (n *node) Start() error {
// 	serverTLSConfig := &tls.Config{
// 		Certificates: []tls.Certificate{n.Certificate.GetTLSCertificate()},
// 		ClientCAs:    n.Certificate.CertPool,
// 		// ClientAuth:   tls.RequireAndVerifyClientCert,
// 		ClientAuth: tls.VerifyClientCertIfGiven,
// 	}
// 	n.Echo.TLSServer.TLSConfig = serverTLSConfig

// 	tlsListener, err := tls.Listen("tcp", n.Port, serverTLSConfig)
// 	if err != nil {
// 		return err
// 	}
// }
