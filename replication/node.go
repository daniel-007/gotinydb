package replication

import (
	"math/big"

	"github.com/hashicorp/raft"
	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

type (
	Node struct {
		Echo        *echo.Echo
		Certificate *securelink.Certificate

		Raft     *raft.Raft
		raftChan chan<- bool

		Address, Port string
	}
)

func NewNode(address, port string, raftStore RaftStore) (*Node, error) {
	n := new(Node)
	err := n.startRaft(raftStore)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (n *Node) GetToken() (string, error) {
	return n.Certificate.GetToken(n.Port)
}

func (n *Node) GetID() *big.Int {
	return n.Certificate.Cert.SerialNumber
}

func (n *Node) RaftTransport() {
}
