// Package securelink is not really for certificate management.
// It more a tool to make a cluster connection security easy.
// Build an save your CA. It will be able to generate Certificate pointers which
// can connect and check peer just on certificate validity.
//
// No need to check the host, you just want to make sur client and server use your CA.
package securelink

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"time"
)

type (
	// Certificate provides an easy way to use certificates with tls package
	Certificate struct {
		Cert       *x509.Certificate
		PrivateKey *ecdsa.PrivateKey

		CACert   *x509.Certificate
		CertPool *x509.CertPool
		IsCA     bool
	}

	// CA provides new Certificate pointers
	CA struct {
		*Certificate
	}

	certExport struct {
		Cert       []byte
		PrivateKey []byte
		CACert     []byte
	}
)

func genPrivateKey() (*ecdsa.PrivateKey, error) {
	privateKey, err := ecdsa.GenerateKey(Curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func buildCertPEM(input []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: input,
	})
}

func buildKeyPEM(input []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: input,
	})
}

// NewCA returns a new CA pointer which is supposed to be used as server certificate
// and client and server certificate for remote instances.
// names are used as domain names.
func NewCA(lifeTime time.Duration, names ...string) (*CA, error) {
	privateKey, err := genPrivateKey()
	if err != nil {
		return nil, err
	}

	certTemplate := GetCertTemplate(true, names, nil, lifeTime)

	var certAsDER []byte
	certAsDER, err = x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, privateKey.Public(), privateKey)
	if err != nil {
		return nil, err
	}

	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(certAsDER)
	if err != nil {
		return nil, err
	}

	ca := &CA{
		&Certificate{
			Cert:       cert,
			PrivateKey: privateKey,
			CACert:     cert,
			IsCA:       true,
		},
	}
	ca.CertPool = ca.GetCertPool()

	return ca, nil
}

// NewCert returns a new certificate pointer which can be used for tls connection
func (c *CA) NewCert(lifeTime time.Duration, names ...string) (*Certificate, error) {
	privateKey, err := genPrivateKey()
	if err != nil {
		return nil, err
	}

	// Sign certificate with the CA
	var certAsDER []byte
	certAsDER, err = x509.CreateCertificate(
		rand.Reader,
		GetCertTemplate(false, names, nil, lifeTime),
		c.Cert,
		privateKey.Public(),
		c.PrivateKey,
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
		Cert:       cert,
		PrivateKey: privateKey,
		CACert:     c.Cert,
		CertPool:   c.GetCertPool(),
		IsCA:       false,
	}, nil
}

// GetCertPEM is useful to start a new client or server with tls.X509KeyPair
func (c *Certificate) GetCertPEM() []byte {
	return buildCertPEM(c.Cert.Raw)
}

func (c *Certificate) getPrivateKeyDER() []byte {
	curveAsBytes, _ := x509.MarshalECPrivateKey(c.PrivateKey)
	return curveAsBytes
}

// GetPrivateKeyPEM is useful to start a new client or server with tls.X509KeyPair
func (c *Certificate) GetPrivateKeyPEM() []byte {
	return buildKeyPEM(c.getPrivateKeyDER())
}

// func (c *Certificate) MarshalRawKey() ([]byte, error) {
// 	ret, err := x509.MarshalECPrivateKey(c.PrivateKey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	c.RawKey = ret

// 	return ret, nil
// }

// func (c *Certificate) UnmarshalRawKey(input []byte) error {
// 	if input == nil || len(input) == 0 {
// 		input = c.RawKey
// 	}

// 	key, err := x509.ParseECPrivateKey(input)
// 	if err != nil {
// 		return err
// 	}

// 	c.PrivateKey = key
// 	return nil
// }

// GetTLSCertificate is useful in
// tls.Config{Certificates: []tls.Certificate{ca.GetTLSCertificate()}}
func (c *Certificate) GetTLSCertificate() tls.Certificate {
	cert, _ := tls.X509KeyPair(c.GetCertPEM(), c.GetPrivateKeyPEM())
	return cert
}

// GetCertPool is useful in tls.Config{RootCAs: ca.GetCertPool()}
func (c *CA) GetCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(c.Cert)

	return pool
}

// Marshal convert the Certificate pointer into a slice of byte for
// transport or future use
func (c *Certificate) Marshal() []byte {
	export := &certExport{
		Cert:       c.Cert.Raw,
		PrivateKey: c.getPrivateKeyDER(),
		CACert:     c.CACert.Raw,
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

	var privateKey *ecdsa.PrivateKey
	privateKey, err = x509.ParseECPrivateKey(export.PrivateKey)
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
		Cert:       cert,
		PrivateKey: privateKey,
		CACert:     caCert,
		CertPool:   certPool,
		IsCA:       cert.IsCA,
	}, nil
}
