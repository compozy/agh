package globaldb

import (
	"fmt"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
	taskpkg "github.com/compozy/agh/internal/task"
)

func executionProfileFromGenerated(row sqlcgen.GetTaskExecutionProfileRow) (taskpkg.ExecutionProfile, error) {
	createdAt, err := store.ParseTimestamp(row.CreatedAt)
	if err != nil {
		return taskpkg.ExecutionProfile{}, fmt.Errorf("store: parse task execution profile created_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(row.UpdatedAt)
	if err != nil {
		return taskpkg.ExecutionProfile{}, fmt.Errorf("store: parse task execution profile updated_at: %w", err)
	}
	return taskpkg.ExecutionProfile{
		TaskID: row.TaskID,
		Coordinator: taskpkg.CoordinatorProfile{
			Mode: taskpkg.CoordinatorMode(row.CoordinatorMode), AgentName: row.CoordinatorAgentName,
			Provider: row.CoordinatorProvider, Model: row.CoordinatorModel, Guidance: row.CoordinatorGuidance,
		},
		Worker: taskpkg.WorkerProfile{
			Mode: taskpkg.WorkerMode(row.WorkerMode), AgentName: row.WorkerAgentName,
			Provider: row.WorkerProvider, Model: row.WorkerModel,
		},
		Review: taskpkg.ReviewProfile{
			AgentName: row.ReviewAgentName, Provider: row.ReviewProvider, Model: row.ReviewModel,
		},
		Sandbox: taskpkg.SandboxPolicy{
			Mode: taskpkg.SandboxMode(row.SandboxMode), SandboxRef: row.SandboxRef,
		},
		Runtime:   taskpkg.RuntimePolicy{Mode: taskpkg.RuntimeMode(row.RuntimeMode)},
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}
