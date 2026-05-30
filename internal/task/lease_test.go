package task

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestClaimCriteriaValidationAndTokenHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	base := ClaimCriteria{
		Scope:                ScopeGlobal,
		ClaimerSessionID:     "sess-claim",
		RequiredCapabilities: []string{"golang"},
		LeaseDuration:        time.Minute,
		Now:                  now,
	}

	tests := []struct {
		name     string
		criteria ClaimCriteria
		wantErr  error
	}{
		{
			name:     "missing claimer session",
			criteria: ClaimCriteria{Scope: ScopeGlobal, LeaseDuration: time.Minute, Now: now},
			wantErr:  ErrValidation,
		},
		{
			name: "workspace scope requires workspace id",
			criteria: ClaimCriteria{
				Scope:            ScopeWorkspace,
				ClaimerSessionID: "sess-claim",
				LeaseDuration:    time.Minute,
				Now:              now,
			},
			wantErr: ErrInvalidScopeBinding,
		},
		{
			name: "capability ids reject whitespace",
			criteria: ClaimCriteria{
				Scope:                ScopeGlobal,
				ClaimerSessionID:     "sess-claim",
				RequiredCapabilities: []string{"golang sqlite"},
				LeaseDuration:        time.Minute,
				Now:                  now,
			},
			wantErr: ErrValidation,
		},
		{
			name: "lease duration is bounded",
			criteria: ClaimCriteria{
				Scope:            ScopeGlobal,
				ClaimerSessionID: "sess-claim",
				LeaseDuration:    MaxRunLeaseDuration + time.Nanosecond,
				Now:              now,
			},
			wantErr: ErrValidation,
		},
		{
			name:     "valid criteria defaults claimed by",
			criteria: base,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.criteria.Normalize(now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Normalize() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if got.ClaimedBy == nil || got.ClaimedBy.Kind != ActorKindAgentSession ||
				got.ClaimedBy.Ref != "sess-claim" {
				t.Fatalf("ClaimedBy = %#v, want agent session claimer", got.ClaimedBy)
			}
		})
	}

	rawToken := "agh_claim_unit_token"
	hash, err := ClaimTokenHash(rawToken)
	if err != nil {
		t.Fatalf("ClaimTokenHash() error = %v", err)
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Fatalf("ClaimTokenHash() = %q, want sha256 prefix", hash)
	}
	if strings.Contains(hash, rawToken) {
		t.Fatalf("ClaimTokenHash() = %q contains raw token", hash)
	}
	if !VerifyClaimToken(rawToken, hash) {
		t.Fatal("VerifyClaimToken(raw, hash) = false, want true")
	}
	if !VerifyClaimToken(" "+rawToken+" ", strings.TrimPrefix(hash, "sha256:")) {
		t.Fatal("VerifyClaimToken() should accept canonical hash without prefix and trim raw token")
	}
	if VerifyClaimToken("wrong-token", hash) {
		t.Fatal("VerifyClaimToken(wrong, hash) = true, want false")
	}
	if _, err := ClaimTokenHash(" "); !errors.Is(err, ErrValidation) {
		t.Fatalf("ClaimTokenHash(empty) error = %v, want %v", err, ErrValidation)
	}

	redacted := RedactClaimTokens("bad token agh_claim_secret-123 and agh_claim_other_456")
	if strings.Contains(redacted, "agh_claim_secret-123") ||
		strings.Contains(redacted, "agh_claim_other_456") ||
		strings.Count(redacted, "agh_claim_[REDACTED]") != 2 {
		t.Fatalf("RedactClaimTokens() = %q, want both raw claim tokens redacted", redacted)
	}
}

