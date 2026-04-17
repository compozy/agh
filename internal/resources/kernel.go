package resources

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

const (
	defaultMaxSpecBytes       = 256 << 10
	defaultMaxSnapshotRecords = 256
	defaultMaxSnapshotBytes   = 2 << 20

	rawRecordSelectQuery = `SELECT
		kind,
		id,
		version,
		scope_kind,
		scope_id,
		owner_kind,
		owner_id,
		source_kind,
		source_id,
		spec_json,
		created_at,
		updated_at
	FROM resource_records`
	selectRecordByKeyQuery = rawRecordSelectQuery + `
	WHERE kind = ? AND id = ?`
	selectSourceRecordsQuery = rawRecordSelectQuery + `
	WHERE source_kind = ? AND source_id = ?
	ORDER BY kind ASC, id ASC`
	selectSourceStateQuery = `SELECT
		source_kind,
		source_id,
		session_nonce,
		last_snapshot_version,
		updated_at
	FROM resource_source_state
	WHERE source_kind = ? AND source_id = ?`
	resourceRecordOrderBy      = " ORDER BY kind ASC, id ASC"
	deleteRecordByVersionQuery = `
		DELETE FROM resource_records
		WHERE kind = ? AND id = ? AND version = ?`
	deleteStaleSourceRecordQuery = `
		DELETE FROM resource_records
		WHERE kind = ? AND id = ? AND source_kind = ? AND source_id = ?`
	deleteSourceRecordsQuery = `
		DELETE FROM resource_records
		WHERE source_kind = ? AND source_id = ?`
	deleteSourceStateQuery = `
		DELETE FROM resource_source_state
		WHERE source_kind = ? AND source_id = ?`
	activateSourceStateQuery = `INSERT INTO resource_source_state (
		source_kind,
		source_id,
		session_nonce,
		last_snapshot_version,
		updated_at
	) VALUES (?, ?, ?, 0, ?)
	ON CONFLICT(source_kind, source_id)
	DO UPDATE SET
		session_nonce = excluded.session_nonce,
		last_snapshot_version = 0,
		updated_at = excluded.updated_at`
	updateSourceStateQuery = `UPDATE resource_source_state
		SET last_snapshot_version = ?, updated_at = ?
		WHERE source_kind = ? AND source_id = ? AND session_nonce = ?`
)

// Option configures a Kernel instance.
type Option func(*Kernel)

// WithNow overrides the clock used by the kernel.
func WithNow(now func() time.Time) Option {
	return func(k *Kernel) {
		if now != nil {
			k.now = now
		}
	}
}

// WithMaxSpecBytes overrides the per-record payload ceiling.
func WithMaxSpecBytes(limit int) Option {
	return func(k *Kernel) {
		if limit > 0 {
			k.maxSpecBytes = limit
		}
	}
}

// WithMaxSnapshotRecords overrides the per-snapshot record count ceiling.
func WithMaxSnapshotRecords(limit int) Option {
	return func(k *Kernel) {
		if limit > 0 {
			k.maxSnapshotRecords = limit
		}
	}
}

// WithMaxSnapshotBytes overrides the per-snapshot byte ceiling.
func WithMaxSnapshotBytes(limit int) Option {
	return func(k *Kernel) {
		if limit > 0 {
			k.maxSnapshotBytes = limit
		}
	}
}

// Kernel implements the canonical raw desired-state persistence contract.
type Kernel struct {
	db *sql.DB

	now                func() time.Time
	maxSpecBytes       int
	maxSnapshotRecords int
	maxSnapshotBytes   int

	sourceLocksMu sync.Mutex
	sourceLocks   map[string]*sourceLock
}

var _ RawStore = (*Kernel)(nil)
var _ SourceSessionManager = (*Kernel)(nil)

type sourceLock struct {
	mu   sync.Mutex
	refs int
}

// NewKernel constructs a new raw resource persistence kernel over the supplied database.
func NewKernel(db *sql.DB, opts ...Option) (*Kernel, error) {
	if db == nil {
		return nil, errors.New("resources: database is required")
	}

	kernel := &Kernel{
		db:                 db,
		now:                func() time.Time { return time.Now().UTC() },
		maxSpecBytes:       defaultMaxSpecBytes,
		maxSnapshotRecords: defaultMaxSnapshotRecords,
		maxSnapshotBytes:   defaultMaxSnapshotBytes,
		sourceLocks:        make(map[string]*sourceLock),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(kernel)
		}
	}
	if kernel.now == nil {
		return nil, errors.New("resources: clock is required")
	}
	if kernel.maxSpecBytes <= 0 {
		return nil, errors.New("resources: max spec bytes must be positive")
	}
	if kernel.maxSnapshotRecords <= 0 {
		return nil, errors.New("resources: max snapshot records must be positive")
	}
	if kernel.maxSnapshotBytes <= 0 {
		return nil, errors.New("resources: max snapshot bytes must be positive")
	}
	return kernel, nil
}

