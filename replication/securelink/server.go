package securelink

import (
	"crypto/tls"
	"net/http"

	"github.com/labstack/echo"
)

type (
	// Server start a web server which accept only connection with a client certificate
	// with the same CA as the server
	Server struct {
		Address     string
		Echo        *echo.Echo
		Certificate *Certificate
	}
)

// NewServer initiates the server at the given address
func NewServer(address string, certificate *Certificate) (*Server, error) {
	e := echo.New()

	return &Server{
		Address:     address,
		Echo:        e,
		Certificate: certificate,
	}, nil
}

// Start starts the HTTP and TLS servers
func (s *Server) Start() error {
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{s.Certificate.GetTLSCertificate()},
		ClientCAs:    s.Certificate.CertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	s.Echo.TLSServer.TLSConfig = serverTLSConfig

	tlsListener, err := tls.Listen("tcp", s.Address, serverTLSConfig)
	if err != nil {
		return err
	}

	return s.Echo.TLSServer.Serve(tlsListener)
}

// NewConnector provides a HTTP client with custom root CA
func NewConnector(host string, cert *Certificate) *http.Client {
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