func TestClaimResultSanitizesRawClaimTokenMetadata(t *testing.T) {
	t.Parallel()

	result := ClaimResult{
		Task: Task{
			Metadata: json.RawMessage(`{"claim_token":"task-raw","nested":{"claim_token":"nested-raw","keep":true}}`),
		},
		Run: Run{
			Metadata: json.RawMessage(`{"claim_token":"run-raw","items":[{"claim_token":"item-raw","ok":true}]}`),
			Result:   json.RawMessage(`{"claim_token":"result-raw","ok":true}`),
		},
		CoordinationChannel: &CoordinationChannelMetadata{
			ID:                  " coord.core ",
			AllowedMessageKinds: []string{"status", "status", " reply "},
		},
	}

	claimResultWithoutRawTokenInMetadata(&result)
	for label, raw := range map[string]json.RawMessage{
		"task metadata": result.Task.Metadata,
		"run metadata":  result.Run.Metadata,
		"run result":    result.Run.Result,
	} {
		if strings.Contains(strings.ToLower(string(raw)), "claim_token") {
			t.Fatalf("%s still contains raw claim_token field: %s", label, raw)
		}
	}
	if result.CoordinationChannel == nil {
		t.Fatal("CoordinationChannel = nil, want sanitized metadata")
	}
	if got, want := result.CoordinationChannel.ID, "coord.core"; got != want {
		t.Fatalf("CoordinationChannel.ID = %q, want %q", got, want)
	}
	if got, want := result.CoordinationChannel.DisplayName, "coord.core"; got != want {
		t.Fatalf("CoordinationChannel.DisplayName = %q, want %q", got, want)
	}
	if got, want := result.CoordinationChannel.AllowedMessageKinds, []string{
		"status",
		"reply",
	}; len(
		got,
	) != len(
		want,
	) ||
		got[0] != want[0] ||
		got[1] != want[1] {
		t.Fatalf("AllowedMessageKinds = %#v, want %#v", got, want)
	}
}

func TestManagerLookupActiveRunForSessionRejectsUnsafeLeases(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		sessionID string
		runID     string
		seed      func(t *testing.T, store *inMemoryManagerStore)
		want      AutonomyReasonCode
		cause     error
		rawToken  string
	}{
		{
			name:      "Should reject missing session identity",
			sessionID: " ",
			runID:     "run-1",
			want:      AutonomySessionRequired,
			cause:     ErrPermissionDenied,
		},
		{
			name:      "Should reject missing active lease",
			sessionID: "sess-a",
			runID:     "run-1",
			want:      AutonomyNoActiveLease,
			cause:     ErrInvalidClaimToken,
		},
		{
			name:      "Should reject foreign run while session holds another lease",
			sessionID: "sess-a",
			runID:     "run-2",
			rawToken:  "agh_claim_FOREIGN123",
			seed: func(t *testing.T, store *inMemoryManagerStore) {
				t.Helper()
				seedAutonomyLeaseRun(t, store, "run-1", "sess-a", "agh_claim_FOREIGN123", now.Add(time.Minute))
			},
			want:  AutonomyForeignRun,
			cause: ErrPermissionDenied,
		},
		{
			name:      "Should reject stale lease",
			sessionID: "sess-a",
			runID:     "run-1",
			rawToken:  "agh_claim_EXPIRED123",
			seed: func(t *testing.T, store *inMemoryManagerStore) {
				t.Helper()
				seedAutonomyLeaseRun(t, store, "run-1", "sess-a", "agh_claim_EXPIRED123", now.Add(-time.Minute))
			},
			want:  AutonomyLeaseExpired,
			cause: ErrLeaseExpired,
		},
		{
			name:      "Should reject double active lease",
			sessionID: "sess-a",
			runID:     "run-1",
			rawToken:  "agh_claim_DOUBLE_A123",
			seed: func(t *testing.T, store *inMemoryManagerStore) {
				t.Helper()
				seedAutonomyLeaseRun(t, store, "run-1", "sess-a", "agh_claim_DOUBLE_A123", now.Add(time.Minute))
				seedAutonomyLeaseRun(t, store, "run-2", "sess-a", "agh_claim_DOUBLE_B123", now.Add(time.Minute))
			},
			want:  AutonomyLeaseAlreadyHeld,
			cause: ErrActiveRunLease,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newInMemoryManagerStore()
			if tt.seed != nil {
				tt.seed(t, store)
			}
			manager := newTaskManagerForTestWithOptions(t, store, WithManagerNow(func() time.Time {
				return now
			}))
			_, err := manager.LookupActiveRunForSession(context.Background(), tt.sessionID, tt.runID)
			if !errors.Is(err, tt.cause) {
				t.Fatalf("LookupActiveRunForSession() error = %v, want cause %v", err, tt.cause)
			}
			reason, ok := AutonomyReasonOf(err)
			if !ok || reason != tt.want {
				t.Fatalf("AutonomyReasonOf() = %q/%v, want %q", reason, ok, tt.want)
			}
			if tt.rawToken != "" && strings.Contains(err.Error(), tt.rawToken) {
				t.Fatalf("LookupActiveRunForSession() leaked raw token in error: %v", err)
			}
		})
	}
}