// ActivateSourceSession registers the active nonce and resets the snapshot version counter for one source.
func (k *Kernel) ActivateSourceSession(
	ctx context.Context,
	actor MutationActor,
	source ResourceSource,
	sessionNonce string,
) error {
	if ctx == nil {
		return errors.New("resources: activate source session context is required")
	}

	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return err
	}
	if normalizedActor.Kind == MutationActorKindExtension {
		return fmt.Errorf("%w: extension actors cannot activate source sessions", ErrPermissionDenied)
	}

	normalizedSource := source.Normalize()
	if err := normalizedSource.Validate("source"); err != nil {
		return err
	}
	trimmedNonce := strings.TrimSpace(sessionNonce)
	if trimmedNonce == "" {
		return fmt.Errorf("%w: session_nonce is required", ErrValidation)
	}

	unlock := k.lockSource(normalizedSource)
	defer unlock()

	return k.withImmediateTransaction(ctx, "activate source session", func(conn *sql.Conn) error {
		updatedAt := store.FormatTimestamp(k.now())
		if _, err := conn.ExecContext(
			ctx,
			activateSourceStateQuery,
			normalizedSource.Kind,
			normalizedSource.ID,
			trimmedNonce,
			updatedAt,
		); err != nil {
			return fmt.Errorf(
				"resources: activate source session %q/%q: %w",
				normalizedSource.Kind,
				normalizedSource.ID,
				err,
			)
		}
		return nil
	})
}

// ResetSource deletes all source-owned records and source state in one transaction.
func (k *Kernel) ResetSource(ctx context.Context, actor MutationActor, source ResourceSource) error {
	if ctx == nil {
		return errors.New("resources: reset source context is required")
	}

	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return err
	}
	if normalizedActor.Kind == MutationActorKindExtension {
		return fmt.Errorf("%w: extension actors cannot reset sources", ErrPermissionDenied)
	}

	normalizedSource := source.Normalize()
	if err := normalizedSource.Validate("source"); err != nil {
		return err
	}

	unlock := k.lockSource(normalizedSource)
	defer unlock()

	return k.withImmediateTransaction(ctx, "reset source", func(conn *sql.Conn) error {
		if _, err := conn.ExecContext(
			ctx,
			deleteSourceRecordsQuery,
			normalizedSource.Kind,
			normalizedSource.ID,
		); err != nil {
			return fmt.Errorf(
				"resources: delete source records %q/%q: %w",
				normalizedSource.Kind,
				normalizedSource.ID,
				err,
			)
		}
		if _, err := conn.ExecContext(
			ctx,
			deleteSourceStateQuery,
			normalizedSource.Kind,
			normalizedSource.ID,
		); err != nil {
			return fmt.Errorf(
				"resources: delete source state %q/%q: %w",
				normalizedSource.Kind,
				normalizedSource.ID,
				err,
			)
		}
		return nil
	})
}

// PutRaw creates or updates one raw desired-state record using optimistic concurrency.
func (k *Kernel) PutRaw(ctx context.Context, actor MutationActor, draft RawDraft) (record RawRecord, err error) {
	if ctx == nil {
		return RawRecord{}, errors.New("resources: put raw context is required")
	}

	normalizedActor, normalizedDraft, err := k.preparePutRaw(actor, draft)
	if err != nil {
		return RawRecord{}, err
	}

	tx, err := k.db.BeginTx(ctx, nil)
	if err != nil {
		return RawRecord{}, fmt.Errorf("resources: begin put transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			joinCleanupError(&err, rollbackTx(tx))
		}
	}()

	record, err = k.putRawWithExecutor(ctx, tx, normalizedActor, normalizedDraft)
	if err != nil {
		return RawRecord{}, err
	}
	if err = tx.Commit(); err != nil {
		return RawRecord{}, fmt.Errorf("resources: commit put %q/%q: %w", record.Kind, record.ID, err)
	}
	committed = true
	return record, nil
}

// DeleteRaw deletes one raw desired-state record using optimistic concurrency.
func (k *Kernel) DeleteRaw(
	ctx context.Context,
	actor MutationActor,
	kind ResourceKind,
	id string,
	expectedVersion int64,
) (err error) {
	if ctx == nil {
		return errors.New("resources: delete raw context is required")
	}

	normalizedActor, normalizedKind, trimmedID, err := k.prepareDeleteRaw(actor, kind, id)
	if err != nil {
		return err
	}

	tx, err := k.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("resources: begin delete transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			joinCleanupError(&err, rollbackTx(tx))
		}
	}()

	err = k.deleteRawWithExecutor(ctx, tx, normalizedActor, normalizedKind, trimmedID, expectedVersion)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("resources: commit delete %q/%q: %w", normalizedKind, trimmedID, err)
	}
	committed = true
	return nil
}

