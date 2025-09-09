package geoip_test

import (
	"testing"

	"github.com/leighmacdonald/tf-tui/internal/geoip"
	"github.com/stretchr/testify/require"
)

func TestLookup(t *testing.T) {
	record, errLookip := geoip.Lookup("12.55.66.88")
	require.NoError(t, errLookip)
	require.Equal(t, "US", record.Country.ISOCode)

	_, err := geoip.Lookup("bad")
	require.Error(t, err)
}
