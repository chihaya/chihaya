package http

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

func TestWriteError(t *testing.T) {
	table := []struct {
		reason, expected string
	}{
		{"hello world", "d14:failure reason11:hello worlde"},
		{"what's up", "d14:failure reason9:what's upe"},
	}

	for _, tt := range table {
		t.Run(fmt.Sprintf("%s expecting %s", tt.reason, tt.expected), func(t *testing.T) {
			r := httptest.NewRecorder()
			err := WriteError(r, bittorrent.ClientError(tt.reason))
			require.Nil(t, err)
			require.Equal(t, r.Body.String(), tt.expected)
		})
	}
}

func TestWriteStatus(t *testing.T) {
	table := []struct {
		reason, expected string
	}{
		{"something is missing", "d14:failure reason20:something is missinge"},
	}

	for _, tt := range table {
		t.Run(fmt.Sprintf("%s expecting %s", tt.reason, tt.expected), func(t *testing.T) {
			r := httptest.NewRecorder()
			err := WriteError(r, bittorrent.ClientError(tt.reason))
			require.Nil(t, err)
			require.Equal(t, r.Body.String(), tt.expected)
		})
	}
}
