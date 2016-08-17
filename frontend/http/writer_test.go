package http

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya/bittorrent"
)

func TestWriteError(t *testing.T) {
	var table = []struct {
		reason, expected string
	}{
		{"hello world", "d14:failure reason11:hello worlde"},
		{"what's up", "d14:failure reason9:what's upe"},
	}

	for _, tt := range table {
		r := httptest.NewRecorder()
		err := WriteError(r, bittorrent.ClientError(tt.reason))
		assert.Nil(t, err)
		assert.Equal(t, r.Body.String(), tt.expected)
	}
}

func TestWriteStatus(t *testing.T) {
	r := httptest.NewRecorder()
	err := WriteError(r, bittorrent.ClientError("something is missing"))
	assert.Nil(t, err)
	assert.Equal(t, r.Body.String(), "d14:failure reason20:something is missinge")
}
