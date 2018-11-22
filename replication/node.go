package replication

import (
	"math/big"
	"net/url"

	"github.com/hashicorp/raft"
	"github.com/labstack/echo"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

type (
	Node struct {
		Echo        *echo.Echo
		Certificate *securelink.Certificate
		Raft        *raft.Raft

		Port string
	}
)

func (n *Node) GetToken() (string, error) {
	data := url.Values{}
	data.Set("port", n.Port)
	return n.Certificate.GetToken(data)
}

func (n *Node) GetID() *big.Int {
	return n.Certificate.Cert.SerialNumber
}
