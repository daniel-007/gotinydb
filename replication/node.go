package replication

import (
	"net"

	"github.com/alexandrestein/gotinydb/replication/securelink"
	uuid "github.com/satori/go.uuid"
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
	}

	// MasterNode is almost equal to Node but specific to master
	MasterNode interface {
		Node

		GetCA() *securelink.CA
	}

	node struct {
		*nodeExport
		Certificate *securelink.Certificate

		Server *securelink.Server
	}

	nodeExport struct {
		ID        string
		Addresses []string
		Port      string
		IsMaster  bool
	}
)

func newNode(certificate *securelink.Certificate, port string) (*node, error) {
	id := uuid.Must(uuid.NewV4()).String()

	addresses, err := getAddresses()
	if err != nil {
		return nil, err
	}

	var server *securelink.Server
	server, err = securelink.NewServer(certificate, port)
	if err != nil {
		return nil, err
	}

	return &node{
		nodeExport: &nodeExport{
			ID:        id,
			Addresses: addresses,
			Port:      port,
		},
		Certificate: certificate,
		Server:      server,
	}, nil
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

func getAddresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ret := []string{}

	for _, nic := range interfaces {
		var addrs []net.Addr
		addrs, err = nic.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ipAsString := addr.String()
			ip, _, err := net.ParseCIDR(ipAsString)
			if err != nil {
				continue
			}

			// If ip accessible from outside
			if ip.IsGlobalUnicast() {
				ret = append(ret, ip.String())
			}
		}
	}

	return ret, nil
}

func (n *node) GetID() string {
	return n.ID
}

func (n *node) GetAddresses() []string {
	return n.Addresses
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
