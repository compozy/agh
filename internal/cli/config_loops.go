package cli

import "maps"

func loopDefaultConfigSetPathKinds() map[string]configSetValueKind {
	return map[string]configSetValueKind{
		"loops.defaults.delivery.iteration_cap":         configSetInt,
		"loops.defaults.delivery.no_progress.window":    configSetInt,
		"loops.defaults.delivery.gates.max_revisions":   configSetInt,
		"loops.defaults.delivery.budget.tokens":         configSetInt,
		"loops.defaults.delivery.budget.wall_clock_sec": configSetInt,
		"loops.defaults.delivery.budget.on_exceeded":    configSetString,
		"loops.defaults.delivery.fan_out_width":         configSetInt,
		"loops.defaults.watch.iteration_cap":            configSetInt,
		"loops.defaults.watch.no_progress.window":       configSetInt,
		"loops.defaults.watch.gates.max_revisions":      configSetInt,
		"loops.defaults.watch.budget.tokens":            configSetInt,
		"loops.defaults.watch.budget.wall_clock_sec":    configSetInt,
		"loops.defaults.watch.budget.on_exceeded":       configSetString,
		"loops.defaults.watch.fan_out_width":            configSetInt,
	}
}

func mergeConfigSetValueKinds(
	base map[string]configSetValueKind,
	overlay map[string]configSetValueKind,
) map[string]configSetValueKind {
	merged := maps.Clone(base)
	maps.Copy(merged, overlay)
	return merged
}
