package daemon

import (
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
)

func applyLoopRunPinningForTest(run *looppkg.Run, at time.Time) {
	run.StartedAt = at
	run.DefinitionVersion = 1
	run.DefinitionDigest = "sha256:test-definition"
	run.DefinitionSnapshot = []byte(
		`{"apiVersion":"agh.loop/v1","kind":"Loop","meta":{"name":"` + run.LoopName + `","version":1},"contract":{"goal":"test","definition_of_done":"done","iteration_cap":1,"no_progress":{"window":1},"budget":{"tokens":1,"wall_clock_sec":1}},"graph":{"nodes":[],"edges":[]}}`,
	)
	run.ActiveHumanCriteria = []byte(`[]`)
	if run.StartMetadata == nil {
		run.StartMetadata = map[string]any{}
	}
}
