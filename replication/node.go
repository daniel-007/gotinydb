package replication

import (
	"github.com/alexandrestein/gotinydb/replication/securelink"
	"github.com/labstack/echo"
)

type (
	Node struct {
		Echo        *echo.Echo
		Certificate *securelink.Certificate

		Port string
	}
)
