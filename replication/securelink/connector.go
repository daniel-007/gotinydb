package securelink

import (
	"crypto/tls"
	"net/http"
)

// NewHTTPSConnector provides a HTTP/S client with custom root CA and with the
// given client certificate
func NewHTTPSConnector(host string, cert *Certificate) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: GetBaseTLSConfig(host, cert),
		},
	}
}

// GetBaseTLSConfig returns a TLS configuration with the given certificate as
// "Certificate" and setup the "RootCAs" with the given certificate CertPool
func GetBaseTLSConfig(host string, cert *Certificate) *tls.Config {
	return &tls.Config{
		ServerName:   host,
		Certificates: []tls.Certificate{cert.GetTLSCertificate()},
		RootCAs:      cert.CertPool,
	}
}
