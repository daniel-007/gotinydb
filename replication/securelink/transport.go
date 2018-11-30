package securelink

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
)

type (
	handler struct {
		name string

		handleFunction FuncHandler

		matchFunction FuncServiceMatch
	}

	// Handler provides a way to use multiple handlers inside a sign TLS listener.
	// You specify the TLS certificate for server but the same certificate is used in case
	// of Dial.
	Handler interface {
		Name() string

		Handle(conn net.Conn) error

		Match(hostName string) bool
	}

	// TransportConn is an interface to
	TransportConn struct {
		*tls.Conn
		Server bool
	}
)

// NewHandler builds a new Hanlder pointer to use in a server object
func NewHandler(name string, serviceMatchFunc FuncServiceMatch, handlerFunction FuncHandler) Handler {
	return &handler{
		name:           name,
		handleFunction: handlerFunction,
		matchFunction:  serviceMatchFunc,
	}
}

// Handle is called when a client connect to the server and the client point to the service.
func (t *handler) Handle(conn net.Conn) (err error) {
	if t.handleFunction == nil {
		return fmt.Errorf("no handler registered")
	}

	return t.handleFunction(conn)
}

func (t *handler) Name() string {
	return t.name
}

func (t *handler) Match(hostName string) bool {
	return t.matchFunction(hostName)
}

func newTransportConn(conn net.Conn, server bool) (*TransportConn, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil, fmt.Errorf("can't build Transport connection, the net.Conn interface is not a *tls.Conn pointer %T", conn)
	}

	tc := &TransportConn{
		Conn:   tlsConn,
		Server: server,
	}

	return tc, nil
}

// GetID provides a way to get an ID which in the package can be found
// as the first host name from the certificate.
// This function contact the server at the given address with an "insecure" connection
// to get it's certificate. Checks that the certificate is valid for the given certificate if given.
// From the certificate it extract the first HostName which is return.
func GetID(addr string, cert *Certificate) (serverID string) {
	tlsConfig := GetBaseTLSConfig("", cert)
	tlsConfig.InsecureSkipVerify = true
	conn, err := tls.Dial("tcp", string(addr), tlsConfig)
	if err != nil {
		return ""
	}

	err = conn.Handshake()
	if err != nil {
		return ""
	}

	if len(conn.ConnectionState().PeerCertificates) < 1 {
		return ""
	}

	remoteCert := conn.ConnectionState().PeerCertificates[0]
	opts := x509.VerifyOptions{
		Roots: cert.CertPool,
	}

	if _, err := remoteCert.Verify(opts); err != nil {
		return ""
	}

	return remoteCert.SerialNumber.String()
}
