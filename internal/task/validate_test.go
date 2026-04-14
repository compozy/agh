package task

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateScopeBinding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scope       Scope
		workspaceID string
		wantErr     error
	}{
		{name: "global without workspace", scope: ScopeGlobal},
		{name: "workspace with workspace", scope: ScopeWorkspace, workspaceID: "ws-1"},
		{name: "global with workspace", scope: ScopeGlobal, workspaceID: "ws-1", wantErr: ErrInvalidScopeBinding},
		{name: "workspace without workspace", scope: ScopeWorkspace, wantErr: ErrInvalidScopeBinding},
		{name: "unsupported scope", scope: Scope("tenant"), wantErr: ErrValidation},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateScopeBinding(tt.scope, tt.workspaceID, "task", "workspace_id")
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateScopeBinding() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateScopeBinding() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateScopeBinding() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImmutableTaskFields(t *testing.T) {
	t.Parallel()

	current := validTask()
	tests := []struct {
		name        string
		mutate      func(*Task)
		wantField   string
		expectError bool
	}{
		{
			name: "created by immutable",
			mutate: func(next *Task) {
				next.CreatedBy.Ref = "human-2"
			},
			wantField:   TaskFieldCreatedBy,
			expectError: true,
		},
		{
			name: "origin immutable",
			mutate: func(next *Task) {
				next.Origin.Ref = "http:api"
			},
			wantField:   TaskFieldOrigin,
			expectError: true,
		},
		{
			name: "scope immutable",
			mutate: func(next *Task) {
				next.Scope = ScopeWorkspace
				next.WorkspaceID = "ws-1"
			},
			wantField:   TaskFieldScope,
			expectError: true,
		},
		{
			name: "workspace id immutable",
			mutate: func(next *Task) {
				next.WorkspaceID = "ws-2"
			},
			wantField:   TaskFieldWorkspaceID,
			expectError: true,
		},
		{
			name: "parent task id immutable",
			mutate: func(next *Task) {
				next.ParentTaskID = "task-2"
			},
			wantField:   TaskFieldParentTaskID,
			expectError: true,
		},
		{
			name: "mutable fields allowed",
			mutate: func(next *Task) {
				next.Title = "Updated"
				next.Description = "changed"
				next.NetworkChannel = "network:alpha"
				next.Owner = &Ownership{Kind: OwnerKindPool, Ref: "triage"}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := current
			tt.mutate(&next)

			err := ValidateImmutableTaskFields(current, next)
			if !tt.expectError {
				if err != nil {
					t.Fatalf("ValidateImmutableTaskFields() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateImmutableTaskFields() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrImmutableField) {
				t.Fatalf("ValidateImmutableTaskFields() error = %v, want ErrImmutableField", err)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("ValidateImmutableTaskFields() error = %q, want field %q", err.Error(), tt.wantField)
			}
		})
	}
}

func TestPayloadSizeGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{
			name: "metadata within limit",
			run: func() error {
				return ValidateMetadataSize(jsonBlob(MaxMetadataBytes-8), "task.metadata")
			},
		},
		{
			name: "metadata over limit",
			run: func() error {
				return ValidateMetadataSize(jsonBlob(MaxMetadataBytes+1), "task.metadata")
			},
			wantErr: ErrPayloadTooLarge,
		},
		{
			name: "payload over limit",
			run: func() error {
				return ValidatePayloadSize(jsonBlob(MaxPayloadBytes+1), "task_event.payload")
			},
			wantErr: ErrPayloadTooLarge,
		},
		{
			name: "result over limit",
			run: func() error {
				return ValidateResultSize(jsonBlob(MaxResultBytes+1), "task_run.result")
			},
			wantErr: ErrPayloadTooLarge,
		},
		{
			name: "invalid json",
			run: func() error {
				return ValidatePayloadSize(json.RawMessage(`{`), "task_event.payload")
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.run()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("payload guard error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("payload guard error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("payload guard error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraphLimitGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{
			name: "depth at limit",
			run: func() error {
				return ValidateHierarchyDepth(MaxHierarchyDepth)
			},
		},
		{
			name: "depth over limit",
			run: func() error {
				return ValidateHierarchyDepth(MaxHierarchyDepth + 1)
			},
			wantErr: ErrGraphLimitExceeded,
		},
		{
			name: "dependency count over limit",
			run: func() error {
				return ValidateDependencyCount(MaxDependencyCount + 1)
			},
			wantErr: ErrGraphLimitExceeded,
		},
		{
			name: "direct child count over limit",
			run: func() error {
				return ValidateDirectChildCount(MaxDirectChildren + 1)
			},
			wantErr: ErrGraphLimitExceeded,
		},
		{
			name: "negative count rejected",
			run: func() error {
				return ValidateDependencyCount(-1)
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.run()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("graph limit error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("graph limit error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("graph limit error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskFieldMutabilityHelpers(t *testing.T) {
	t.Parallel()

	for _, field := range ImmutableTaskFields() {
		if !IsImmutableTaskField(field) {
			t.Fatalf("IsImmutableTaskField(%q) = false, want true", field)
		}
	}
	for _, field := range MutableTaskFields() {
		if !IsMutableTaskField(field) {
			t.Fatalf("IsMutableTaskField(%q) = false, want true", field)
		}
	}
	if IsImmutableTaskField("title") {
		t.Fatal("IsImmutableTaskField(\"title\") = true, want false")
	}
	if IsMutableTaskField("scope") {
		t.Fatal("IsMutableTaskField(\"scope\") = true, want false")
	}
}

func TestDomainValidationHelpers(t *testing.T) {
	t.Parallel()

	t.Run("task valid", func(t *testing.T) {
		t.Parallel()
		if err := validTask().Validate(); err != nil {
			t.Fatalf("Task.Validate() error = %v", err)
		}
	})

	t.Run("task invalid owner", func(t *testing.T) {
		t.Parallel()
		taskRecord := validTask()
		taskRecord.Owner = &Ownership{Kind: OwnerKindHuman}
		err := taskRecord.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("Task.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task dependency self dependency", func(t *testing.T) {
		t.Parallel()
		err := (TaskDependency{
			TaskID:          "task-1",
			DependsOnTaskID: "task-1",
			Kind:            DependencyKindBlocks,
		}).Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskDependency.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task run queued session invalid", func(t *testing.T) {
		t.Parallel()
		run := validRun()
		run.SessionID = "sess-1"
		err := run.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskRun.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task event invalid payload", func(t *testing.T) {
		t.Parallel()
		event := validEvent()
		event.Payload = json.RawMessage(`{`)
		err := event.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskEvent.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task patch requires mutable field", func(t *testing.T) {
		t.Parallel()
		err := (TaskPatch{}).Validate("patch")
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskPatch.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("run failure requires error", func(t *testing.T) {
		t.Parallel()
		err := (RunFailure{}).Validate("failure")
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("RunFailure.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("start task session requires matching task and run", func(t *testing.T) {
		t.Parallel()
		req := StartTaskSession{
			Task:  validTask(),
			Run:   validRun(),
			Actor: validActorContext(),
		}
		req.Run.TaskID = "task-2"
		err := req.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("StartTaskSession.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task query validates filters", func(t *testing.T) {
		t.Parallel()
		err := (TaskQuery{
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
			Status:      TaskStatusReady,
			OwnerKind:   OwnerKindPool,
			Limit:       10,
		}).Validate("query")
		if err != nil {
			t.Fatalf("TaskQuery.Validate() error = %v", err)
		}
	})

	t.Run("start task session valid", func(t *testing.T) {
		t.Parallel()
		req := StartTaskSession{
			Task:  validTask(),
			Run:   validRun(),
			Actor: validActorContext(),
		}
		if err := req.Validate(); err != nil {
			t.Fatalf("StartTaskSession.Validate() error = %v", err)
		}
	})
}

func validTask() Task {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	return Task{
		ID:           "task-1",
		Identifier:   "TASK-1",
		Scope:        ScopeGlobal,
		Title:        "Bootstrap internal/task",
		Description:  "Create the task domain",
		Status:       TaskStatusReady,
		Owner:        &Ownership{Kind: OwnerKindHuman, Ref: "user-1"},
		CreatedBy:    ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
		Origin:       Origin{Kind: OriginKindCLI, Ref: "agh task create"},
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     json.RawMessage(`{"priority":"high"}`),
		ClosedAt:     time.Time{},
		ParentTaskID: "",
	}
}

func validRun() TaskRun {
	now := time.Date(2026, 4, 14, 12, 30, 0, 0, time.UTC)
	return TaskRun{
		ID:       "run-1",
		TaskID:   "task-1",
		Status:   TaskRunStatusQueued,
		Attempt:  1,
		Origin:   Origin{Kind: OriginKindCLI, Ref: "agh task run enqueue"},
		QueuedAt: now,
		Result:   json.RawMessage(`{"ok":true}`),
	}
}

func validEvent() TaskEvent {
	now := time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC)
	return TaskEvent{
		ID:        "evt-1",
		TaskID:    "task-1",
		EventType: "task.created",
		Actor:     ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
		Origin:    Origin{Kind: OriginKindCLI, Ref: "agh task create"},
		Payload:   json.RawMessage(`{"source":"cli"}`),
		Timestamp: now,
	}
}

func validTaskRunIdempotency() TaskRunIdempotency {
	now := time.Date(2026, 4, 14, 13, 30, 0, 0, time.UTC)
	return TaskRunIdempotency{
		IdempotencyKey: "idem-1",
		RunID:          "run-1",
		Origin:         Origin{Kind: OriginKindAutomation, Ref: "rule:nightly"},
		CreatedAt:      now,
	}
}

func validActorContext() ActorContext {
	return ActorContext{
		Actor:  ActorIdentity{Kind: ActorKindHuman, Ref: "user-1"},
		Origin: Origin{Kind: OriginKindCLI, Ref: "agh task run start"},
		Authority: Authority{
			Read:            true,
			Write:           true,
			CreateGlobal:    true,
			CreateWorkspace: true,
		},
	}
}

func jsonBlob(targetSize int) json.RawMessage {
	if targetSize <= 2 {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(`"` + strings.Repeat("a", targetSize-2) + `"`)
}

func TestEnumAndIdentityValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{name: "task status valid", run: func() error { return TaskStatusReady.Validate("status") }},
		{name: "task status invalid", run: func() error { return TaskStatus("waiting").Validate("status") }, wantErr: ErrValidation},
		{name: "task run status valid", run: func() error { return TaskRunStatusRunning.Validate("run.status") }},
		{name: "task run status invalid", run: func() error { return TaskRunStatus("paused").Validate("run.status") }, wantErr: ErrValidation},
		{name: "actor kind valid", run: func() error { return ActorKindHuman.Validate("actor.kind") }},
		{name: "actor kind invalid", run: func() error { return ActorKind("bot").Validate("actor.kind") }, wantErr: ErrValidation},
		{name: "owner kind valid", run: func() error { return OwnerKindPool.Validate("owner.kind") }},
		{name: "owner kind invalid", run: func() error { return OwnerKind("queue").Validate("owner.kind") }, wantErr: ErrValidation},
		{name: "origin kind valid", run: func() error { return OriginKindCLI.Validate("origin.kind") }},
		{name: "origin kind invalid", run: func() error { return OriginKind("mqtt").Validate("origin.kind") }, wantErr: ErrValidation},
		{name: "dependency kind valid", run: func() error { return DependencyKindBlocks.Validate("dependency.kind") }},
		{name: "dependency kind invalid", run: func() error { return DependencyKind("soft").Validate("dependency.kind") }, wantErr: ErrValidation},
		{name: "stop reason valid", run: func() error { return StopReasonCancellation.Validate("stop.reason") }},
		{name: "stop reason invalid", run: func() error { return StopReason("later").Validate("stop.reason") }, wantErr: ErrValidation},
		{name: "actor identity valid", run: func() error { return validTask().CreatedBy.Validate("actor") }},
		{name: "actor identity invalid", run: func() error { return ActorIdentity{Kind: ActorKindHuman}.Validate("actor") }, wantErr: ErrValidation},
		{name: "origin valid", run: func() error { return validTask().Origin.Validate("origin") }},
		{name: "origin invalid", run: func() error { return Origin{Kind: OriginKindCLI}.Validate("origin") }, wantErr: ErrValidation},
		{name: "authority valid", run: func() error { return validActorContext().Authority.Validate("authority") }},
		{name: "authority invalid", run: func() error {
			return Authority{CreateGlobal: true}.Validate("authority")
		}, wantErr: ErrValidation},
		{name: "actor context valid", run: func() error { return validActorContext().Validate() }},
		{name: "actor context invalid", run: func() error {
			ctx := validActorContext()
			ctx.Actor.Ref = ""
			return ctx.Validate()
		}, wantErr: ErrValidation},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.run()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validation error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validation error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validation error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestAndQueryValidation(t *testing.T) {
	t.Parallel()

	title := "Updated title"
	channel := "network:alpha"
	metadata := json.RawMessage(`{"priority":"medium"}`)

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{
			name: "create task valid",
			run: func() error {
				return CreateTask{
					Scope:       ScopeWorkspace,
					Title:       "Create task",
					Owner:       &Ownership{Kind: OwnerKindPool, Ref: "triage"},
					Metadata:    json.RawMessage(`{"kind":"bootstrap"}`),
					WorkspaceID: "ws-1",
				}.Validate("create")
			},
		},
		{
			name: "create task invalid parent self",
			run: func() error {
				return CreateTask{
					ID:           "task-1",
					Scope:        ScopeGlobal,
					Title:        "Create task",
					ParentTaskID: "task-1",
				}.Validate("create")
			},
			wantErr: ErrValidation,
		},
		{
			name: "task patch valid",
			run: func() error {
				return TaskPatch{
					Title:          &title,
					NetworkChannel: &channel,
					Metadata:       &metadata,
				}.Validate("patch")
			},
		},
		{
			name: "task patch owner conflict",
			run: func() error {
				return TaskPatch{
					Owner:      &Ownership{Kind: OwnerKindPool, Ref: "triage"},
					ClearOwner: true,
				}.Validate("patch")
			},
			wantErr: ErrValidation,
		},
		{
			name: "cancel task metadata valid",
			run: func() error {
				return CancelTask{Metadata: json.RawMessage(`{"reason":"user"}`)}.Validate("cancel")
			},
		},
		{
			name: "add dependency valid",
			run: func() error {
				return AddDependency{
					TaskID:          "task-1",
					DependsOnTaskID: "task-0",
					Kind:            DependencyKindBlocks,
				}.Validate("dependency")
			},
		},
		{
			name: "add dependency invalid",
			run: func() error {
				return AddDependency{
					TaskID:          "task-1",
					DependsOnTaskID: "task-1",
					Kind:            DependencyKindBlocks,
				}.Validate("dependency")
			},
			wantErr: ErrValidation,
		},
		{
			name: "enqueue run valid",
			run: func() error {
				return EnqueueRun{TaskID: "task-1"}.Validate("enqueue")
			},
		},
		{
			name: "enqueue run invalid",
			run: func() error {
				return EnqueueRun{}.Validate("enqueue")
			},
			wantErr: ErrValidation,
		},
		{
			name: "claim run valid",
			run: func() error {
				return ClaimRun{}.Validate("claim")
			},
		},
		{
			name: "claim run invalid path",
			run: func() error {
				return ClaimRun{}.Validate(" ")
			},
			wantErr: ErrValidation,
		},
		{
			name: "start run valid",
			run: func() error {
				return StartRun{}.Validate("start")
			},
		},
		{
			name: "start run invalid path",
			run: func() error {
				return StartRun{}.Validate("")
			},
			wantErr: ErrValidation,
		},
		{
			name: "cancel run metadata valid",
			run: func() error {
				return CancelRun{Metadata: json.RawMessage(`{"reason":"user"}`)}.Validate("cancel")
			},
		},
		{
			name: "run result valid",
			run: func() error {
				return RunResult{Value: json.RawMessage(`{"ok":true}`)}.Validate("result")
			},
		},
		{
			name: "task run query valid",
			run: func() error {
				return TaskRunQuery{Status: TaskRunStatusRunning, Limit: 2}.Validate("runs")
			},
		},
		{
			name: "task run query invalid",
			run: func() error {
				return TaskRunQuery{Limit: -1}.Validate("runs")
			},
			wantErr: ErrValidation,
		},
		{
			name: "task event query valid",
			run: func() error {
				return TaskEventQuery{Limit: 1}.Validate("events")
			},
		},
		{
			name: "task event query invalid",
			run: func() error {
				return TaskEventQuery{Limit: -1}.Validate("events")
			},
			wantErr: ErrValidation,
		},
		{
			name: "task run idempotency valid",
			run: func() error {
				return validTaskRunIdempotency().Validate()
			},
		},
		{
			name: "task run idempotency invalid",
			run: func() error {
				record := validTaskRunIdempotency()
				record.Origin.Ref = ""
				return record.Validate()
			},
			wantErr: ErrValidation,
		},
		{
			name: "session ref valid",
			run: func() error {
				return SessionRef{SessionID: "sess-1"}.Validate()
			},
		},
		{
			name: "session ref invalid",
			run: func() error {
				return SessionRef{}.Validate()
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.run()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validation error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validation error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validation error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOwnershipIsZero(t *testing.T) {
	t.Parallel()

	if !(Ownership{}).IsZero() {
		t.Fatal("Ownership{}.IsZero() = false, want true")
	}
	if (Ownership{Kind: OwnerKindPool, Ref: "triage"}).IsZero() {
		t.Fatal("Ownership{pool}.IsZero() = true, want false")
	}
}

func TestAdditionalBranchCoverage(t *testing.T) {
	t.Parallel()

	t.Run("task missing id", func(t *testing.T) {
		t.Parallel()
		taskRecord := validTask()
		taskRecord.ID = ""
		err := taskRecord.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("Task.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task parent self", func(t *testing.T) {
		t.Parallel()
		taskRecord := validTask()
		taskRecord.ParentTaskID = taskRecord.ID
		err := taskRecord.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("Task.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task missing title", func(t *testing.T) {
		t.Parallel()
		taskRecord := validTask()
		taskRecord.Title = ""
		err := taskRecord.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("Task.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task run missing claimed by ref", func(t *testing.T) {
		t.Parallel()
		run := validRun()
		run.Status = TaskRunStatusClaimed
		run.ClaimedBy = &ActorIdentity{Kind: ActorKindHuman}
		err := run.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskRun.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task run invalid attempt", func(t *testing.T) {
		t.Parallel()
		run := validRun()
		run.Attempt = 0
		err := run.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskRun.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task event missing event type", func(t *testing.T) {
		t.Parallel()
		event := validEvent()
		event.EventType = ""
		err := event.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskEvent.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("task event missing origin", func(t *testing.T) {
		t.Parallel()
		event := validEvent()
		event.Origin.Ref = ""
		err := event.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("TaskEvent.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("start task session invalid actor", func(t *testing.T) {
		t.Parallel()
		req := StartTaskSession{
			Task:  validTask(),
			Run:   validRun(),
			Actor: validActorContext(),
		}
		req.Actor.Authority = Authority{CreateGlobal: true}
		err := req.Validate()
		if err == nil || !errors.Is(err, ErrValidation) {
			t.Fatalf("StartTaskSession.Validate() error = %v, want ErrValidation", err)
		}
	})

	t.Run("nested path helper empty path", func(t *testing.T) {
		t.Parallel()
		if got := nestedPath("", "field"); got != "field" {
			t.Fatalf("nestedPath('', 'field') = %q, want field", got)
		}
		if got := nestedPath("root", ""); got != "root" {
			t.Fatalf("nestedPath('root', '') = %q, want root", got)
		}
	})
}
