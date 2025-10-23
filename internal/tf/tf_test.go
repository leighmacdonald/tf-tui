package tf_test

import (
	"testing"

	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/stretchr/testify/require"
)

func TestParseCVar(t *testing.T) {
	const data = `ai_frametime_limit                       : 50       : , "sv"           : frametime limit for min efficiency AIE_NORMAL (in sec's).
ai_inhibit_spawners                      : 0        : , "sv", "cheat"  :
ai_lead_time                             : 0        : , "sv"           :
ai_LOS_mode                              : 0        : , "sv", "rep"    :
ai_moveprobe_debug                       : 0        : , "sv"           : `

	cvars := tf.ParseCVars(data)
	require.Len(t, cvars, 5)

}

func TestParsePlugins(t *testing.T) {
	const smPlugins = `[SM] Listing 48 plugins:
  01 "Admin Help" (1.13.0.7251) by AlliedModders LLC
  02 "No Contracker" (1.1) by Malifox, Sreaper
  03 "Player Commands" (1.13.0.7251) by AlliedModders LLC
  04 "Disable Auto-Kick" (0.3) by The-Killer`
	const metaPlugins = `rcon (1 hosts)> meta list
  Listing 9 plugins:
  [01] SourceMod (1.13.0.7251) by AlliedModders LLC
  [02] Sentry Error Logger (1.2) by rob5300 - Creators.TF
  [03] TF2 Tools (1.13.0.7251) by AlliedModders LLC
  [04] SDK Hooks (1.13.0.7251) by AlliedModders LLC
`
	mmPluginsFound := tf.ParseGamePlugins(metaPlugins, false)
	require.Len(t, mmPluginsFound, 4)
	smPluginsFound := tf.ParseGamePlugins(smPlugins, false)
	require.Len(t, smPluginsFound, 4)

}
