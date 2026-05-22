package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

var _ taskpkg.ExecutionProfileStore = (*GlobalDB)(nil)

const (
	profileRoleWorker      = "worker"
	profileRoleReview      = "review"
	profileRoleParticipant = "participant"

	profilePreferenceAllowed   = "allowed"
	profilePreferencePreferred = "preferred"
	profilePreferenceRequired  = "required"
)

type taskExecutionProfileRow struct {
	profile   taskpkg.ExecutionProfile
	createdAt string
	updatedAt string
}

// GetExecutionProfile returns the persisted typed execution profile for one task.
func (g *GlobalDB) GetExecutionProfile(
	ctx context.Context,
	taskID string,
) (taskpkg.ExecutionProfile, error) {
	if err := g.checkReady(ctx, "get task execution profile"); err != nil {
		return taskpkg.ExecutionProfile{}, err
	}
	trimmedID, err := requireTaskValue(taskID, "task execution profile task id")
	if err != nil {
		return taskpkg.ExecutionProfile{}, err
	}

	profile, found, err := loadExecutionProfile(ctx, g.db, trimmedID)
	if err != nil {
		return taskpkg.ExecutionProfile{}, err
	}
	if !found {
		return taskpkg.ExecutionProfile{}, taskpkg.ErrExecutionProfileNotFound
	}
	return profile, nil
}

// UpsertExecutionProfile replaces one task-owned execution profile and its selectors atomically.
func (g *GlobalDB) UpsertExecutionProfile(
	ctx context.Context,
	profile *taskpkg.ExecutionProfile,
) (stored taskpkg.ExecutionProfile, err error) {
	if err := g.checkReady(ctx, "upsert task execution profile"); err != nil {
		return taskpkg.ExecutionProfile{}, err
	}
	normalized, err := profile.Normalize(taskpkg.DefaultExecutionProfileValidationOptions())
	if err != nil {
		return taskpkg.ExecutionProfile{}, err
	}

	if err := g.withTaskImmediateTransaction(ctx, "upsert task execution profile", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalized.TaskID); err != nil {
			return err
		}

		now := g.now().UTC()
		existing, found, loadErr := loadExecutionProfile(ctx, exec, normalized.TaskID)
		if loadErr != nil {
			return loadErr
		}
		if found {
			normalized.CreatedAt = existing.CreatedAt
		}
		if normalized.CreatedAt.IsZero() {
			normalized.CreatedAt = now
		}
		normalized.UpdatedAt = now

		if err := upsertExecutionProfileRow(ctx, exec, &normalized); err != nil {
			return err
		}
		if err := replaceExecutionProfileSelectors(ctx, exec, &normalized); err != nil {
			return err
		}
		reloaded, reloadErr := reloadExecutionProfile(ctx, exec, normalized.TaskID)
		if reloadErr != nil {
			return reloadErr
		}
		stored = reloaded
		return nil
	}); err != nil {
		return taskpkg.ExecutionProfile{}, err
	}
	return stored, nil
}

// DeleteExecutionProfile removes one profile and its selector rows.
func (g *GlobalDB) DeleteExecutionProfile(ctx context.Context, taskID string) error {
	if err := g.checkReady(ctx, "delete task execution profile"); err != nil {
		return err
	}
	trimmedID, err := requireTaskValue(taskID, "task execution profile task id")
	if err != nil {
		return err
	}

	return g.withTaskImmediateTransaction(ctx, "delete task execution profile", func(exec taskSQLExecutor) error {
		if err := deleteExecutionProfileSelectors(ctx, exec, trimmedID); err != nil {
			return err
		}
		result, err := exec.ExecContext(ctx, `DELETE FROM task_execution_profiles WHERE task_id = ?`, trimmedID)
		if err != nil {
			return fmt.Errorf("store: delete task execution profile %q: %w", trimmedID, err)
		}
		return requireRowsAffected(
			result,
			taskpkg.ErrExecutionProfileNotFound,
			trimmedID,
			"task execution profile",
		)
	})
}

