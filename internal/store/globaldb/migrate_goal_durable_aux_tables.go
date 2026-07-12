package globaldb

const loopGoalJudgeAttemptsTableStatement = `CREATE TABLE loop_goal_judge_attempts (
	attempt_id        TEXT PRIMARY KEY CHECK (length(trim(attempt_id)) > 0),
	loop_run_id       TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
	generation        INTEGER NOT NULL CHECK (generation >= 1),
	node_id           TEXT NOT NULL CHECK (length(trim(node_id)) > 0),
	item_index        INTEGER NOT NULL DEFAULT 0 CHECK (item_index >= 0),
	turn              INTEGER NOT NULL CHECK (turn >= 1),
	judge_digest      TEXT NOT NULL CHECK (length(trim(judge_digest)) > 0),
	status            TEXT NOT NULL CHECK (status IN ('running','completed','ambiguous')),
	outcome           TEXT CHECK (
		outcome IS NULL OR outcome IN ('approved','rejected','blocked','error','timeout','invalid_output')
	),
	blocking_json     TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(blocking_json)),
	evidence_ref      TEXT,
	tokens_used       INTEGER CHECK (tokens_used IS NULL OR tokens_used >= 0),
	usage_base_tokens INTEGER NOT NULL DEFAULT 0 CHECK (usage_base_tokens >= 0),
	started_at        TIMESTAMP NOT NULL,
	completed_at      TIMESTAMP CHECK (completed_at IS NULL OR completed_at >= started_at),
	CHECK (
		(status = 'running' AND outcome IS NULL AND completed_at IS NULL)
		OR
		(status = 'completed' AND outcome IS NOT NULL AND completed_at IS NOT NULL)
		OR
		(status = 'ambiguous' AND outcome IS NULL AND completed_at IS NOT NULL)
	),
	UNIQUE (loop_run_id, generation, node_id, item_index, turn)
)`

const loopSessionBindingsTableStatement = `CREATE TABLE loop_session_bindings (
	loop_run_id       TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
	handle            TEXT NOT NULL CHECK (length(trim(handle)) > 0),
	binding_epoch     INTEGER NOT NULL CHECK (binding_epoch >= 1),
	binding_attempt_id TEXT NOT NULL UNIQUE CHECK (length(trim(binding_attempt_id)) > 0),
	session_id        TEXT NOT NULL CHECK (length(trim(session_id)) > 0),
	workspace_id      TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
	creation_profile_ref TEXT NOT NULL CHECK (length(trim(creation_profile_ref)) > 0),
	policy_spec_digest TEXT NOT NULL CHECK (length(trim(policy_spec_digest)) > 0),
	creation_digest   TEXT NOT NULL CHECK (length(trim(creation_digest)) > 0),
	ownership         TEXT NOT NULL CHECK (ownership IN ('origin-borrowed','run-owned')),
	state             TEXT NOT NULL CHECK (state IN ('creating','active','failed','closed','reseeded')),
	failure_code      TEXT,
	created_at        TIMESTAMP NOT NULL,
	activated_at      TIMESTAMP CHECK (activated_at IS NULL OR activated_at >= created_at),
	failed_at         TIMESTAMP CHECK (failed_at IS NULL OR failed_at >= created_at),
	closed_at         TIMESTAMP CHECK (closed_at IS NULL OR closed_at >= created_at),
	CHECK (
		(state = 'creating' AND activated_at IS NULL AND failed_at IS NULL AND failure_code IS NULL AND closed_at IS NULL)
		OR
		(state = 'active' AND activated_at IS NOT NULL AND failed_at IS NULL AND failure_code IS NULL AND closed_at IS NULL)
		OR
		(state = 'failed' AND activated_at IS NULL AND failed_at IS NOT NULL
		 AND length(trim(failure_code)) > 0 AND closed_at IS NULL)
		OR
		(state IN ('closed','reseeded') AND activated_at IS NOT NULL AND failed_at IS NULL
		 AND failure_code IS NULL AND closed_at IS NOT NULL)
	),
	CHECK (ownership = 'run-owned' OR (binding_epoch = 1 AND state IN ('active','closed'))),
	PRIMARY KEY (loop_run_id, handle, binding_epoch)
)`

const loopGoalSessionOutboxTableStatement = `CREATE TABLE loop_goal_session_outbox (
	id                INTEGER PRIMARY KEY AUTOINCREMENT,
	event_id          TEXT NOT NULL UNIQUE CHECK (length(trim(event_id)) > 0),
	workspace_id      TEXT NOT NULL CHECK (length(trim(workspace_id)) > 0),
	origin_session_id TEXT NOT NULL CHECK (length(trim(origin_session_id)) > 0),
	loop_run_id       TEXT NOT NULL REFERENCES loop_runs(id) ON DELETE CASCADE,
	bound_session_id  TEXT,
	cause             TEXT NOT NULL CHECK (cause IN ('start','replace','status','clear','reseed')),
	created_at        TIMESTAMP NOT NULL,
	delivered_at      TIMESTAMP CHECK (delivered_at IS NULL OR delivered_at >= created_at)
)`