func TestManagerLookupActiveRunForSessionReturnsInternalHandle(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	rawToken := "agh_claim_ACTIVE123"
	store := newInMemoryManagerStore()
	seedAutonomyLeaseRun(t, store, "run-1", "sess-a", rawToken, now.Add(time.Minute))
	manager := newTaskManagerForTestWithOptions(t, store, WithManagerNow(func() time.Time {
		return now
	}))

	handle, err := manager.LookupActiveRunForSession(context.Background(), " sess-a ", " run-1 ")
	if err != nil {
		t.Fatalf("LookupActiveRunForSession() error = %v", err)
	}
	if handle.RunID != "run-1" ||
		handle.SessionID != "sess-a" ||
		handle.ClaimToken != rawToken ||
		!VerifyClaimToken(rawToken, handle.ClaimTokenHash) {
		t.Fatalf("LookupActiveRunForSession() = %#v, want active internal lease handle", handle)
	}
}

func seedAutonomyLeaseRun(
	t *testing.T,
	store *inMemoryManagerStore,
	runID string,
	sessionID string,
	rawToken string,
	leaseUntil time.Time,
) {
	t.Helper()

	tokenHash, err := ClaimTokenHash(rawToken)
	if err != nil {
		t.Fatalf("ClaimTokenHash() error = %v", err)
	}
	normalizedRunID := strings.TrimSpace(runID)
	normalizedSessionID := strings.TrimSpace(sessionID)
	store.runs[normalizedRunID] = Run{
		ID:             normalizedRunID,
		TaskID:         "task-" + strings.TrimPrefix(normalizedRunID, "run-"),
		Status:         TaskRunStatusClaimed,
		SessionID:      normalizedSessionID,
		ClaimedBy:      &ActorIdentity{Kind: ActorKindAgentSession, Ref: normalizedSessionID},
		ClaimToken:     strings.TrimSpace(rawToken),
		ClaimTokenHash: tokenHash,
		ClaimedAt:      leaseUntil.Add(-time.Minute),
		HeartbeatAt:    leaseUntil.Add(-time.Minute),
		LeaseUntil:     leaseUntil,
		QueuedAt:       leaseUntil.Add(-2 * time.Minute),
	}
}

