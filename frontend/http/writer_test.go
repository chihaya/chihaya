package http

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

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
		require.Nil(t, err)
		require.Equal(t, r.Body.String(), tt.expected)
	}
}

func TestWriteStatus(t *testing.T) {
	r := httptest.NewRecorder()
	err := WriteError(r, bittorrent.ClientError("something is missing"))
	require.Nil(t, err)
	require.Equal(t, r.Body.String(), "d14:failure reason20:something is missinge")
}
