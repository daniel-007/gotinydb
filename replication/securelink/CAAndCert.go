// Package securelink is not really for certificate management.
// It more a tool to make a cluster connection security easy.
// Build an save your CA. It will be able to generate Certificate pointers which
// can connect and check peer just on certificate validity.
//
// No need to check the host, you just want to make sur client and server use your CA.
package securelink

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"
)

type (
	// Certificate provides an easy way to use certificates with tls package
	Certificate struct {
		Cert    *x509.Certificate
		KeyPair *KeyPair

		CACert   *x509.Certificate
		CertPool *x509.CertPool
		IsCA     bool
	}

	certExport struct {
		Cert    []byte
		KeyPair []byte
		CACert  []byte
	}
)

func buildCertPEM(input []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: input,
	})
}

func buildEcKeyPEM(input []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: input,
	})
}

func genKeyPair(keyType KeyType, keyLength KeyLength) (*KeyPair, error) {
	if keyType == KeyTypeRSA {
		switch keyLength {
		case KeyLengthRsa2048, KeyLengthRsa3072, KeyLengthRsa4096, KeyLengthRsa8192:
			return NewRSA(keyLength), nil
		}
	} else if keyType == KeyTypeEc {
		switch keyLength {
		case KeyLengthEc256, KeyLengthEc384, KeyLengthEc521:
			return NewEc(keyLength), nil
		}
	}

	return nil, ErrKeyConfigNotCompatible
}

// GetSignatureAlgorithm returns the signature algorithm for the given key type and key size
func GetSignatureAlgorithm(keyType KeyType, keyLength KeyLength) x509.SignatureAlgorithm {
	if keyType == KeyTypeRSA {
		switch keyLength {
		case KeyLengthRsa2048:
			return x509.SHA256WithRSAPSS
		case KeyLengthRsa3072:
			return x509.SHA384WithRSAPSS
		case KeyLengthRsa4096, KeyLengthRsa8192:
			return x509.SHA512WithRSAPSS
		}
	} else if keyType == KeyTypeEc {
		switch keyLength {
		case KeyLengthEc256:
			return x509.ECDSAWithSHA256
		case KeyLengthEc384:
			return x509.ECDSAWithSHA384
		case KeyLengthEc521:
			return x509.ECDSAWithSHA512
		}
	}
	return x509.UnknownSignatureAlgorithm
}

// NewCA returns a new CA pointer which is supposed to be used as server certificate
// and client and server certificate for remote instances.
// names are used as domain names.
func NewCA(keyType KeyType, keyLength KeyLength, lifeTime time.Duration, certTemplate *x509.Certificate, names ...string) (*Certificate, error) {
	keyPair, err := genKeyPair(keyType, keyLength)
	if err != nil {
		return nil, err
	}

	certTemplate.IsCA = true
	certTemplate.DNSNames = append(certTemplate.DNSNames, names...)
	certTemplate.SignatureAlgorithm = GetSignatureAlgorithm(keyType, keyLength)

	certAsDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, keyPair.Public, keyPair.Private)
	if err != nil {
		return nil, err
	}

	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(certAsDER)
	if err != nil {
		return nil, err
	}

	ca := &Certificate{
		Cert:    cert,
		KeyPair: keyPair,
		CACert:  cert,
		IsCA:    true,
	}
	ca.CertPool = ca.GetCertPool()

	return ca, nil
}

// NewCert returns a new certificate pointer which can be used for tls connection
func (c *Certificate) NewCert(keyType KeyType, keyLength KeyLength, lifeTime time.Duration, certTemplate *x509.Certificate, names ...string) (*Certificate, error) {
	if !c.IsCA {
		return nil, fmt.Errorf("this is not a CA")
	}

	keyPair, err := genKeyPair(keyType, keyLength)
	if err != nil {
		return nil, err
	}

	certTemplate.IsCA = false
	certTemplate.DNSNames = append(certTemplate.DNSNames, names...)
	certTemplate.SignatureAlgorithm = GetSignatureAlgorithm(c.KeyPair.Type, c.KeyPair.Length)

	// Sign certificate with the CA
	var certAsDER []byte
	certAsDER, err = x509.CreateCertificate(
		rand.Reader,
		certTemplate,
		c.Cert,
		keyPair.Public,
		c.KeyPair.Private,
	)
	if err != nil {
		return nil, err
	}

	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(certAsDER)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Cert:     cert,
		KeyPair:  keyPair,
		CACert:   c.Cert,
		CertPool: c.GetCertPool(),
		IsCA:     false,
	}, nil
}

// GetCertPEM is useful to start a new client or server with tls.X509KeyPair
func (c *Certificate) GetCertPEM() []byte {
	return buildCertPEM(c.Cert.Raw)
}

// GetTLSCertificate is useful in
// tls.Config{Certificates: []tls.Certificate{ca.GetTLSCertificate()}}
func (c *Certificate) GetTLSCertificate() tls.Certificate {
	cert, _ := tls.X509KeyPair(c.GetCertPEM(), c.KeyPair.GetPrivatePEM())
	// cert, _ := tls.X509KeyPair(c.GetCertPEM(), c.GetPrivateKeyPEM())
	return cert
}

// GetCertPool is useful in tls.Config{RootCAs: ca.GetCertPool()}
func (c *Certificate) GetCertPool() *x509.CertPool {
	if !c.IsCA {
		return nil
	}
	pool := x509.NewCertPool()
	pool.AddCert(c.Cert)

	return pool
}

// Marshal convert the Certificate pointer into a slice of byte for
// transport or future use
func (c *Certificate) Marshal() []byte {
	export := &certExport{
		Cert:    c.Cert.Raw,
		KeyPair: c.KeyPair.Marshal(),
		CACert:  c.CACert.Raw,
	}

	ret, _ := json.Marshal(export)

	return ret
}

// Unmarshal build a new Certificate pointer with the information given
// by the input
func Unmarshal(input []byte) (*Certificate, error) {
	export := new(certExport)
	err := json.Unmarshal(input, export)
	if err != nil {
		return nil, err
	}

	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(export.Cert)
	if err != nil {
		return nil, err
	}

	var keyPair *KeyPair
	keyPair, err = UnmarshalKeyPair(export.KeyPair)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	var caCert *x509.Certificate
	caCert, err = x509.ParseCertificate(export.CACert)
	if err != nil {
		return nil, err
	}
	certPool.AddCert(caCert)

	return &Certificate{
		Cert:     cert,
		KeyPair:  keyPair,
		CACert:   caCert,
		CertPool: certPool,
		IsCA:     cert.IsCA,
	}, nil
}