func TestManagerClaimNextRunAndLeaseFencing(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	operator := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-agent"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "codex"}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Claim lease task",
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	firstRun, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, operator)
	if err != nil {
		t.Fatalf("EnqueueRun(first) error = %v", err)
	}
	secondTask, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Second claim lease task",
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask(second) error = %v", err)
	}
	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: secondTask.ID}, operator); err != nil {
		t.Fatalf("EnqueueRun(second) error = %v", err)
	}

	claimNow := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-agent",
		LeaseDuration:    time.Minute,
		Now:              claimNow,
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if got, want := claim.Run.ID, firstRun.ID; got != want {
		t.Fatalf("ClaimNextRun() run id = %q, want %q", got, want)
	}
	if claim.ClaimToken == "" {
		t.Fatal("ClaimToken is empty")
	}
	if claim.Run.ClaimToken != "" {
		t.Fatalf("Run.ClaimToken = %q, want empty read model", claim.Run.ClaimToken)
	}
	if !VerifyClaimToken(claim.ClaimToken, claim.Run.ClaimTokenHash) {
		t.Fatal("ClaimToken does not verify against persisted hash")
	}

	if _, err := manager.CompleteRun(context.Background(), firstRun.ID, RunResult{
		Value: json.RawMessage(`{"legacy":true}`),
	}, agent); !errors.Is(err, ErrInvalidClaimToken) {
		t.Fatalf("CompleteRun(unfenced active lease) error = %v, want %v", err, ErrInvalidClaimToken)
	}
	if _, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         firstRun.ID,
		ClaimToken:    "wrong-token",
		LeaseDuration: time.Minute,
		Now:           claimNow.Add(10 * time.Second),
	}, agent); !errors.Is(err, ErrInvalidClaimToken) {
		t.Fatalf("HeartbeatRunLease(stale token) error = %v, want %v", err, ErrInvalidClaimToken)
	}
	if _, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         firstRun.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: time.Minute,
		Now:           claim.LeaseUntil,
	}, agent); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("HeartbeatRunLease(expired lease) error = %v, want %v", err, ErrLeaseExpired)
	}
	heartbeat, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         firstRun.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: 2 * time.Minute,
		Now:           claimNow.Add(30 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("HeartbeatRunLease(current token) error = %v", err)
	}
	if got, want := heartbeat.LeaseUntil, claimNow.Add(150*time.Second); !got.Equal(want) {
		t.Fatalf("HeartbeatRunLease().LeaseUntil = %v, want %v", got, want)
	}

	if _, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-agent",
		LeaseDuration:    time.Minute,
		Now:              claimNow.Add(40 * time.Second),
	}, agent); !errors.Is(err, ErrActiveRunLease) {
		t.Fatalf("ClaimNextRun(second active lease) error = %v, want %v", err, ErrActiveRunLease)
	}

	completed, err := manager.CompleteRunLease(context.Background(), LeaseCompletion{
		RunID:      firstRun.ID,
		ClaimToken: claim.ClaimToken,
		Result:     RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        claimNow.Add(45 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("CompleteRunLease() error = %v", err)
	}
	if got, want := completed.Status, TaskRunStatusCompleted; got != want {
		t.Fatalf("completed.Status = %q, want %q", got, want)
	}
	if completed.LeaseUntil.IsZero() == false || completed.HeartbeatAt.IsZero() == false {
		t.Fatalf(
			"completed lease fields = lease_until %v heartbeat_at %v, want zero",
			completed.LeaseUntil,
			completed.HeartbeatAt,
		)
	}
	if completed.ClaimTokenHash == "" {
		t.Fatal("completed.ClaimTokenHash = empty, want retained fencing history")
	}

	secondClaim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-agent",
		LeaseDuration:    time.Minute,
		Now:              claimNow.Add(time.Minute),
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun(after completion) error = %v", err)
	}
	released, err := manager.ReleaseRunLease(context.Background(), LeaseRelease{
		RunID:      secondClaim.Run.ID,
		ClaimToken: secondClaim.ClaimToken,
		Reason:     "handoff",
		Now:        claimNow.Add(70 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("ReleaseRunLease() error = %v", err)
	}
	if got, want := released.Status, TaskRunStatusQueued; got != want {
		t.Fatalf("released.Status = %q, want %q", got, want)
	}
	if released.ClaimTokenHash != "" || released.SessionID != "" || released.ClaimedBy != nil {
		t.Fatalf("released ownership fields = hash %q session %q claimed_by %#v, want cleared",
			released.ClaimTokenHash,
			released.SessionID,
			released.ClaimedBy,
		)
	}
	if _, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         released.ID,
		ClaimToken:    secondClaim.ClaimToken,
		LeaseDuration: time.Minute,
		Now:           claimNow.Add(80 * time.Second),
	}, agent); !errors.Is(err, ErrInvalidClaimToken) {
		t.Fatalf("HeartbeatRunLease(after release) error = %v, want %v", err, ErrInvalidClaimToken)
	}

	failClaim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-agent",
		LeaseDuration:    time.Minute,
		Now:              claimNow.Add(90 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun(for failure) error = %v", err)
	}
	failed, err := manager.FailRunLease(context.Background(), LeaseFailure{
		RunID:      failClaim.Run.ID,
		ClaimToken: failClaim.ClaimToken,
		Failure:    RunFailure{Error: "worker failed"},
		Now:        claimNow.Add(100 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("FailRunLease() error = %v", err)
	}
	if got, want := failed.Status, TaskRunStatusFailed; got != want {
		t.Fatalf("failed.Status = %q, want %q", got, want)
	}
	if got, want := failed.Error, "worker failed"; got != want {
		t.Fatalf("failed.Error = %q, want %q", got, want)
	}
}

func TestManagerClaimNextRunRequiresWriteAuthority(t *testing.T) {
	t.Parallel()

	manager := newTaskManagerForTest(t, newInMemoryManagerStore())
	actor := validActorContext()
	actor.Authority.Write = false
	actor.Authority.CreateGlobal = false
	actor.Authority.CreateWorkspace = false

	if _, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-agent",
	}, actor); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("ClaimNextRun(read-only actor) error = %v, want %v", err, ErrPermissionDenied)
	}
}