// GetRaw fetches one raw desired-state record under the actor's read boundary.
func (k *Kernel) GetRaw(ctx context.Context, actor MutationActor, kind ResourceKind, id string) (RawRecord, error) {
	if ctx == nil {
		return RawRecord{}, errors.New("resources: get raw context is required")
	}

	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return RawRecord{}, err
	}
	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("kind"); err != nil {
		return RawRecord{}, err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return RawRecord{}, fmt.Errorf("%w: id is required", ErrValidation)
	}
	if !actorAllowsKind(normalizedActor, normalizedKind) {
		return RawRecord{}, fmt.Errorf("%w: actor cannot read resource kind %q", ErrPermissionDenied, normalizedKind)
	}

	record, found, err := lookupRecordWithExecutor(ctx, k.db, normalizedKind, trimmedID)
	if err != nil {
		return RawRecord{}, err
	}
	if !found {
		return RawRecord{}, ErrNotFound
	}
	if err := validateActorReadAccess(normalizedActor, record); err != nil {
		return RawRecord{}, err
	}
	return record, nil
}

// ListRaw lists raw desired-state records under the actor's read boundary.
func (k *Kernel) ListRaw(ctx context.Context, actor MutationActor, filter ResourceFilter) ([]RawRecord, error) {
	if ctx == nil {
		return nil, errors.New("resources: list raw context is required")
	}

	normalizedActor, normalizedFilter, err := k.prepareListRaw(actor, filter)
	if err != nil {
		return nil, err
	}
	if normalizedActor.Kind == MutationActorKindExtension && k.extensionReadGrantsEmpty(normalizedActor) {
		return []RawRecord{}, nil
	}

	query, args := buildListRawQuery(normalizedActor, normalizedFilter)

	rows, err := k.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("resources: query records: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	records := make([]RawRecord, 0)
	for rows.Next() {
		record, scanErr := scanRawRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if accessErr := validateActorReadAccess(normalizedActor, record); accessErr != nil {
			if errors.Is(accessErr, ErrPermissionDenied) {
				continue
			}
			return nil, accessErr
		}
		records = append(records, record)
		if normalizedFilter.Limit > 0 && len(records) == normalizedFilter.Limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resources: iterate records: %w", err)
	}
	return records, nil
}

// ApplySourceSnapshotRaw replaces one source's desired-state snapshot under optimistic source sequencing.
func (k *Kernel) ApplySourceSnapshotRaw(ctx context.Context, actor MutationActor, snapshot SourceSnapshot) error {
	if ctx == nil {
		return errors.New("resources: apply snapshot context is required")
	}

	normalizedActor, normalizedSnapshot, normalizedDrafts, err := k.prepareSnapshotApply(actor, snapshot)
	if err != nil {
		return err
	}

	unlock := k.lockSource(normalizedActor.Source)
	defer unlock()

	return k.withImmediateTransaction(ctx, "apply source snapshot", func(conn *sql.Conn) error {
		return k.applySnapshotWithExecutor(
			ctx,
			conn,
			normalizedActor,
			normalizedSnapshot,
			normalizedDrafts,
		)
	})
}

type sourceState struct {
	Source              ResourceSource
	SessionNonce        string
	LastSnapshotVersion int64
	UpdatedAt           time.Time
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func normalizeFilter(filter ResourceFilter) (ResourceFilter, error) {
	normalized := filter
	normalized.Kind = filter.Kind.Normalize()
	if normalized.Kind != "" {
		if err := normalized.Kind.Validate("filter.kind"); err != nil {
			return ResourceFilter{}, err
		}
	}
	if filter.Scope != nil {
		scope := filter.Scope.Normalize()
		if err := scope.Validate("filter.scope"); err != nil {
			return ResourceFilter{}, err
		}
		normalized.Scope = &scope
	}
	if filter.Owner != nil {
		owner := filter.Owner.Normalize()
		if err := owner.Validate("filter.owner"); err != nil {
			return ResourceFilter{}, err
		}
		normalized.Owner = &owner
	}
	if filter.Source != nil {
		source := filter.Source.Normalize()
		if err := source.Validate("filter.source"); err != nil {
			return ResourceFilter{}, err
		}
		normalized.Source = &source
	}
	if normalized.Limit < 0 {
		return ResourceFilter{}, fmt.Errorf("%w: filter.limit cannot be negative: %d", ErrValidation, normalized.Limit)
	}
	return normalized, nil
}

func (k *Kernel) preparePutRaw(actor MutationActor, draft RawDraft) (MutationActor, RawDraft, error) {
	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return MutationActor{}, RawDraft{}, err
	}
	if normalizedActor.Kind == MutationActorKindExtension {
		return MutationActor{}, RawDraft{}, fmt.Errorf(
			"%w: extension actors cannot use direct raw mutations",
			ErrDirectMutationNotAllowed,
		)
	}
	if err := normalizedActor.Source.Validate("actor.source"); err != nil {
		return MutationActor{}, RawDraft{}, err
	}

	normalizedDraft, err := normalizeDraft(draft, k.maxSpecBytes)
	if err != nil {
		return MutationActor{}, RawDraft{}, err
	}
	if err := validateActorWriteAccess(normalizedActor, normalizedDraft.Kind, normalizedDraft.Scope); err != nil {
		return MutationActor{}, RawDraft{}, err
	}
	return normalizedActor, normalizedDraft, nil
}

func (k *Kernel) putRawWithExecutor(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	draft RawDraft,
) (RawRecord, error) {
	existing, found, err := lookupRecordWithExecutor(ctx, exec, draft.Kind, draft.ID)
	if err != nil {
		return RawRecord{}, err
	}
	if !found {
		if draft.ExpectedVersion != 0 {
			return RawRecord{}, ErrNotFound
		}
		return k.insertRawRecord(ctx, exec, actor, draft)
	}

	if err := validateActorWriteAccess(actor, existing.Kind, existing.Scope); err != nil {
		return RawRecord{}, err
	}
	if existing.Source != actor.Source {
		return RawRecord{}, fmt.Errorf(
			"%w: actor cannot mutate source %q/%q",
			ErrPermissionDenied,
			existing.Source.Kind,
			existing.Source.ID,
		)
	}
	if draft.ExpectedVersion == 0 || existing.Version != draft.ExpectedVersion {
		return RawRecord{}, fmt.Errorf("%w: expected version %d", ErrConflict, draft.ExpectedVersion)
	}

	now := k.now()
	record := RawRecord{
		Kind:      draft.Kind,
		ID:        draft.ID,
		Version:   existing.Version + 1,
		Scope:     draft.Scope,
		Owner:     ownerFromActor(actor),
		Source:    actor.Source,
		SpecJSON:  append([]byte(nil), draft.SpecJSON...),
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}
	if err := updateRecord(ctx, exec, record, draft.ExpectedVersion); err != nil {
		return RawRecord{}, err
	}
	return record, nil
}

func (k *Kernel) insertRawRecord(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	draft RawDraft,
) (RawRecord, error) {
	now := k.now()
	record := RawRecord{
		Kind:      draft.Kind,
		ID:        draft.ID,
		Version:   1,
		Scope:     draft.Scope,
		Owner:     ownerFromActor(actor),
		Source:    actor.Source,
		SpecJSON:  append([]byte(nil), draft.SpecJSON...),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := insertRecord(ctx, exec, record); err != nil {
		return RawRecord{}, err
	}
	return record, nil
}

func (k *Kernel) prepareDeleteRaw(
	actor MutationActor,
	kind ResourceKind,
	id string,
) (MutationActor, ResourceKind, string, error) {
	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return MutationActor{}, "", "", err
	}
	if normalizedActor.Kind == MutationActorKindExtension {
		return MutationActor{}, "", "", fmt.Errorf(
			"%w: extension actors cannot use direct raw mutations",
			ErrDirectMutationNotAllowed,
		)
	}
	if err := normalizedActor.Source.Validate("actor.source"); err != nil {
		return MutationActor{}, "", "", err
	}

	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("kind"); err != nil {
		return MutationActor{}, "", "", err
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return MutationActor{}, "", "", fmt.Errorf("%w: id is required", ErrValidation)
	}
	return normalizedActor, normalizedKind, trimmedID, nil
}

func (k *Kernel) deleteRawWithExecutor(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	kind ResourceKind,
	id string,
	expectedVersion int64,
) error {
	if expectedVersion < 0 {
		return fmt.Errorf("%w: expected_version cannot be negative: %d", ErrValidation, expectedVersion)
	}

	existing, found, err := lookupRecordWithExecutor(ctx, exec, kind, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	if err := validateActorWriteAccess(actor, existing.Kind, existing.Scope); err != nil {
		return err
	}
	if existing.Source != actor.Source {
		return fmt.Errorf(
			"%w: actor cannot mutate source %q/%q",
			ErrPermissionDenied,
			existing.Source.Kind,
			existing.Source.ID,
		)
	}

	result, err := exec.ExecContext(ctx, deleteRecordByVersionQuery, kind, id, expectedVersion)
	if err != nil {
		return fmt.Errorf("resources: delete record %q/%q: %w", kind, id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("resources: rows affected for delete %q/%q: %w", kind, id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: expected version %d", ErrConflict, expectedVersion)
	}
	return nil
}

func (k *Kernel) prepareListRaw(
	actor MutationActor,
	filter ResourceFilter,
) (MutationActor, ResourceFilter, error) {
	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return MutationActor{}, ResourceFilter{}, err
	}
	normalizedFilter, err := normalizeFilter(filter)
	if err != nil {
		return MutationActor{}, ResourceFilter{}, err
	}

	if normalizedFilter.Kind != "" && !actorAllowsKind(normalizedActor, normalizedFilter.Kind) {
		return MutationActor{}, ResourceFilter{}, fmt.Errorf(
			"%w: actor cannot read resource kind %q",
			ErrPermissionDenied,
			normalizedFilter.Kind,
		)
	}
	if normalizedFilter.Scope != nil {
		if !actorAllowsScopeKind(normalizedActor, normalizedFilter.Scope.Kind) {
			return MutationActor{}, ResourceFilter{}, fmt.Errorf(
				"%w: actor cannot read scope kind %q",
				ErrPermissionDenied,
				normalizedFilter.Scope.Kind,
			)
		}
		if !actorAllowsScope(normalizedActor, *normalizedFilter.Scope) {
			return MutationActor{}, ResourceFilter{}, fmt.Errorf(
				"%w: actor max scope does not allow %q/%q",
				ErrPermissionDenied,
				normalizedFilter.Scope.Kind,
				normalizedFilter.Scope.ID,
			)
		}
	}
	if normalizedActor.Kind == MutationActorKindExtension &&
		normalizedFilter.Source != nil &&
		*normalizedFilter.Source != normalizedActor.Source {
		return MutationActor{}, ResourceFilter{}, fmt.Errorf(
			"%w: actor cannot read source %q/%q",
			ErrPermissionDenied,
			normalizedFilter.Source.Kind,
			normalizedFilter.Source.ID,
		)
	}
	return normalizedActor, normalizedFilter, nil
}

func (k *Kernel) extensionReadGrantsEmpty(actor MutationActor) bool {
	return actor.Kind == MutationActorKindExtension &&
		(len(actor.GrantedKinds) == 0 || len(actor.GrantedScopes) == 0)
}

func buildListRawQuery(actor MutationActor, filter ResourceFilter) (string, []any) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 10)

	if actor.Kind == MutationActorKindExtension {
		clauses = append(clauses, "source_kind = ?", "source_id = ?")
		args = append(args, actor.Source.Kind, actor.Source.ID)
	}
	if filter.Kind != "" {
		clauses = append(clauses, "kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.Scope != nil {
		clauses = append(clauses, "scope_kind = ?")
		args = append(args, filter.Scope.Kind)
		if filter.Scope.Kind == ResourceScopeKindGlobal {
			clauses = append(clauses, "scope_id IS NULL")
		} else {
			clauses = append(clauses, "scope_id = ?")
			args = append(args, filter.Scope.ID)
		}
	}
	if filter.Owner != nil {
		clauses = append(clauses, "owner_kind = ?", "owner_id = ?")
		args = append(args, filter.Owner.Kind, filter.Owner.ID)
	}
	if filter.Source != nil && actor.Kind != MutationActorKindExtension {
		clauses = append(clauses, "source_kind = ?", "source_id = ?")
		args = append(args, filter.Source.Kind, filter.Source.ID)
	}

	var builder strings.Builder
	builder.WriteString(rawRecordSelectQuery)
	if len(clauses) > 0 {
		builder.WriteString("\nWHERE ")
		builder.WriteString(strings.Join(clauses, "\n\tAND "))
	}
	builder.WriteString(resourceRecordOrderBy)
	return builder.String(), args
}

func (k *Kernel) prepareSnapshotApply(
	actor MutationActor,
	snapshot SourceSnapshot,
) (MutationActor, SourceSnapshot, []RawDraft, error) {
	normalizedActor, err := normalizeActor(actor)
	if err != nil {
		return MutationActor{}, SourceSnapshot{}, nil, err
	}
	if normalizedActor.Kind != MutationActorKindExtension {
		return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
			"%w: only extension actors may apply source snapshots",
			ErrPermissionDenied,
		)
	}
	if normalizedActor.SessionNonce == "" {
		return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
			"%w: actor.session_nonce is required",
			ErrValidation,
		)
	}
	if k.extensionReadGrantsEmpty(normalizedActor) {
		return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
			"%w: extension actor requires granted kinds and scopes",
			ErrPermissionDenied,
		)
	}

	normalizedSnapshot, err := normalizeSnapshot(snapshot, k.maxSnapshotRecords)
	if err != nil {
		return MutationActor{}, SourceSnapshot{}, nil, err
	}

	normalizedDrafts := make([]RawDraft, 0, len(snapshot.Records))
	seenKeys := make(map[string]struct{}, len(snapshot.Records))
	totalBytes := 0
	for _, draft := range snapshot.Records {
		normalizedDraft, draftErr := normalizeDraft(draft, k.maxSpecBytes)
		if draftErr != nil {
			return MutationActor{}, SourceSnapshot{}, nil, draftErr
		}
		if normalizedDraft.ExpectedVersion != 0 {
			return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
				"%w: snapshot records must use expected_version 0",
				ErrValidation,
			)
		}
		if accessErr := validateActorWriteAccess(
			normalizedActor,
			normalizedDraft.Kind,
			normalizedDraft.Scope,
		); accessErr != nil {
			return MutationActor{}, SourceSnapshot{}, nil, accessErr
		}

		key := resourceKey(normalizedDraft.Kind, normalizedDraft.ID)
		if _, exists := seenKeys[key]; exists {
			return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
				"%w: duplicate snapshot record %q/%q",
				ErrValidation,
				normalizedDraft.Kind,
				normalizedDraft.ID,
			)
		}
		seenKeys[key] = struct{}{}

		totalBytes += len(normalizedDraft.SpecJSON)
		if totalBytes > k.maxSnapshotBytes {
			return MutationActor{}, SourceSnapshot{}, nil, fmt.Errorf(
				"%w: snapshot payload exceeds %d bytes",
				ErrPayloadTooLarge,
				k.maxSnapshotBytes,
			)
		}
		normalizedDrafts = append(normalizedDrafts, normalizedDraft)
	}

	return normalizedActor, normalizedSnapshot, normalizedDrafts, nil
}

