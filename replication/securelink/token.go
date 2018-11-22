package securelink

import (
	"fmt"

	"github.com/alexandrestein/gotinydb/replication/common"
	"github.com/alexandrestein/gotinydb/replication/securelink/securecache"
	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
)

func (c *Certificate) getTokenSignAlgorithm() jwt.SigningMethod {
	if c.KeyPair.Type == KeyTypeRSA {
		if c.KeyPair.Length == KeyLengthRsa2048 {
			return jwt.GetSigningMethod("RS256")
		} else if c.KeyPair.Length == KeyLengthRsa3072 {
			return jwt.GetSigningMethod("RS384")
		} else if c.KeyPair.Length == KeyLengthRsa4096 || c.KeyPair.Length == KeyLengthRsa8192 {
			return jwt.GetSigningMethod("RS512")
		}
	} else if c.KeyPair.Type == KeyTypeEc {
		if c.KeyPair.Length == KeyLengthEc256 {
			return jwt.GetSigningMethod("ES256")
		} else if c.KeyPair.Length == KeyLengthEc384 {
			return jwt.GetSigningMethod("ES384")
		} else if c.KeyPair.Length == KeyLengthEc521 {
			return jwt.GetSigningMethod("ES512")
		}
	}
	return nil
}

// GetToken returns a string representation of a temporary token (10 minutes validity with cache2go)
func (c *Certificate) GetToken(portString string) (string, error) {
	reqID, claims := c.buildNewConnectionRequest(portString)

	token := jwt.NewWithClaims(c.getTokenSignAlgorithm(), claims)

	tokenString, err := token.SignedString(c.KeyPair.Private)
	if err != nil {
		return "", err
	}

	securecache.WaitingRequestTable.Add(reqID, securecache.CacheValueWaitingRequestsTimeOut, token)

	return tokenString, nil
}

// ReadToken returns a Token pointer from it's string representation
func (c *Certificate) ReadToken(tokenString string, verify bool) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return c.KeyPair.Public, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid && verify {
		return nil, fmt.Errorf("token invalid")
	}

	return token, nil
}

// VerifyToken returns true if the string representation of the token has valid signature
// and it can be found in the list of active token (cache2go)
func (c *Certificate) VerifyToken(tokenString string) bool {
	token, err := c.ReadToken(tokenString, true)
	if err != nil {
		return false
	}

	id := token.Claims.(jwt.MapClaims)["jti"].(string)
	_, err = securecache.WaitingRequestTable.Delete(id)
	if err != nil {
		return false
	}

	return true
}

func (c *Certificate) buildNewConnectionRequest(portString string) (requestID string, _ jwt.Claims) {
	caCertSignature := make([]byte, len(c.CACert.Signature))
	copy(caCertSignature, c.CACert.Signature)

	addresses, _ := common.GetAddresses()

	reqID := uuid.NewV4().String()
	req := jwt.MapClaims{
		"aud":        jwtNewNodeAudience,
		"exp":        jwtNewNodeExpiresAt(),
		"jti":        reqID,
		"iat":        jwtNewNodeIssuedAt(),
		"iss":        c.ID().String(),
		"nbf":        jwtNewNodeNotBefore(),
		"sub":        jwtNewNodeSubject,
		"issAddrs":   addresses,
		"issPort":    portString,
		"caCertSign": caCertSignature,
	}

	return reqID, req
}