func TestManagerBlockRunLeaseParksRunNeedsAttention(t *testing.T) {
	t.Parallel()

	manager := newTaskManagerForTest(t, newInMemoryManagerStore())
	operator := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-block"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "worker"}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Human blocked task",
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, operator); err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	now := time.Now().UTC()
	claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-block",
		LeaseDuration:    time.Minute,
		Now:              now,
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}

	blocked, err := manager.BlockRunLease(context.Background(), LeaseBlock{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Reason:     "blocked_on_human: Figma OAuth required",
		Now:        now.Add(10 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("BlockRunLease() error = %v", err)
	}
	if got, want := blocked.Status, TaskRunStatusNeedsAttention; got != want {
		t.Fatalf("blocked.Status = %q, want %q", got, want)
	}
	if blocked.ClaimTokenHash != "" ||
		blocked.SessionID != "" ||
		blocked.ClaimedBy != nil ||
		!blocked.LeaseUntil.IsZero() {
		t.Fatalf(
			"blocked ownership fields = hash %q session %q claimed_by %#v lease %v, want cleared",
			blocked.ClaimTokenHash,
			blocked.SessionID,
			blocked.ClaimedBy,
			blocked.LeaseUntil,
		)
	}
	view, err := manager.GetTask(context.Background(), taskRecord.ID, operator)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got, want := view.Task.Status, TaskStatusBlocked; got != want {
		t.Fatalf("task status after block = %q, want %q", got, want)
	}
	if _, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-other",
		LeaseDuration:    time.Minute,
		Now:              now.Add(20 * time.Second),
	}, agent); !errors.Is(err, ErrNoClaimableRun) {
		t.Fatalf("ClaimNextRun(after block) error = %v, want %v", err, ErrNoClaimableRun)
	}
}

func TestManagerCompleteRunLeaseRequiresCompletionContractArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := newTaskManagerForTestWithOptions(
		t,
		newInMemoryManagerStore(),
		WithCompletionContractRootResolver(func(context.Context, Task, Run) (string, error) {
			return root, nil
		}),
	)
	operator := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-contract"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "worker"}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Receipt gated task",
		Metadata: json.RawMessage(
			`{"completion_contract":{"required_artifacts":[{"path":"receipts/phase.yaml"}]}}`,
		),
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, operator); err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	now := time.Now().UTC()
	claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-contract",
		LeaseDuration:    time.Minute,
		Now:              now,
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	_, err = manager.CompleteRunLease(context.Background(), LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        now.Add(10 * time.Second),
	}, agent)
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("CompleteRunLease(missing receipt) error = %v, want %v", err, ErrValidation)
	}

	if err := os.MkdirAll(root+"/receipts", 0o755); err != nil {
		t.Fatalf("MkdirAll(receipts) error = %v", err)
	}
	if err := os.WriteFile(root+"/receipts/phase.yaml", []byte("status: completed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(receipt) error = %v", err)
	}
	completed, err := manager.CompleteRunLease(context.Background(), LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        now.Add(20 * time.Second),
	}, agent)
	if err != nil {
		t.Fatalf("CompleteRunLease(with receipt) error = %v", err)
	}
	if got, want := completed.Status, TaskRunStatusCompleted; got != want {
		t.Fatalf("completed.Status = %q, want %q", got, want)
	}
}

