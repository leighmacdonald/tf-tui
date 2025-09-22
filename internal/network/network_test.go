package network_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/leighmacdonald/tf-tui/internal/network"
	"github.com/stretchr/testify/require"
)

func TestQueryExternalIP(t *testing.T) {
	ipaddr, err := network.FetchIPInfo(context.Background())
	require.NoError(t, err)
	require.True(t, ipaddr.IP != "")
	addr, err2 := netip.ParseAddr(ipaddr.IP)
	require.NoError(t, err2)
	require.True(t, addr.Is4())
}
