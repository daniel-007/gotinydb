package echohandler

import (
	"net"

	"github.com/labstack/echo"
)

type (
	Handler struct {
		Echo *echo.Echo
	}
)

func (h *Handler) Handle(conn net.Conn) error {

	return nil
}
