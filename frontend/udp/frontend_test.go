package udp_test

import (
	"testing"

	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/storage"
	_ "github.com/chihaya/chihaya/storage/memory"
)

func TestStartStopRaceIssue437(t *testing.T) {
	ps, err := storage.NewPeerStore("memory", nil)
	if err != nil {
		t.Fatal(err)
	}
	var responseConfig middleware.ResponseConfig
	lgc := middleware.NewLogic(responseConfig, ps, nil, nil)
	fe, err := udp.NewFrontend(lgc, udp.Config{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	errC := fe.Stop()
	errs := <-errC
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
}
