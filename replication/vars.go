package replication

import (
	"regexp"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

// Those variables defines the default certificates key algorithm and key size
var (
	DefaultCertKeyAlgorithm = securelink.KeyTypeEc
	DefaultCertKeyLength    = securelink.KeyLengthEc384
)

var (
	// CheckRaftHostRequestReg is used to check if the client is looking for the raft
	// service inside the TLS service
	CheckRaftHostRequestReg = regexp.MustCompile("^raft\\.")
)
