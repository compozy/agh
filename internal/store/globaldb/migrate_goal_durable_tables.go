package globaldb

const loopGoalTurnsTableStatement = `CREATE TABLE loop_goal_turns (
	loop_run_id       TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
	seq               INTEGER NOT NULL CHECK (seq >= 1),
	generation        INTEGER NOT NULL CHECK (generation >= 1),
	node_id           TEXT NOT NULL CHECK (length(trim(node_id)) > 0),
	item_index        INTEGER NOT NULL DEFAULT 0 CHECK (item_index >= 0),
	turn              INTEGER NOT NULL CHECK (turn >= 1),
	session_id        TEXT NOT NULL CHECK (length(trim(session_id)) > 0),
	binding_handle    TEXT NOT NULL CHECK (length(trim(binding_handle)) > 0),
	binding_epoch     INTEGER NOT NULL CHECK (binding_epoch >= 1),
	prompt_id         TEXT NOT NULL CHECK (length(trim(prompt_id)) > 0),
	prompt_attempt    INTEGER NOT NULL DEFAULT 0 CHECK (prompt_attempt >= 0),
	usage_base_tokens INTEGER NOT NULL DEFAULT 0 CHECK (usage_base_tokens >= 0),
	result_status     TEXT CHECK (
		result_status IS NULL OR result_status IN ('completed','invalid-result','failed','ambiguous')
	),
	stop_reason       TEXT CHECK (
		stop_reason IS NULL OR stop_reason IN ('end_turn','max_tokens','max_turn_requests','refusal','cancelled')
	),
	reason_code       TEXT,
	verdict_outcome   TEXT CHECK (
		verdict_outcome IS NULL OR verdict_outcome IN ('approved','rejected','blocked','error','timeout','invalid_output')
	),
	blocking_json     TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(blocking_json)),
	evidence_ref      TEXT,
	prompt_ref        TEXT,
	tokens_used       INTEGER CHECK (tokens_used IS NULL OR tokens_used >= 0),
	actor_kind        TEXT NOT NULL CHECK (length(trim(actor_kind)) > 0),
	actor_id          TEXT NOT NULL CHECK (length(trim(actor_id)) > 0),
	started_at        TIMESTAMP NOT NULL,
	ended_at          TIMESTAMP CHECK (ended_at IS NULL OR ended_at >= started_at),
	CHECK (
		(result_status IS NULL AND ended_at IS NULL AND stop_reason IS NULL
		 AND reason_code IS NULL AND verdict_outcome IS NULL)
		OR
		(result_status = 'completed' AND ended_at IS NOT NULL AND stop_reason IS NOT NULL AND reason_code IS NULL)
		OR
		(result_status = 'invalid-result' AND ended_at IS NOT NULL AND stop_reason IS NULL
		 AND reason_code = 'goal_stop_reason_invalid' AND verdict_outcome IS NULL)
		OR
		(result_status = 'failed' AND ended_at IS NOT NULL AND stop_reason IS NULL
		 AND reason_code = 'goal_prompt_request_failed' AND verdict_outcome IS NULL)
		OR
		(result_status = 'ambiguous' AND ended_at IS NOT NULL AND stop_reason IS NULL
		 AND reason_code IN ('goal_recovery_ambiguous','goal_control_revoked_in_flight')
		 AND verdict_outcome IS NULL)
	),
	PRIMARY KEY (loop_run_id, generation, node_id, item_index, turn),
	UNIQUE (loop_run_id, seq),
	UNIQUE (loop_run_id, prompt_id)
)`

