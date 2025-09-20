package events

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	const statusFull = `hostname: Uncletopia | Chicago | 1 | All Maps
version : 9978583/24 9978583 secure
udp/ip  : ?.?.?.?:?  (public IP from Steam: 108.181.62.21)
steamid : [G:1:4430560] (85568392924469984)
account : not logged in  (No account specified)
map     : pl_patagonia at: 0 x, 0 y, 0 z
tags    : nocrits,nodmgspread,payload,uncletopia
sourcetv:  ?.?.?.?:?, delay 0.0s  (local: 0.0.0.0:27016)
players : 24 humans, 1 bots (33 max)
edicts  : 1678 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#      2 "Uncletopia | Chicago | 1 | All " BOT                       active
#     98 "Toonice [no sound]" [U:1:442729157]     1:02:19    66    0 active 1.1.1.1:27005
#    114 "Cajun Fox"         [U:1:33211782]      40:13       83    0 active 6.6.6.6:27005
`
	type tc struct {
		Line   string
		Result Event
	}

	// 08/16/2025 - 01:13:50: Umevol killed (TPT) Mystic Ghost with scattergun.
	// 08/16/2025 - 01:13:52: GlorpiusJinglebuck killed jaydendillonk with knife. (crit)

	cases := []tc{
		{
			Line: "#     98 \"Toonice [no sound]\" [U:1:442729157]     1:02:19    66    0 active 1.1.1.1:27005",
			Result: Event{Type: StatusID, Data: StatusIDEvent{
				Player:    "Toonice [no sound]",
				UserID:    98,
				PlayerSID: steamid.New("[U:1:442729157]"),
				Connected: 3739,
				Ping:      66,
				Loss:      0,
				State:     "active",
				Address:   "1.1.1.1:27005",
			}},
		}, {
			Line:   "hostname: Uncletopia | Chicago | 1 | All Maps",
			Result: Event{Type: Hostname, Data: HostnameEvent{Hostname: "Uncletopia | Chicago | 1 | All Maps"}},
		}, // {Line: "version : 9978583/24 9978583 secure", Result: Event{Type: Hostname}},
		{
			Line:   "version : 9978583/24 9978583 secure",
			Result: Event{Type: Version, Data: VersionEvent{Version: 9978583, Secure: true}},
		}, {
			Line:   "map     : pl_patagonia at: 0 x, 0 y, 0 z",
			Result: Event{Type: Map, Data: MapEvent{MapName: "pl_patagonia"}},
		}, {
			Line:   "tags    : nocrits,nodmgspread,payload,uncletopia",
			Result: Event{Type: Tags, Data: TagsEvent{Tags: []string{"nocrits", "nodmgspread", "payload", "uncletopia"}}},
		}, {
			Line:   "udp/ip  : ?.?.?.?:?  (public IP from Steam: 108.181.62.21)",
			Result: Event{Type: Address, Data: AddressEvent{Address: netip.MustParseAddr("108.181.62.21")}},
		}, {
			Line:   "08/16/2025 - 01:13:50: Umevol killed (TPT) Mystic Ghost with scattergun.",
			Result: Event{Type: Kill, Data: KillEvent{Player: "Umevol", Victim: "(TPT) Mystic Ghost", Weapon: "scattergun"}},
		}, {
			Line:   "08/16/2025 - 01:13:52: GlorpiusJinglebuck killed jaydendillonk with knife. (crit)",
			Result: Event{Type: Kill, Data: KillEvent{Player: "GlorpiusJinglebuck", Victim: "jaydendillonk", Weapon: "knife", Crit: true}},
		}, {
			Line:   "Umevol killed (TPT) Mystic Ghost with scattergun.",
			Result: Event{Type: Kill, Data: KillEvent{Player: "Umevol", Victim: "(TPT) Mystic Ghost", Weapon: "scattergun"}},
		}, {
			Line:   "GlorpiusJinglebuck killed jaydendillonk with knife. (crit)",
			Result: Event{Type: Kill, Data: KillEvent{Player: "GlorpiusJinglebuck", Victim: "jaydendillonk", Weapon: "knife", Crit: true}},
		},
	}

	parser := newParser()

	for index, testCase := range cases {
		evt, err := parser.parse(testCase.Line)
		require.NoError(t, err, fmt.Sprintf("Test %d fail - parse", index))
		require.Equal(t, testCase.Result.Type, evt.Type, fmt.Sprintf("Test %d fail - type", index))
		require.Equal(t, testCase.Result.Data, evt.Data)
	}
}
