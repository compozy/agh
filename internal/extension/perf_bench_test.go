package extensionpkg

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

func BenchmarkDecodeHostAPIParamsTaskCreate(b *testing.B) {
	b.ReportAllocs()

	raw := json.RawMessage(fmt.Sprintf(`{
		"id":"task-bench",
		"identifier":"bench-task",
		"scope":"workspace",
		"workspace":"ws-bench",
		"network_channel":"agent/bench",
		"title":"Benchmark task",
		"description":"Benchmark payload decode",
		"metadata":{"body":"%s","labels":["alpha","beta","gamma"]}
	}`, extensionBenchmarkText(512)))

	for b.Loop() {
		var params hostAPITaskCreateParams
		if err := decodeHostAPIParams(raw, &params); err != nil {
			b.Fatalf("decodeHostAPIParams() error = %v", err)
		}
		if params.Title != "Benchmark task" {
			b.Fatalf("params.Title = %q, want %q", params.Title, "Benchmark task")
		}
	}
}

func BenchmarkTaskSummaryPayloadsFromSummaries(b *testing.B) {
	b.ReportAllocs()

	summaries := extensionBenchmarkTaskSummaries(512)

	for b.Loop() {
		payloads := taskSummaryPayloadsFromSummaries(summaries)
		if len(payloads) != len(summaries) {
			b.Fatalf("len(payloads) = %d, want %d", len(payloads), len(summaries))
		}
		if payloads[len(payloads)-1].ID == "" {
			b.Fatal("last payload id is empty")
		}
	}
}

func BenchmarkTaskRunPayloadsFromRuns(b *testing.B) {
	b.ReportAllocs()

	runs := extensionBenchmarkTaskRuns(256)

	for b.Loop() {
		payloads := taskRunPayloadsFromRuns(runs)
		if len(payloads) != len(runs) {
			b.Fatalf("len(payloads) = %d, want %d", len(payloads), len(runs))
		}
		if len(payloads[len(payloads)-1].Result) == 0 {
			b.Fatal("last payload result is empty")
		}
	}
}

func extensionBenchmarkTaskSummaries(count int) []taskpkg.Summary {
	summaries := make([]taskpkg.Summary, 0, count)
	now := time.Unix(1_700_000_000, 0).UTC()
	for i := range count {
		summaries = append(summaries, taskpkg.Summary{
			ID:             fmt.Sprintf("task-%03d", i),
			Identifier:     fmt.Sprintf("bench-%03d", i),
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-bench",
			ParentTaskID:   fmt.Sprintf("parent-%03d", i%8),
			NetworkChannel: "agent/bench",
			Title:          fmt.Sprintf("Benchmark task %03d", i),
			Status:         taskpkg.TaskStatusReady,
			Owner: &taskpkg.Ownership{
				Kind: taskpkg.OwnerKindExtension,
				Ref:  fmt.Sprintf("owner-%03d", i%4),
			},
			CreatedBy: taskpkg.ActorIdentity{
				Kind: taskpkg.ActorKindExtension,
				Ref:  "bench-ext",
			},
			Origin: taskpkg.Origin{
				Kind: taskpkg.OriginKindExtension,
				Ref:  "bench-ext",
			},
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedAt: now.Add(time.Duration(i+1) * time.Second),
			ClosedAt:  time.Time{},
		})
	}
	return summaries
}

func extensionBenchmarkTaskRuns(count int) []taskpkg.Run {
	runs := make([]taskpkg.Run, 0, count)
	now := time.Unix(1_700_000_000, 0).UTC()
	result := json.RawMessage(
		fmt.Sprintf(`{"summary":%q,"scores":[1,2,3,4],"ok":true}`, extensionBenchmarkText(1024)),
	)
	for i := range count {
		runs = append(runs, taskpkg.Run{
			ID:             fmt.Sprintf("run-%03d", i),
			TaskID:         fmt.Sprintf("task-%03d", i),
			Status:         taskpkg.TaskRunStatusRunning,
			Attempt:        i%3 + 1,
			ClaimedBy:      extensionBenchmarkClaimedBy(i),
			SessionID:      fmt.Sprintf("session-%03d", i),
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindExtension, Ref: "bench-ext"},
			IdempotencyKey: fmt.Sprintf("idem-%03d", i),
			NetworkChannel: "agent/bench",
			QueuedAt:       now.Add(time.Duration(i) * time.Second),
			ClaimedAt:      now.Add(time.Duration(i+1) * time.Second),
			StartedAt:      now.Add(time.Duration(i+2) * time.Second),
			EndedAt:        time.Time{},
			Error:          "",
			Result:         append(json.RawMessage(nil), result...),
		})
	}
	return runs
}

func extensionBenchmarkClaimedBy(i int) *taskpkg.ActorIdentity {
	if i%5 == 0 {
		return nil
	}
	return &taskpkg.ActorIdentity{
		Kind: taskpkg.ActorKindExtension,
		Ref:  fmt.Sprintf("bench-ext-%d", i%7),
	}
}

func extensionBenchmarkText(size int) string {
	if size <= 0 {
		return ""
	}
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte('a' + (i % 26))
	}
	return string(buf)
}
