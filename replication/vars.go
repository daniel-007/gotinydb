package replication

import "github.com/alexandrestein/gotinydb/replication/securelink"

// Those variables defines the default certificates key algorithm and key size
var (
	DefaultCertKeyAlgorithm securelink.KeyType   = securelink.KeyTypeEc
	DefaultCertKeyLength    securelink.KeyLength = securelink.KeyLengthEc384
)
