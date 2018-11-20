package replication

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

type (
	handler struct {
		*node
	}
)

// Defines constants to for the API path
var (
	APIVersion          = "v0.0.0"
	PostCertificatePATH = "newClient"
	GetClusterMapPATH   = "map"

	ServerIDHeadearName = "Server-Id"
)

func (n *node) settupHandlers() {
	handler := &handler{n}

	apiGroup := n.Server.Echo.Group(
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

	clientCert := h.GetCert()
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

		c.Response().Header().Add(ServerIDHeadearName, h.ID)

		return next(c)
	})
}
