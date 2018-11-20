package securelink

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"

	"github.com/alexandrestein/gotinydb/replication/securelink/securecache"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/muesli/cache2go"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
)

type (
	// Server start a web server which accept only connection with a client certificate
	// with the same CA as the server
	Server struct {
		Port        string
		Echo        *echo.Echo
		Certificate *Certificate
	}

	NewConnectionRequest struct {
		ID              string
		IssuerID        string
		IssuerAddresses []string
		IssuerPort      string
		CACertSignature []byte
	}
)

// NewServer initiates the server at the given address
func NewServer(certificate *Certificate, port string) (*Server, error) {
	e := echo.New()
	e.Logger.SetLevel(log.OFF)

	s := &Server{
		Port:        port,
		Echo:        e,
		Certificate: certificate,
	}

	s.settupHandlers()

	return s, nil
}

// Start starts the HTTP and TLS servers
func (s *Server) Start() error {
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{s.Certificate.GetTLSCertificate()},
		ClientCAs:    s.Certificate.CertPool,
		// ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientAuth: tls.VerifyClientCertIfGiven,
	}
	s.Echo.TLSServer.TLSConfig = serverTLSConfig

	tlsListener, err := tls.Listen("tcp", s.Port, serverTLSConfig)
	if err != nil {
		return err
	}

	return s.Echo.TLSServer.Serve(tlsListener)
}

func (s *Server) Close() error {
	return s.Echo.Close()
}

func (s *Server) GetBigID() *big.Int {
	return s.Certificate.Cert.SerialNumber
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

func (s *Server) GetAddresses() ([]string, error) {
	return getAddresses()
}

func getAddresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ret := []string{}

	for _, nic := range interfaces {
		var addrs []net.Addr
		addrs, err = nic.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ipAsString := addr.String()
			ip, _, err := net.ParseCIDR(ipAsString)
			if err != nil {
				continue
			}

			ipAsString = ip.String()
			ip2 := net.ParseIP(ipAsString)
			if to4 := ip2.To4(); to4 == nil {
				ipAsString = "[" + ipAsString + "]"
			}

			// If ip accessible from outside
			if ip.IsGlobalUnicast() {
				ret = append(ret, ipAsString)
			}
		}
	}

	return ret, nil
}

func (s *Server) GetToken() (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES384, Key: s.Certificate.PrivateKey}, nil)
	if err != nil {
		return "", err
	}

	reqID, token := s.buildNewConnectionRequest()

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

func (s *Server) ReadToken(token string, verify bool) (_ *NewConnectionRequest, signature []byte, _ error) {
	object, err := jose.ParseSigned(token)
	if err != nil {
		return nil, nil, err
	}

	var output []byte
	if verify {
		output, err = object.Verify(s.Certificate.PrivateKey.Public())
		if err != nil {
			return nil, nil, err
		}
	} else {
		output = object.UnsafePayloadWithoutVerification()
	}

	signature = object.Signatures[0].Signature

	values := new(NewConnectionRequest)
	err = json.Unmarshal(output, values)
	if err != nil {
		return nil, nil, err
	}

	if values.ID == "" {
		return nil, nil, fmt.Errorf("the request token does not containe any ID")
	}

	return values, signature, nil
}

func (s *Server) VerifyToken(token string) bool {
	values, signature, err := s.ReadToken(token, true)
	if err != nil {
		return false
	}

	// cache := cache2go.Cache(CacheValueWaitingRequestsTable)
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

func (s *Server) buildNewConnectionRequest() (requestID string, reqAsJSON []byte) {
	certSignature := make([]byte, len(s.Certificate.Cert.Signature))
	copy(certSignature, s.Certificate.Cert.Signature)

	addresses, _ := s.GetAddresses()

	req := &NewConnectionRequest{
		ID:              uuid.NewV4().String(),
		IssuerID:        s.GetBigID().String(),
		IssuerPort:      s.Port,
		IssuerAddresses: addresses,
		CACertSignature: certSignature,
	}

	reqAsJSON, _ = json.Marshal(req)
	return req.ID, reqAsJSON
}

func GetClientCertificate(address, port, tokenString string, token *NewConnectionRequest) (cert *Certificate, usedAddress string, _ error) {
	// ip := net.ParseIP(address)
	// if to4 := ip.To4(); to4 == nil {
	// 	address = "[" + address + "]"
	// }

	path := fmt.Sprintf("https://%s%s/%s/%s", address, port, APIVersion, PostCertificatePATH)
	// fmt.Println("path", path)

	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:         token.ID,
				InsecureSkipVerify: true,
			},
		},
	}

	data := url.Values{}
	data.Set("token", tokenString)

	resp, err := insecureClient.PostForm(path, data)
	if err != nil {
		return nil, "", err
	}

	if caSign, tokenSign := fmt.Sprintf("%x", resp.TLS.PeerCertificates[0].Signature), fmt.Sprintf("%x", token.CACertSignature); caSign != tokenSign {
		return nil, "", fmt.Errorf("the signature from the token is not equal to the server certificate \n\t%q \n\t%q", caSign, tokenSign)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("respond status is not 200 but %d: %q", resp.StatusCode, resp.Status)
	}

	buffer := make([]byte, 1000*1000) //1MB
	var nb int
	nb, err = io.ReadFull(resp.Body, buffer)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, "", err
		}
	}
	buffer = buffer[:nb]

	var certificate *Certificate
	certificate, err = Unmarshal(buffer)
	if err != nil {
		return nil, "", err
	}

	return certificate, address, nil
}
