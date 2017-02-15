package bittorrent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	var table = []struct {
		data        string
		expected    Event
		expectedErr error
	}{
		{"", None, nil},
		{"NONE", None, nil},
		{"none", None, nil},
		{"started", Started, nil},
		{"stopped", Stopped, nil},
		{"completed", Completed, nil},
		{"notAnEvent", None, ErrUnknownEvent},
	}

	for _, tt := range table {
		got, err := NewEvent(tt.data)
		require.Equal(t, err, tt.expectedErr, "errors should equal the expected value")
		require.Equal(t, got, tt.expected, "events should equal the expected value")
	}
}
