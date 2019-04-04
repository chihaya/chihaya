package routers

import (
	"github.com/labstack/echo"

	"github.com/ProtocolONE/chihaya/frontend/cord/controllers"
	"github.com/ProtocolONE/chihaya/frontend/cord/core/authentication"
)

// InitCmdRoutes ...
func InitCmdRoutes(e *echo.Echo) {

	e.POST("/api/v1/tracker/torrent", controllers.AddTorrent, authentication.RequireTokenAuthentication)
	e.DELETE("/api/v1/tracker/torrent", controllers.DeleteTorrent, authentication.RequireTokenAuthentication)
}