func loadExecutionProfile(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (taskpkg.ExecutionProfile, bool, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT
			task_id, coordinator_mode, coordinator_agent_name, coordinator_provider,
			coordinator_model, coordinator_guidance, worker_mode, worker_agent_name,
			worker_provider, worker_model, review_agent_name, review_provider,
			review_model, sandbox_mode, sandbox_ref, created_at, updated_at
		 FROM task_execution_profiles
		 WHERE task_id = ?`,
		taskID,
	)
	profile, err := scanExecutionProfileRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ExecutionProfile{}, false, nil
		}
		return taskpkg.ExecutionProfile{}, false, err
	}
	if err := loadExecutionProfileSelectors(ctx, exec, &profile); err != nil {
		return taskpkg.ExecutionProfile{}, false, err
	}
	return profile, true, nil
}

func reloadExecutionProfile(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (taskpkg.ExecutionProfile, error) {
	profile, found, err := loadExecutionProfile(ctx, exec, taskID)
	if err != nil {
		return taskpkg.ExecutionProfile{}, err
	}
	if !found {
		return taskpkg.ExecutionProfile{}, taskpkg.ErrExecutionProfileNotFound
	}
	return profile, nil
}

func scanExecutionProfileRow(scanner rowScanner) (taskpkg.ExecutionProfile, error) {
	var row taskExecutionProfileRow
	if err := scanner.Scan(
		&row.profile.TaskID,
		&row.profile.Coordinator.Mode,
		&row.profile.Coordinator.AgentName,
		&row.profile.Coordinator.Provider,
		&row.profile.Coordinator.Model,
		&row.profile.Coordinator.Guidance,
		&row.profile.Worker.Mode,
		&row.profile.Worker.AgentName,
		&row.profile.Worker.Provider,
		&row.profile.Worker.Model,
		&row.profile.Review.AgentName,
		&row.profile.Review.Provider,
		&row.profile.Review.Model,
		&row.profile.Sandbox.Mode,
		&row.profile.Sandbox.SandboxRef,
		&row.createdAt,
		&row.updatedAt,
	); err != nil {
		return taskpkg.ExecutionProfile{}, err
	}

	createdAt, err := store.ParseTimestamp(row.createdAt)
	if err != nil {
		return taskpkg.ExecutionProfile{}, fmt.Errorf("store: parse task execution profile created_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(row.updatedAt)
	if err != nil {
		return taskpkg.ExecutionProfile{}, fmt.Errorf("store: parse task execution profile updated_at: %w", err)
	}
	row.profile.CreatedAt = createdAt
	row.profile.UpdatedAt = updatedAt
	return row.profile, nil
}

func upsertExecutionProfileRow(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_execution_profiles (
			task_id, coordinator_mode, coordinator_agent_name, coordinator_provider,
			coordinator_model, coordinator_guidance, worker_mode, worker_agent_name,
			worker_provider, worker_model, review_agent_name, review_provider,
			review_model, sandbox_mode, sandbox_ref, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET
			coordinator_mode = excluded.coordinator_mode,
			coordinator_agent_name = excluded.coordinator_agent_name,
			coordinator_provider = excluded.coordinator_provider,
			coordinator_model = excluded.coordinator_model,
			coordinator_guidance = excluded.coordinator_guidance,
			worker_mode = excluded.worker_mode,
			worker_agent_name = excluded.worker_agent_name,
			worker_provider = excluded.worker_provider,
			worker_model = excluded.worker_model,
			review_agent_name = excluded.review_agent_name,
			review_provider = excluded.review_provider,
			review_model = excluded.review_model,
			sandbox_mode = excluded.sandbox_mode,
			sandbox_ref = excluded.sandbox_ref,
			updated_at = excluded.updated_at`,
		profile.TaskID,
		string(profile.Coordinator.Mode),
		profile.Coordinator.AgentName,
		profile.Coordinator.Provider,
		profile.Coordinator.Model,
		profile.Coordinator.Guidance,
		string(profile.Worker.Mode),
		profile.Worker.AgentName,
		profile.Worker.Provider,
		profile.Worker.Model,
		profile.Review.AgentName,
		profile.Review.Provider,
		profile.Review.Model,
		string(profile.Sandbox.Mode),
		profile.Sandbox.SandboxRef,
		store.FormatTimestamp(profile.CreatedAt),
		store.FormatTimestamp(profile.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert task execution profile %q: %w", profile.TaskID, err)
	}
	return nil
}

func replaceExecutionProfileSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	if err := deleteExecutionProfileSelectors(ctx, exec, profile.TaskID); err != nil {
		return err
	}
	if err := insertTaskProfileAgentSelectors(ctx, exec, profile); err != nil {
		return err
	}
	if err := insertTaskProfileChannelSelectors(ctx, exec, profile); err != nil {
		return err
	}
	if err := insertTaskProfilePeerSelectors(ctx, exec, profile); err != nil {
		return err
	}
	return insertTaskProfileCapabilitySelectors(ctx, exec, profile)
}

func deleteExecutionProfileSelectors(ctx context.Context, exec taskSQLExecutor, taskID string) error {
	for _, table := range []string{
		"task_profile_agents",
		"task_profile_channels",
		"task_profile_peers",
		"task_profile_capabilities",
	} {
		query := fmt.Sprintf("DELETE FROM %s WHERE task_id = ?", table)
		if _, err := exec.ExecContext(ctx, query, taskID); err != nil {
			return fmt.Errorf("store: delete %s rows for task %q: %w", table, taskID, err)
		}
	}
	return nil
}

func insertTaskProfileAgentSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	rows := []profileSelectorRow{
		{profileRoleWorker, profilePreferenceAllowed, profile.Worker.AllowedAgentNames},
		{profileRoleWorker, profilePreferencePreferred, profile.Worker.PreferredAgentNames},
		{profileRoleReview, profilePreferenceAllowed, profile.Review.AllowedAgentNames},
		{profileRoleReview, profilePreferencePreferred, profile.Review.PreferredAgentNames},
		{profileRoleParticipant, profilePreferenceAllowed, profile.Participants.AllowedAgentNames},
		{profileRoleParticipant, profilePreferencePreferred, profile.Participants.PreferredAgentNames},
	}
	return insertTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_agents",
		"agent_name",
		profile.TaskID,
		rows,
	)
}

func insertTaskProfileChannelSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	rows := []profileSelectorRow{
		{profileRoleReview, profilePreferenceAllowed, profile.Review.AllowedChannelIDs},
		{profileRoleReview, profilePreferencePreferred, profile.Review.PreferredChannelIDs},
		{profileRoleParticipant, profilePreferenceAllowed, profile.Participants.AllowedChannelIDs},
		{profileRoleParticipant, profilePreferencePreferred, profile.Participants.PreferredChannelIDs},
	}
	return insertTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_channels",
		"channel_id",
		profile.TaskID,
		rows,
	)
}

func insertTaskProfilePeerSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	rows := []profileSelectorRow{
		{profileRoleReview, profilePreferenceAllowed, profile.Review.AllowedPeerIDs},
		{profileRoleReview, profilePreferencePreferred, profile.Review.PreferredPeerIDs},
		{profileRoleParticipant, profilePreferenceAllowed, profile.Participants.AllowedPeerIDs},
		{profileRoleParticipant, profilePreferencePreferred, profile.Participants.PreferredPeerIDs},
	}
	return insertTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_peers",
		"peer_id",
		profile.TaskID,
		rows,
	)
}

func insertTaskProfileCapabilitySelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	rows := []profileSelectorRow{
		{profileRoleWorker, profilePreferenceRequired, profile.Worker.RequiredCapabilities},
		{profileRoleWorker, profilePreferencePreferred, profile.Worker.PreferredCapabilities},
		{profileRoleReview, profilePreferenceRequired, profile.Review.RequiredCapabilities},
		{profileRoleReview, profilePreferencePreferred, profile.Review.PreferredCapabilities},
		{profileRoleParticipant, profilePreferenceRequired, profile.Participants.RequiredCapabilities},
		{profileRoleParticipant, profilePreferencePreferred, profile.Participants.PreferredCapabilities},
	}
	return insertTaskProfileSelectors(ctx, exec, "task_profile_capabilities", "capability_id", profile.TaskID, rows)
}

type profileSelectorRow struct {
	role       string
	preference string
	values     []string
}

func insertTaskProfileSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	table string,
	valueColumn string,
	taskID string,
	rows []profileSelectorRow,
) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (task_id, role, preference, %s) VALUES (?, ?, ?, ?)",
		table,
		valueColumn,
	)
	for _, row := range rows {
		for _, value := range row.values {
			if _, err := exec.ExecContext(ctx, query, taskID, row.role, row.preference, value); err != nil {
				return fmt.Errorf("store: insert %s selector for task %q: %w", table, taskID, err)
			}
		}
	}
	return nil
}

func loadExecutionProfileSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	if err := loadTaskProfileAgentSelectors(ctx, exec, profile); err != nil {
		return err
	}
	if err := loadTaskProfileChannelSelectors(ctx, exec, profile); err != nil {
		return err
	}
	if err := loadTaskProfilePeerSelectors(ctx, exec, profile); err != nil {
		return err
	}
	return loadTaskProfileCapabilitySelectors(ctx, exec, profile)
}

func loadTaskProfileAgentSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	return queryTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_agents",
		"agent_name",
		profile.TaskID,
		func(row loadedProfileSelectorRow) {
			switch {
			case row.matches(profileRoleWorker, profilePreferenceAllowed):
				profile.Worker.AllowedAgentNames = append(profile.Worker.AllowedAgentNames, row.value)
			case row.matches(profileRoleWorker, profilePreferencePreferred):
				profile.Worker.PreferredAgentNames = append(profile.Worker.PreferredAgentNames, row.value)
			case row.matches(profileRoleReview, profilePreferenceAllowed):
				profile.Review.AllowedAgentNames = append(profile.Review.AllowedAgentNames, row.value)
			case row.matches(profileRoleReview, profilePreferencePreferred):
				profile.Review.PreferredAgentNames = append(profile.Review.PreferredAgentNames, row.value)
			case row.matches(profileRoleParticipant, profilePreferenceAllowed):
				profile.Participants.AllowedAgentNames = append(profile.Participants.AllowedAgentNames, row.value)
			case row.matches(profileRoleParticipant, profilePreferencePreferred):
				profile.Participants.PreferredAgentNames = append(profile.Participants.PreferredAgentNames, row.value)
			}
		},
	)
}

func loadTaskProfileChannelSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	return queryTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_channels",
		"channel_id",
		profile.TaskID,
		func(row loadedProfileSelectorRow) {
			switch {
			case row.matches(profileRoleReview, profilePreferenceAllowed):
				profile.Review.AllowedChannelIDs = append(profile.Review.AllowedChannelIDs, row.value)
			case row.matches(profileRoleReview, profilePreferencePreferred):
				profile.Review.PreferredChannelIDs = append(profile.Review.PreferredChannelIDs, row.value)
			case row.matches(profileRoleParticipant, profilePreferenceAllowed):
				profile.Participants.AllowedChannelIDs = append(profile.Participants.AllowedChannelIDs, row.value)
			case row.matches(profileRoleParticipant, profilePreferencePreferred):
				profile.Participants.PreferredChannelIDs = append(profile.Participants.PreferredChannelIDs, row.value)
			}
		},
	)
}

func loadTaskProfilePeerSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	return queryTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_peers",
		"peer_id",
		profile.TaskID,
		func(row loadedProfileSelectorRow) {
			switch {
			case row.matches(profileRoleReview, profilePreferenceAllowed):
				profile.Review.AllowedPeerIDs = append(profile.Review.AllowedPeerIDs, row.value)
			case row.matches(profileRoleReview, profilePreferencePreferred):
				profile.Review.PreferredPeerIDs = append(profile.Review.PreferredPeerIDs, row.value)
			case row.matches(profileRoleParticipant, profilePreferenceAllowed):
				profile.Participants.AllowedPeerIDs = append(profile.Participants.AllowedPeerIDs, row.value)
			case row.matches(profileRoleParticipant, profilePreferencePreferred):
				profile.Participants.PreferredPeerIDs = append(profile.Participants.PreferredPeerIDs, row.value)
			}
		},
	)
}

func loadTaskProfileCapabilitySelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	profile *taskpkg.ExecutionProfile,
) error {
	return queryTaskProfileSelectors(
		ctx,
		exec,
		"task_profile_capabilities",
		"capability_id",
		profile.TaskID,
		func(row loadedProfileSelectorRow) {
			switch {
			case row.matches(profileRoleWorker, profilePreferenceRequired):
				profile.Worker.RequiredCapabilities = append(profile.Worker.RequiredCapabilities, row.value)
			case row.matches(profileRoleWorker, profilePreferencePreferred):
				profile.Worker.PreferredCapabilities = append(profile.Worker.PreferredCapabilities, row.value)
			case row.matches(profileRoleReview, profilePreferenceRequired):
				profile.Review.RequiredCapabilities = append(profile.Review.RequiredCapabilities, row.value)
			case row.matches(profileRoleReview, profilePreferencePreferred):
				profile.Review.PreferredCapabilities = append(profile.Review.PreferredCapabilities, row.value)
			case row.matches(profileRoleParticipant, profilePreferenceRequired):
				profile.Participants.RequiredCapabilities = append(profile.Participants.RequiredCapabilities, row.value)
			case row.matches(profileRoleParticipant, profilePreferencePreferred):
				profile.Participants.PreferredCapabilities = append(
					profile.Participants.PreferredCapabilities,
					row.value,
				)
			}
		},
	)
}

type loadedProfileSelectorRow struct {
	role       string
	preference string
	value      string
}

func (r loadedProfileSelectorRow) matches(role string, preference string) bool {
	return r.role == role && r.preference == preference
}

func queryTaskProfileSelectors(
	ctx context.Context,
	exec taskSQLExecutor,
	table string,
	valueColumn string,
	taskID string,
	apply func(loadedProfileSelectorRow),
) (err error) {
	query := fmt.Sprintf(
		`SELECT role, preference, %s
		 FROM %s
		 WHERE task_id = ?
		 ORDER BY role ASC, preference ASC, %s ASC`,
		valueColumn,
		table,
		valueColumn,
	)
	rows, err := exec.QueryContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("store: query %s selectors for task %q: %w", table, taskID, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close %s selector rows for task %q: %w", table, taskID, closeErr)
		}
	}()

	for rows.Next() {
		var row loadedProfileSelectorRow
		if err := rows.Scan(&row.role, &row.preference, &row.value); err != nil {
			return fmt.Errorf("store: scan %s selector for task %q: %w", table, taskID, err)
		}
		apply(row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate %s selectors for task %q: %w", table, taskID, err)
	}
	return nil
}
