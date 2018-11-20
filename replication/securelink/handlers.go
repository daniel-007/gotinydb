package securelink

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

type (
	handler struct {
		*Server
	}
)

// Defines constants to for the API path
var (
	APIVersion          = "v0.0.0"
	PostCertificatePATH = "newClient"
	GetClusterMapPATH   = "map"

	ServerIDHeadearName = "Server-Id"
)

func (s *Server) settupHandlers() {
	handler := &handler{s}

	apiGroup := s.Echo.Group(
		fmt.Sprintf("/%s/", APIVersion),
		handler.verifyCertificateMiddleware,
	)

	apiGroup.POST(PostCertificatePATH, handler.returnCert)
	apiGroup.GET(GetClusterMapPATH, handler.clusterMap)
}

func (h *handler) returnCert(c echo.Context) error {
	tokenAsString := c.FormValue("token")

	if !h.VerifyToken(tokenAsString) {
		return fmt.Errorf("the given token is not valid")
	}

	clientCert := h.Certificate
	clientCertAsBytes := clientCert.Marshal()

	return c.Blob(http.StatusOK, "text/json", clientCertAsBytes)
}

func (h *handler) clusterMap(c echo.Context) error {
	fmt.Println("map")

	return c.JSON(http.StatusOK, nil)
}

func (h *handler) verifyCertificateMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return echo.HandlerFunc(func(c echo.Context) error {

		if !c.IsTLS() {
			return echo.ErrUnauthorized
		}

		if len(c.Request().TLS.PeerCertificates) <= 0 && c.Path() != fmt.Sprintf("/%s/%s", APIVersion, PostCertificatePATH) {
			return echo.NewHTTPError(http.StatusUnauthorized, "no client certificate")
		}

		c.Response().Header().Add(ServerIDHeadearName, h.GetBigID().String())

		return next(c)
	})
}

// func (h *handler) GetToken() (string, error) {
// 	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES384, Key: h.Certificate.PrivateKey}, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	reqID, token := h.buildNewConnectionRequest()

// 	object, err := signer.Sign(token)
// 	if err != nil {
// 		return "", err
// 	}

// 	serialized, err := object.CompactSerialize()
// 	if err != nil {
// 		return "", err
// 	}

// 	securecache.WaitingRequestTable.Add(reqID, securecache.CacheValueWaitingRequestsTimeOut, object.Signatures[0].Signature)

// 	return serialized, nil
// }

// func (h *handler) readToken(token string, verify bool) (_ *newConnectionRequest, signature []byte, _ error) {
// 	object, err := jose.ParseSigned(token)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	var output []byte
// 	if verify {
// 		output, err = object.Verify(h.Certificate.PrivateKey.Public())
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	} else {
// 		output = object.UnsafePayloadWithoutVerification()
// 	}

// 	signature = object.Signatures[0].Signature

// 	values := new(newConnectionRequest)
// 	err = json.Unmarshal(output, values)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if values.ID == "" {
// 		return nil, nil, fmt.Errorf("the request token does not containe any ID")
// 	}

// 	return values, signature, nil
// }

// func (h *handler) VerifyToken(token string) bool {
// 	values, signature, err := h.readToken(token, true)
// 	if err != nil {
// 		return false
// 	}

// 	// cache := cache2go.Cache(CacheValueWaitingRequestsTable)
// 	var res *cache2go.CacheItem
// 	res, err = securecache.WaitingRequestTable.Value(values.ID)
// 	if err != nil {
// 		return false
// 	}

// 	if fmt.Sprintf("%x", res.Data().([]byte)) == fmt.Sprintf("%x", signature) {
// 		securecache.WaitingRequestTable.Delete(values.ID)
// 		return true
// 	}

// 	return false
// }

// func (h *handler) buildNewConnectionRequest() (requestID string, reqAsJSON []byte) {
// 	certSignature := make([]byte, len(h.Certificate.Cert.Signature))
// 	copy(certSignature, h.Certificate.Cert.Signature)

// 	addresses, _ := h.GetAddresses()

// 	req := &newConnectionRequest{
// 		ID:              uuid.NewV4().String(),
// 		IssuerID:        h.GetBigID().String(),
// 		IssuerPort:      h.Port,
// 		IssuerAddresses: addresses,
// 		CACertSignature: certSignature,
// 	}

// 	reqAsJSON, _ = json.Marshal(req)
// 	return req.ID, reqAsJSON
// }
