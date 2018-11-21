package replication

import (
	"crypto/tls"
	"net/http"

	"github.com/alexandrestein/gotinydb/replication/securelink"
)

// NewConnector provides a HTTP client with custom root CA
func NewConnector(host string, cert *securelink.Certificate) *http.Client {
	mTLSConfig := &tls.Config{
		ServerName:   host,
		Certificates: []tls.Certificate{cert.GetTLSCertificate()},
		RootCAs:      cert.CertPool,
	}

	tr := &http.Transport{
		TLSClientConfig: mTLSConfig,
	}

	return &http.Client{Transport: tr}
}