func (k *Kernel) applySnapshotWithExecutor(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	snapshot SourceSnapshot,
	drafts []RawDraft,
) error {
	state, err := k.requireActiveSourceState(ctx, exec, actor)
	if err != nil {
		return err
	}

	existingRecords, err := listSourceRecordsWithExecutor(ctx, exec, actor.Source)
	if err != nil {
		return err
	}
	existingByKey := make(map[string]RawRecord, len(existingRecords))
	for _, record := range existingRecords {
		existingByKey[resourceKey(record.Kind, record.ID)] = record
	}

	if err := k.applySnapshotDrafts(ctx, exec, actor, drafts, existingByKey); err != nil {
		return err
	}
	if err := deleteRemovedSourceRecords(ctx, exec, actor.Source, existingRecords, drafts); err != nil {
		return err
	}
	return advanceSourceState(ctx, exec, actor, state, snapshot.SourceVersion, k.now())
}

func (k *Kernel) requireActiveSourceState(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
) (sourceState, error) {
	state, found, err := lookupSourceState(ctx, exec, actor.Source)
	if err != nil {
		return sourceState{}, err
	}
	if !found || state.SessionNonce != actor.SessionNonce {
		return sourceState{}, fmt.Errorf(
			"%w: source %q/%q nonce %q",
			ErrSessionNotActive,
			actor.Source.Kind,
			actor.Source.ID,
			actor.SessionNonce,
		)
	}
	return state, nil
}

