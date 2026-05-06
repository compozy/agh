package globaldb

var taskRunReviewTableSchemaStatementList = []string{
	`CREATE TABLE IF NOT EXISTS task_run_reviews (
		review_id            TEXT PRIMARY KEY,
		task_id              TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		run_id               TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE,
		parent_review_id     TEXT REFERENCES task_run_reviews(review_id) ON DELETE SET NULL,
		policy               TEXT NOT NULL CHECK (policy IN ('none', 'on_success', 'on_failure', 'always')),
		review_round         INTEGER NOT NULL CHECK (review_round >= 0),
		attempt              INTEGER NOT NULL CHECK (attempt > 0),
		status               TEXT NOT NULL CHECK (
			status IN ('requested', 'routed', 'in_review', 'recorded', 'circuit_opened', 'canceled')
		),
		outcome              TEXT CHECK (
			outcome IS NULL OR outcome IN (
				'approved', 'rejected', 'blocked', 'error', 'timeout', 'invalid_output'
			)
		),
		confidence           REAL CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1)),
		reason               TEXT NOT NULL DEFAULT '',
		delivery_id          TEXT,
		missing_work_json    TEXT NOT NULL DEFAULT '[]',
		next_round_guidance  TEXT NOT NULL DEFAULT '',
		review_text          TEXT NOT NULL DEFAULT '',
		reviewer_session_id  TEXT,
		reviewer_agent_name  TEXT NOT NULL DEFAULT '',
		reviewer_peer_id     TEXT NOT NULL DEFAULT '',
		reviewer_channel_id  TEXT NOT NULL DEFAULT '',
		reviewed_by_kind     TEXT NOT NULL DEFAULT '',
		reviewed_by_ref      TEXT NOT NULL DEFAULT '',
		requested_at         TEXT NOT NULL,
		routed_at            TEXT,
		started_at           TEXT,
		reviewed_at          TEXT,
		deadline_at          TEXT,
		created_at           TEXT NOT NULL,
		updated_at           TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_task_round_attempt
		ON task_run_reviews(task_id, review_round, attempt);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_run_reviews_run_round_attempt
		ON task_run_reviews(run_id, review_round, attempt);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_run_status
		ON task_run_reviews(run_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_deadline
		ON task_run_reviews(status, deadline_at);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_reviewer_session
		ON task_run_reviews(reviewer_session_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_reviewer_agent
		ON task_run_reviews(reviewer_agent_name, status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_reviewer_peer
		ON task_run_reviews(reviewer_peer_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_run_reviews_reviewer_channel
		ON task_run_reviews(reviewer_channel_id, status);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_run_reviews_reviewer_session_active
		ON task_run_reviews(reviewer_session_id)
		WHERE reviewer_session_id IS NOT NULL AND status IN ('routed', 'in_review');`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_run_reviews_delivery
		ON task_run_reviews(review_id, delivery_id)
		WHERE delivery_id IS NOT NULL;`,
}

var taskReviewGateIndexStatementList = []string{
	`CREATE INDEX IF NOT EXISTS idx_tasks_review_policy ON tasks(review_policy);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_review_round ON tasks(review_round);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_parent_run ON task_runs(parent_run_id);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_review_request
		ON task_runs(review_request_id)
		WHERE review_request_id IS NOT NULL;`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_runs_review_id
		ON task_runs(review_id)
		WHERE review_id IS NOT NULL;`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_task_review_round
		ON task_runs(task_id, review_round)
		WHERE review_round > 0;`,
}

func taskRunReviewTableSchemaStatements() []string {
	return append([]string(nil), taskRunReviewTableSchemaStatementList...)
}

func taskReviewGateIndexStatements() []string {
	return append([]string(nil), taskReviewGateIndexStatementList...)
}