func TestManagerCompleteRunLeaseRejectsAbsoluteCompletionContractArtifacts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	absoluteReceipt := root + "/receipt.yaml"
	if err := os.WriteFile(absoluteReceipt, []byte("status: completed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(receipt) error = %v", err)
	}
	manager := newTaskManagerForTestWithOptions(
		t,
		newInMemoryManagerStore(),
		WithCompletionContractRootResolver(func(context.Context, Task, Run) (string, error) {
			return root, nil
		}),
	)
	operator := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-contract"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "worker"}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Absolute receipt task",
		Metadata: json.RawMessage(
			`{"completion_contract":{"required_artifacts":[{"path":` + strconv.Quote(absoluteReceipt) + `}]}}`,
		),
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, operator); err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	now := time.Now().UTC()
	claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-contract",
		LeaseDuration:    time.Minute,
		Now:              now,
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	_, err = manager.CompleteRunLease(context.Background(), LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     RunResult{Value: json.RawMessage(`{"ok":true}`)},
		Now:        now.Add(10 * time.Second),
	}, agent)
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("CompleteRunLease(absolute receipt) error = %v, want %v", err, ErrValidation)
	}
}

func TestManagerReleaseSessionRunLeasesRequeuesActiveRunsStructurally(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	operator := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-child"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "coder"}

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Structurally released task",
	}, operator)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, operator)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-child",
		LeaseDuration:    time.Minute,
		Now:              now,
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun() error = %v", err)
	}
	if claim.Run.ID != run.ID || claim.Run.ClaimTokenHash == "" {
		t.Fatalf("claim = %#v, want active lease for %q", claim, run.ID)
	}

	results, err := manager.ReleaseSessionRunLeases(context.Background(), SessionLeaseRelease{
		SessionID: "sess-child",
		Reason:    "ttl_expired",
		Now:       now.Add(30 * time.Second),
	}, operator)
	if err != nil {
		t.Fatalf("ReleaseSessionRunLeases() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	result := results[0]
	if result.PreviousRunStatus != TaskRunStatusClaimed ||
		result.PreviousSessionID != "sess-child" ||
		result.PreviousClaimTokenHash == "" ||
		result.Reason != "ttl_expired" {
		t.Fatalf("release result = %#v, want previous active lease metadata", result)
	}
	persisted, err := store.GetTaskRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if persisted.Status != TaskRunStatusQueued ||
		persisted.SessionID != "" ||
		persisted.ClaimedBy != nil ||
		persisted.ClaimTokenHash != "" ||
		!persisted.LeaseUntil.IsZero() ||
		!persisted.HeartbeatAt.IsZero() {
		t.Fatalf("persisted run after structural release = %#v, want queued and unleased", persisted)
	}
	if _, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         run.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: time.Minute,
		Now:           now.Add(40 * time.Second),
	}, agent); !errors.Is(err, ErrInvalidClaimToken) {
		t.Fatalf("HeartbeatRunLease(after structural release) error = %v, want %v", err, ErrInvalidClaimToken)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{
		TaskID:    taskRecord.ID,
		RunID:     run.ID,
		EventType: taskEventRunReleased,
	})
	if err != nil {
		t.Fatalf("ListTaskEvents(released) error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("release events = %#v, want one task.run_released event", events)
	}
}