func (k *Kernel) applySnapshotDrafts(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	drafts []RawDraft,
	existingByKey map[string]RawRecord,
) error {
	for _, draft := range drafts {
		if err := k.applySnapshotDraft(ctx, exec, actor, draft, existingByKey); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kernel) applySnapshotDraft(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	draft RawDraft,
	existingByKey map[string]RawRecord,
) error {
	key := resourceKey(draft.Kind, draft.ID)
	existing, found := existingByKey[key]
	if !found {
		record, lookupFound, err := lookupRecordWithExecutor(ctx, exec, draft.Kind, draft.ID)
		if err != nil {
			return err
		}
		if lookupFound {
			return fmt.Errorf(
				"%w: snapshot cannot overwrite source %q/%q",
				ErrConflict,
				record.Source.Kind,
				record.Source.ID,
			)
		}
		_, err = k.insertRawRecord(ctx, exec, actor, draft)
		return err
	}

	nextRecord := RawRecord{
		Kind:      draft.Kind,
		ID:        draft.ID,
		Version:   existing.Version + 1,
		Scope:     draft.Scope,
		Owner:     ownerFromActor(actor),
		Source:    actor.Source,
		SpecJSON:  append([]byte(nil), draft.SpecJSON...),
		CreatedAt: existing.CreatedAt,
		UpdatedAt: k.now(),
	}
	if recordsEqual(existing, nextRecord) {
		return nil
	}
	return updateRecord(ctx, exec, nextRecord, existing.Version)
}

func deleteRemovedSourceRecords(
	ctx context.Context,
	exec sqlExecutor,
	source ResourceSource,
	existingRecords []RawRecord,
	drafts []RawDraft,
) error {
	desiredKeys := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		desiredKeys[resourceKey(draft.Kind, draft.ID)] = struct{}{}
	}
	for _, record := range existingRecords {
		if _, keep := desiredKeys[resourceKey(record.Kind, record.ID)]; keep {
			continue
		}
		if _, err := exec.ExecContext(
			ctx,
			deleteStaleSourceRecordQuery,
			record.Kind,
			record.ID,
			source.Kind,
			source.ID,
		); err != nil {
			return fmt.Errorf(
				"resources: delete stale source record %q/%q for %q/%q: %w",
				record.Kind,
				record.ID,
				source.Kind,
				source.ID,
				err,
			)
		}
	}
	return nil
}

