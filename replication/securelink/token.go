package securelink

import (
	"encoding/json"
	"fmt"

	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/alexandrestein/gotinydb/replication/securelink/securecache"
	"github.com/muesli/cache2go"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
)

type (
	Token struct {
		ID              string
		IssuerID        string
		IssuerAddresses []string
		IssuerPort      string
		CACertSignature []byte
	}
)

func (c *Certificate) GetToken(issuerID, issuerPort string) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES384, Key: c.PrivateKey}, nil)
	if err != nil {
		return "", err
	}

	reqID, token := c.buildNewConnectionRequest(issuerID, issuerPort)

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

func (c *Certificate) ReadToken(token string, verify bool) (_ *Token, signature []byte, _ error) {
	object, err := jose.ParseSigned(token)
	if err != nil {
		return nil, nil, err
	}

	var output []byte
	if verify {
		output, err = object.Verify(c.PrivateKey.Public())
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

func (c *Certificate) buildNewConnectionRequest(issuerID, issuerPort string) (requestID string, reqAsJSON []byte) {
	certSignature := make([]byte, len(c.Cert.Signature))
	copy(certSignature, c.Cert.Signature)

	addresses, _ := common.GetAddresses()

	req := &Token{
		ID:              uuid.NewV4().String(),
		IssuerID:        issuerID,
		IssuerPort:      issuerPort,
		IssuerAddresses: addresses,
		CACertSignature: certSignature,
	}

	reqAsJSON, _ = json.Marshal(req)
	return req.ID, reqAsJSON
}
