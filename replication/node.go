package replication

import (
	"fmt"

	"github.com/alexandrestein/gotinydb/replication/securelink"
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
		GetID() string
		GetAddresses() []string
		GetPort() string

		GetCert() *securelink.Certificate
		UpdateCert(*securelink.Certificate)

		GetServer() *securelink.Server

		Close() error
	}

	// MasterNode is almost equal to Node but specific to master
	MasterNode interface {
		Node

		GetCA() *securelink.CA

		GetToken() (string, error)
		VerifyToken(serialized string) bool
	}

	node struct {
		*nodeExport
		Certificate *securelink.Certificate

		Server *securelink.Server

		raft *raftNode
		// waitingRequest     []string
		// outGoingConnection map[string]*http.Client
	}

	nodeExport struct {
		ID        string
		UintID    uint64
		Addresses []string
		Port      string
		IsMaster  bool

		// // Those parametiers are used for new token
		// RequestID     string `json:",omitempty"`
		// CertSignature []byte `json:",omitempty"`
	}
)

func newNode(certificate *securelink.Certificate, port string) (*node, error) {
	id := certificate.Cert.SerialNumber.String()

	server, err := securelink.NewServer(certificate, port)
	if err != nil {
		return nil, err
	}

	var addresses []string
	addresses, err = server.GetAddresses()
	if err != nil {
		return nil, err
	}

	n := &node{
		nodeExport: &nodeExport{
			ID:        id,
			Addresses: addresses,
			Port:      port,
		},
		Certificate: certificate,
		Server:      server,

		raft: NewRaft(certificate.Cert.SerialNumber.Uint64(), nil),
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

	return Node(n), nil
}

func NewMasterNode(certificate *securelink.CA, port string) (MasterNode, error) {
	n, err := newNode(certificate.Certificate, port)
	if err != nil {
		return nil, err
	}
	n.IsMaster = true

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

func (n *node) GetID() string {
	return n.ID
}

func (n *node) GetAddresses() []string {
	addrs, _ := n.Server.GetAddresses()
	return addrs
}

func (n *node) GetCert() *securelink.Certificate {
	return n.Certificate
}

func (n *node) UpdateCert(newCert *securelink.Certificate) {
	n.Certificate = newCert
}

func (n *node) GetCA() *securelink.CA {
	if !n.IsMaster {
		return nil
	}

	if !n.Certificate.IsCA {
		return nil
	}

	return &securelink.CA{
		Certificate: n.Certificate,
	}
}

func (n *node) GetPort() string {
	return n.Port
}

func (n *node) GetServer() *securelink.Server {
	return n.Server
}

func (n *node) Close() error {
	n.raft.Close()
	return n.Server.Close()
}

func (n *node) GetToken() (string, error) {
	return n.Server.GetToken()
}

// func (n *node) GetToken() (string, error) {
// 	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES384, Key: n.Certificate.PrivateKey}, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	reqID, token := n.buildServerInfoForToken()

// 	object, err := signer.Sign(token)
// 	if err != nil {
// 		return "", err
// 	}

// 	serialized, err := object.CompactSerialize()
// 	if err != nil {
// 		return "", err
// 	}

// 	cache := cache2go.Cache(CacheValueWaitingRequestsTable)
// 	cache.Add(reqID, CacheValueWaitingRequestsTimeOut, object.Signatures[0].Signature)

// 	return serialized, nil
// }

// func (n *node) readToken(token string, verify bool) (_ *newConnectionRequest, signature []byte, _ error) {
// 	object, err := jose.ParseSigned(token)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	var output []byte
// 	if verify {
// 		output, err = object.Verify(n.Certificate.PrivateKey.Public())
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	} else {
// 		output = object.UnsafePayloadWithoutVerification()
// 	}

// 	signature = object.Signatures[0].Signature

// 	values := new(newConnectionRequest)
// 	err = json.Unmarshal(output, values)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if values.ID == "" {
// 		return nil, nil, fmt.Errorf("the request token does not containe any ID")
// 	}

// 	return values, signature, nil
// }

func (n *node) VerifyToken(token string) bool {
	return n.Server.VerifyToken(token)
}

// func (n *node) VerifyToken(token string) bool {
// 	values, signature, err := n.readToken(token, true)
// 	if err != nil {
// 		return false
// 	}

// 	cache := cache2go.Cache(CacheValueWaitingRequestsTable)
// 	var res *cache2go.CacheItem
// 	res, err = cache.Value(values.ID)
// 	if err != nil {
// 		return false
// 	}

// 	if fmt.Sprintf("%x", res.Data().([]byte)) == fmt.Sprintf("%x", signature) {
// 		cache.Delete(values.ID)
// 		return true
// 	}

// 	return false
// }

func Connect(token, localPort string) (Node, error) {
	n := new(node)
	values, _, err := n.Server.ReadToken(token, false)
	if err != nil {
		return nil, err
	}

	ok := false
	var cert *securelink.Certificate
	// var usedAddress string
	for _, add := range values.IssuerAddresses {
		ct, _, err := securelink.GetClientCertificate(add, values.IssuerPort, token, values)
		// ct, retAdd, err := getClientCertificate(add, values.IssuerPort, token, values)
		if err != nil {
			continue
		}

		ok = true
		cert = ct
		// usedAddress = retAdd
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

	// path := fmt.Sprintf("https://%s%s/%s/%s", usedAddress, values.Port, APIVersion, GetClusterMapPATH)
	// // fmt.Println("path", path)

	// connector := securelink.NewConnector(values.ID, cert)

	// cache := cache2go.Cache(CacheValueConnectionsTable)
	// var resp *http.Response
	// resp, err = connector.Get(path)
	// if err != nil {
	// 	return nil, err
	// }

	// cache.Add(n2.GetID(), time.hou)

	return n2, err
}

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