func advanceSourceState(
	ctx context.Context,
	exec sqlExecutor,
	actor MutationActor,
	state sourceState,
	sourceVersion int64,
	now time.Time,
) error {
	if sourceVersion <= state.LastSnapshotVersion {
		return fmt.Errorf(
			"%w: expected > %d, got %d",
			ErrStaleSourceVersion,
			state.LastSnapshotVersion,
			sourceVersion,
		)
	}

	result, err := exec.ExecContext(
		ctx,
		updateSourceStateQuery,
		sourceVersion,
		store.FormatTimestamp(now),
		actor.Source.Kind,
		actor.Source.ID,
		actor.SessionNonce,
	)
	if err != nil {
		return fmt.Errorf(
			"resources: update source state %q/%q: %w",
			actor.Source.Kind,
			actor.Source.ID,
			err,
		)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"resources: rows affected for source state %q/%q: %w",
			actor.Source.Kind,
			actor.Source.ID,
			err,
		)
	}
	if rowsAffected == 0 {
		return fmt.Errorf(
			"%w: source %q/%q nonce %q",
			ErrSessionNotActive,
			actor.Source.Kind,
			actor.Source.ID,
			actor.SessionNonce,
		)
	}
	return nil
}

func insertRecord(ctx context.Context, exec sqlExecutor, record RawRecord) error {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO resource_records (
			kind, id, version, scope_kind, scope_id, owner_kind,
			owner_id, source_kind, source_id, spec_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Kind,
		record.ID,
		record.Version,
		record.Scope.Kind,
		nullableScopeID(record.Scope),
		record.Owner.Kind,
		record.Owner.ID,
		record.Source.Kind,
		record.Source.ID,
		string(record.SpecJSON),
		store.FormatTimestamp(record.CreatedAt),
		store.FormatTimestamp(record.UpdatedAt),
	); err != nil {
		return fmt.Errorf("resources: insert record %q/%q: %w", record.Kind, record.ID, err)
	}
	return nil
}

