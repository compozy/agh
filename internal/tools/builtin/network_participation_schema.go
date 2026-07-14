package builtin

const networkParticipationRequestSchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["local","live"]},
		"channel_strategy":{"type":"string","enum":["named","run","loop_run"]},
		"channel_id":{"type":"string"},
		"bounds":{
			"type":"object",
			"properties":{
				"max_wakes":{"type":"integer"},
				"max_wake_wall_time":{"type":"string"},
				"max_total_wall_time":{"type":"string"},
				"max_input_tokens":{"type":"integer"},
				"max_output_tokens":{"type":"integer"},
				"max_wake_depth":{"type":"integer"},
				"coalesce_window":{"type":"string"}
			},
			"additionalProperties":false
		}
	},
	"additionalProperties":false
}`
