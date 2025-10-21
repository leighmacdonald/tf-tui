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
