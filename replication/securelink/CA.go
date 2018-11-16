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
	"encoding/pem"
	"time"
)

type (
	// Certificate provides an easy way to use certificates with tls package
	Certificate struct {
		Cert       *x509.Certificate
		PrivateKey *ecdsa.PrivateKey

		CertPool *x509.CertPool
		IsCA     bool
	}

	// CA provides new Certificate pointers
	CA struct {
		*Certificate
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

	certTemplate := getCertTemplate(true, names, nil, lifeTime)

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
		getCertTemplate(false, names, nil, lifeTime),
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
		CertPool:   c.GetCertPool(),
		IsCA:       false,
	}, nil
}

// GetCertPEM is useful to start a new client or server with tls.X509KeyPair
func (c *Certificate) GetCertPEM() []byte {
	return buildCertPEM(c.Cert.Raw)
}

// GetPrivateKeyPEM is useful to start a new client or server with tls.X509KeyPair
func (c *Certificate) GetPrivateKeyPEM() []byte {
	curveAsBytes, _ := x509.MarshalECPrivateKey(c.PrivateKey)
	return buildKeyPEM(
		curveAsBytes,
	)
}

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
