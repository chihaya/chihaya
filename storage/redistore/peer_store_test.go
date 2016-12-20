package redistore_test

import (
	"testing"

	"github.com/RealImage/chihaya/storage/redistore"
)

func TestNew(t *testing.T) {

	cfg := &redistore.Config{Namespace: true,
		Cntrl: "nm",
		Host:  "",
		Port:  "6379",
	}

	_, err := cfg.New()
	if err != nil {
		t.Error("connection failed: " + err.Error())
	}

}
