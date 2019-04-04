package routers

import (
	"github.com/labstack/echo"
)

// InitRoutes ...
func InitRoutes(e *echo.Echo) {

	InitAuthRoutes(e)
	InitCmdRoutes(e)
}
