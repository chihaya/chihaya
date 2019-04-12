package routers

import (
	"github.com/labstack/echo"
)

func InitRoutes(e *echo.Echo) {

	InitAuthRoutes(e)
	InitCmdRoutes(e)
}