const loopGoalCheckpointsTableStatement = `CREATE TABLE loop_goal_checkpoints (
	loop_run_id       TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
	generation        INTEGER NOT NULL CHECK (generation >= 1),
	node_id           TEXT NOT NULL CHECK (length(trim(node_id)) > 0),
	item_index        INTEGER NOT NULL DEFAULT 0 CHECK (item_index >= 0),
	control_epoch     INTEGER NOT NULL DEFAULT 1 CHECK (control_epoch >= 1),
	control_actor_kind TEXT,
	control_actor_id  TEXT,
	control_requested_at TIMESTAMP,
	phase             TEXT NOT NULL CHECK (phase IN (
		'idle','preparing','queued','prompting','compacting','judging','persisting','awaiting_control','terminal'
	)),
	goal_status       TEXT NOT NULL CHECK (goal_status IN (
		'active','paused','blocked','usage-limited','budget-limited','complete'
	)),
	turns_used        INTEGER NOT NULL DEFAULT 0 CHECK (turns_used >= 0),
	turn_limit        INTEGER NOT NULL CHECK (turn_limit >= 1),
	broken_streak     INTEGER NOT NULL DEFAULT 0 CHECK (broken_streak >= 0),
	recovery_streak   INTEGER NOT NULL DEFAULT 0 CHECK (recovery_streak >= 0),
	task_run_id       TEXT,
	queue_entry_id    TEXT,
	prompt_id         TEXT,
	prompt_kind       TEXT CHECK (prompt_kind IS NULL OR prompt_kind IN ('work','continuation','compact')),
	prompt_attempt    INTEGER NOT NULL DEFAULT 0 CHECK (prompt_attempt >= 0),
	context_state     TEXT NOT NULL DEFAULT 'unknown' CHECK (context_state IN ('known','unknown','pending')),
	usage_sequence    INTEGER CHECK (usage_sequence IS NULL OR usage_sequence >= 0),
	usage_pending_after_sequence INTEGER CHECK (usage_pending_after_sequence IS NULL OR usage_pending_after_sequence >= 0),
	session_id        TEXT,
	binding_handle    TEXT,
	binding_epoch     INTEGER CHECK (binding_epoch IS NULL OR binding_epoch >= 1),
	context_nudge_ratio REAL NOT NULL CHECK (context_nudge_ratio >= 0.0 AND context_nudge_ratio <= 1.0),
	control_grant_id  INTEGER NOT NULL DEFAULT 0 CHECK (control_grant_id >= 0),
	control_grant_kind TEXT CHECK (
		control_grant_kind IS NULL OR control_grant_kind IN ('turn-extension','budget','reseed','plain-resume')
	),
	control_grant_cause TEXT,
	control_grant_turn INTEGER CHECK (control_grant_turn IS NULL OR control_grant_turn >= 0),
	control_grant_scope TEXT CHECK (
		control_grant_scope IS NULL OR control_grant_scope IN (
			'turn-limit','settle-current','work-and-settle','rotate-binding','reactivate'
		)
	),
	control_grant_consumed INTEGER NOT NULL DEFAULT 1 CHECK (control_grant_consumed IN (0,1)),
	judge_attempt_id  TEXT,
	compaction_cancel_prompt_id TEXT,
	compaction_cancel_cause TEXT CHECK (compaction_cancel_cause IS NULL OR compaction_cancel_cause = 'timeout'),
	compaction_cancel_requested_at TIMESTAMP,
	report_prompt_id  TEXT,
	report_status     TEXT CHECK (report_status IS NULL OR report_status IN ('complete','blocked')),
	report_evidence_ref TEXT,
	report_binding_epoch INTEGER CHECK (report_binding_epoch IS NULL OR report_binding_epoch >= 1),
	report_actor_kind TEXT,
	report_actor_id   TEXT,
	report_recorded_at TIMESTAMP,
	updated_at        TIMESTAMP NOT NULL,
	CHECK (
		(context_state = 'known' AND usage_sequence IS NOT NULL AND usage_pending_after_sequence IS NULL)
		OR
		(context_state = 'unknown' AND usage_sequence IS NULL AND usage_pending_after_sequence IS NULL)
		OR
		(context_state = 'pending' AND (
			usage_pending_after_sequence IS NULL OR usage_pending_after_sequence = usage_sequence
		))
	),
	CHECK (
		(control_actor_kind IS NULL AND control_actor_id IS NULL AND control_requested_at IS NULL)
		OR
		(control_actor_kind IS NOT NULL AND control_actor_id IS NOT NULL AND control_requested_at IS NOT NULL)
	),
	CHECK (
		(control_grant_id = 0 AND control_grant_kind IS NULL AND control_grant_cause IS NULL
		 AND control_grant_turn IS NULL AND control_grant_scope IS NULL AND control_grant_consumed = 1)
		OR
		(control_grant_id >= 1 AND control_grant_kind IS NOT NULL AND control_grant_cause IS NOT NULL
		 AND control_grant_scope IS NOT NULL AND control_grant_turn IS NOT NULL)
	),
	CHECK (
		(compaction_cancel_cause IS NULL AND compaction_cancel_prompt_id IS NULL AND compaction_cancel_requested_at IS NULL)
		OR
		(compaction_cancel_cause = 'timeout' AND compaction_cancel_prompt_id IS NOT NULL
		 AND compaction_cancel_requested_at IS NOT NULL)
	),
	CHECK (
		(report_status IS NULL AND report_prompt_id IS NULL AND report_evidence_ref IS NULL
		 AND report_binding_epoch IS NULL AND report_actor_kind IS NULL AND report_actor_id IS NULL
		 AND report_recorded_at IS NULL)
		OR
		(report_status IS NOT NULL AND report_prompt_id IS NOT NULL AND report_binding_epoch IS NOT NULL
		 AND report_actor_kind IS NOT NULL AND report_actor_id IS NOT NULL AND report_recorded_at IS NOT NULL)
	),
	PRIMARY KEY (loop_run_id, generation, node_id, item_index)
)`
