package securelink

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"
)

type (
	// Handler provides a way to use multiple handlers inside a sign TLS listener.
	// You specify the TLS certificate for server but the same certificate is used in case
	// of Dial.
	Handler struct {
		name string

		handleFunction FuncHandler

		matchFunction FuncServiceMatch
	}

	TransportConn struct {
		*tls.Conn
		Server bool
	}
)

// NewHandler builds a new Hanlder pointer to use in a server object
func NewHandler(name string, serviceMatchFunc FuncServiceMatch, handlerFunction FuncHandler) *Handler {
	return &Handler{
		name:           name,
		handleFunction: handlerFunction,
		matchFunction:  serviceMatchFunc,
	}
}

// Handle is called when a client connect to the server and the client point to the service.
func (t *Handler) Handle(conn *TransportConn) (err error) {
	if t.handleFunction == nil {
		return fmt.Errorf("no handler registered")
	}

	return t.handleFunction(conn)
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

func (tc *TransportConn) Read(b []byte) (n int, err error) {
	n, err = tc.Conn.Read(b)

	return
}

func (tc *TransportConn) Write(b []byte) (n int, err error) {
	n, err = tc.Conn.Write(b)

	return
}

func (tc *TransportConn) Close() (err error) {
	return tc.Conn.Close()
}

func (tc *TransportConn) LocalAddr() net.Addr {
	return tc.Conn.LocalAddr()
}

func (tc *TransportConn) RemoteAddr() net.Addr {
	return tc.Conn.RemoteAddr()
}

func (tc *TransportConn) SetDeadline(t time.Time) (err error) {
	return tc.Conn.SetDeadline(t)
}

func (tc *TransportConn) SetReadDeadline(t time.Time) (err error) {
	return tc.Conn.SetReadDeadline(t)
}

func (tc *TransportConn) SetWriteDeadline(t time.Time) (err error) {
	return tc.Conn.SetWriteDeadline(t)
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
