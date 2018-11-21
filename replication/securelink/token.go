package securelink

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/alexandrestein/gotinydb/replication/securelink/securecache"
	"github.com/muesli/cache2go"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
)

type (
	// Token defines sign objects which
	Token struct {
		ID              string
		IssuerID        string
		IssuerAddresses []string
		CACertSignature []byte

		Values url.Values
	}
)

func (c *Certificate) getTokenSignAlgorithm() jose.SignatureAlgorithm {
	if c.KeyPair.Type == KeyTypeRSA {
		if c.KeyPair.Length == KeyLengthRsa2048 {
			return jose.RS256
		} else if c.KeyPair.Length == KeyLengthRsa3072 {
			return jose.RS384
		} else if c.KeyPair.Length == KeyLengthRsa4096 || c.KeyPair.Length == KeyLengthRsa8192 {
			return jose.RS512
		}
	} else if c.KeyPair.Type == KeyTypeEc {
		if c.KeyPair.Length == KeyLengthEc256 {
			return jose.ES256
		} else if c.KeyPair.Length == KeyLengthEc384 {
			return jose.ES384
		} else if c.KeyPair.Length == KeyLengthEc521 {
			return jose.ES512
		}
	}
	return ""
}

// GetToken returns a string representation of a temporary token (10 minutes validity with cache2go)
func (c *Certificate) GetToken(data url.Values) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: c.getTokenSignAlgorithm(), Key: c.KeyPair.Private}, nil)
	if err != nil {
		return "", err
	}

	reqID, token := c.buildNewConnectionRequest(data)

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

// ReadToken returns a Token pointer from it's string representation
func (c *Certificate) ReadToken(token string, verify bool) (_ *Token, signature []byte, _ error) {
	object, err := jose.ParseSigned(token)
	if err != nil {
		return nil, nil, err
	}

	var output []byte
	if verify {
		output, err = object.Verify(c.KeyPair.Public)
		if err != nil {
			return nil, nil, err
		}
	} else {
		output = object.UnsafePayloadWithoutVerification()
	}

	signature = object.Signatures[0].Signature

	values := new(Token)
	err = json.Unmarshal(output, values)
	if err != nil {
		return nil, nil, err
	}

	if values.ID == "" {
		return nil, nil, fmt.Errorf("the request token does not containe any ID")
	}

	return values, signature, nil
}

// VerifyToken returns true if the string representation of the token has valid signature
// and it can be found in the list of active token (cache2go)
func (c *Certificate) VerifyToken(token string) bool {
	values, signature, err := c.ReadToken(token, true)
	if err != nil {
		return false
	}

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

func (c *Certificate) buildNewConnectionRequest(data url.Values) (requestID string, reqAsJSON []byte) {
	certSignature := make([]byte, len(c.Cert.Signature))
	copy(certSignature, c.Cert.Signature)

	addresses, _ := common.GetAddresses()

	req := &Token{
		ID:              uuid.NewV4().String(),
		IssuerID:        c.Cert.SerialNumber.String(),
		IssuerAddresses: addresses,
		CACertSignature: certSignature,

		Values: data,
	}

	reqAsJSON, _ = json.Marshal(req)
	return req.ID, reqAsJSON
}