func updateRecord(ctx context.Context, exec sqlExecutor, record RawRecord, expectedVersion int64) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE resource_records
		 SET version = ?, scope_kind = ?, scope_id = ?, owner_kind = ?, owner_id = ?,
		     source_kind = ?, source_id = ?, spec_json = ?, updated_at = ?
		 WHERE kind = ? AND id = ? AND version = ?`,
		record.Version,
		record.Scope.Kind,
		nullableScopeID(record.Scope),
		record.Owner.Kind,
		record.Owner.ID,
		record.Source.Kind,
		record.Source.ID,
		string(record.SpecJSON),
		store.FormatTimestamp(record.UpdatedAt),
		record.Kind,
		record.ID,
		expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("resources: update record %q/%q: %w", record.Kind, record.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("resources: rows affected for update %q/%q: %w", record.Kind, record.ID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: expected version %d", ErrConflict, expectedVersion)
	}
	return nil
}

func lookupRecordWithExecutor(
	ctx context.Context,
	exec sqlExecutor,
	kind ResourceKind,
	id string,
) (RawRecord, bool, error) {
	row := exec.QueryRowContext(ctx, selectRecordByKeyQuery, kind, id)
	record, err := scanRawRecord(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return RawRecord{}, false, nil
	case err != nil:
		return RawRecord{}, false, err
	default:
		return record, true, nil
	}
}

func listSourceRecordsWithExecutor(ctx context.Context, exec sqlExecutor, source ResourceSource) ([]RawRecord, error) {
	rows, err := exec.QueryContext(
		ctx,
		selectSourceRecordsQuery,
		source.Kind,
		source.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("resources: query source records %q/%q: %w", source.Kind, source.ID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	records := make([]RawRecord, 0)
	for rows.Next() {
		record, scanErr := scanRawRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resources: iterate source records %q/%q: %w", source.Kind, source.ID, err)
	}
	return records, nil
}

func lookupSourceState(ctx context.Context, exec sqlExecutor, source ResourceSource) (sourceState, bool, error) {
	var (
		sourceKind          string
		sourceID            string
		sessionNonce        string
		lastSnapshotVersion int64
		updatedAtRaw        string
	)
	if err := exec.QueryRowContext(
		ctx,
		selectSourceStateQuery,
		source.Kind,
		source.ID,
	).Scan(&sourceKind, &sourceID, &sessionNonce, &lastSnapshotVersion, &updatedAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sourceState{}, false, nil
		}
		return sourceState{}, false, fmt.Errorf("resources: query source state %q/%q: %w", source.Kind, source.ID, err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return sourceState{}, false, fmt.Errorf(
			"resources: parse source state timestamp %q/%q: %w",
			source.Kind,
			source.ID,
			err,
		)
	}
	return sourceState{
		Source: ResourceSource{
			Kind: ResourceSourceKind(sourceKind),
			ID:   sourceID,
		},
		SessionNonce:        sessionNonce,
		LastSnapshotVersion: lastSnapshotVersion,
		UpdatedAt:           updatedAt,
	}, true, nil
}

type rawRecordScanner interface {
	Scan(dest ...any) error
}

func scanRawRecord(scanner rawRecordScanner) (RawRecord, error) {
	var (
		record       RawRecord
		scopeKind    string
		scopeID      sql.NullString
		ownerKind    string
		ownerID      string
		sourceKind   string
		sourceID     string
		specJSON     string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&record.Kind,
		&record.ID,
		&record.Version,
		&scopeKind,
		&scopeID,
		&ownerKind,
		&ownerID,
		&sourceKind,
		&sourceID,
		&specJSON,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return RawRecord{}, err
	}

	record.Scope = ResourceScope{
		Kind: ResourceScopeKind(scopeKind),
		ID:   strings.TrimSpace(scopeID.String),
	}
	record.Owner = ResourceOwner{
		Kind: ResourceOwnerKind(ownerKind),
		ID:   ownerID,
	}
	record.Source = ResourceSource{
		Kind: ResourceSourceKind(sourceKind),
		ID:   sourceID,
	}
	record.SpecJSON = []byte(specJSON)

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return RawRecord{}, fmt.Errorf("resources: parse created_at for %q/%q: %w", record.Kind, record.ID, err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return RawRecord{}, fmt.Errorf("resources: parse updated_at for %q/%q: %w", record.Kind, record.ID, err)
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return record, nil
}

func rollbackTx(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}

func rollbackImmediate(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return nil
	}
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		return err
	}
	return nil
}

func joinCleanupError(target *error, cleanupErr error) {
	if cleanupErr == nil || target == nil {
		return
	}
	if *target == nil {
		*target = cleanupErr
		return
	}
	*target = errors.Join(*target, cleanupErr)
}

func (k *Kernel) withImmediateTransaction(
	ctx context.Context,
	action string,
	run func(conn *sql.Conn) error,
) (err error) {
	conn, err := k.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("resources: open connection for %s: %w", action, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("resources: begin immediate %s transaction: %w", action, err)
	}

	finished := false
	defer func() {
		if !finished {
			if rollbackErr := rollbackImmediate(rollbackCtx, conn); rollbackErr != nil && err == nil {
				err = fmt.Errorf("resources: rollback %s transaction: %w", action, rollbackErr)
			}
		}
	}()

	if err := run(conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("resources: commit %s transaction: %w", action, err)
	}

	finished = true
	return nil
}

func (k *Kernel) lockSource(source ResourceSource) func() {
	key := string(source.Kind) + "\x00" + source.ID

	k.sourceLocksMu.Lock()
	lock, ok := k.sourceLocks[key]
	if !ok {
		lock = &sourceLock{}
		k.sourceLocks[key] = lock
	}
	lock.refs++
	k.sourceLocksMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()

		k.sourceLocksMu.Lock()
		defer k.sourceLocksMu.Unlock()

		lock.refs--
		if lock.refs == 0 {
			delete(k.sourceLocks, key)
		}
	}
}

func resourceKey(kind ResourceKind, id string) string {
	return string(kind) + "\x00" + id
}

func recordsEqual(left RawRecord, right RawRecord) bool {
	return left.Kind == right.Kind &&
		left.ID == right.ID &&
		left.Scope == right.Scope &&
		left.Owner == right.Owner &&
		left.Source == right.Source &&
		bytes.Equal(left.SpecJSON, right.SpecJSON)
}

func nullableScopeID(scope ResourceScope) any {
	if scope.Kind == ResourceScopeKindGlobal {
		return nil
	}
	return scope.ID
}
