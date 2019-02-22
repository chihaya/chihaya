package cord

import (
	"fmt"
	"context"

	"github.com/ProtocolONE/chihaya/frontend/cord/config"
	"github.com/ProtocolONE/chihaya/frontend/cord/routers"
	"github.com/ProtocolONE/chihaya/frontend/cord/core"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	
	"github.com/ProtocolONE/chihaya/frontend"
	"github.com/ProtocolONE/chihaya/pkg/log"
	"github.com/ProtocolONE/chihaya/pkg/stop"
)

// Frontend represents the state of an HTTP BitTorrent Frontend.
type Frontend struct {
	server    *echo.Echo

	logic frontend.TrackerLogic
}

// NewFrontend creates a new instance of an HTTP Frontend that asynchronously
// serves requests.
func NewFrontend(logic frontend.TrackerLogic) (*Frontend, error) {

	f := &Frontend{
		logic:  logic,
	}

	err := core.InitCord()
	if err != nil {
		return nil, err
	}

	conf := config.Get()

	f.server = echo.New()

	// Middleware
	f.server.Use(middleware.Logger())
	f.server.Use(middleware.Recover())

	// Routes
	routers.InitRoutes(f.server)

	// Start server
	go func() {
		if err := f.server.Start(fmt.Sprintf(":%d", conf.Service.ServicePort)); err != nil {
			if "http: Server closed" != err.Error() {
				log.Fatal("failed while serving cord api", log.Err(err))
			}
		}
	}()

	return f, nil
}

// Stop provides a thread-safe way to shutdown a currently running Frontend.
func (f *Frontend) Stop() stop.Result {

	stopGroup := stop.NewGroup()
	stopGroup.AddFunc(f.makeStopFunc(f.server))

	return stopGroup.Stop()
}

func (f *Frontend) makeStopFunc(stopSrv *echo.Echo) stop.Func {
	return func() stop.Result {
		c := make(stop.Channel)
		go func() {
			c.Done(stopSrv.Shutdown(context.Background()))
		}()
		return c.Result()
	}
}

