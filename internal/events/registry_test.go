package events

import (
	"strings"
	"testing"
)

func TestRegistryMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should expose one complete metadata row per event", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]struct{})
		for _, meta := range All() {
			if strings.TrimSpace(meta.Name) == "" {
				t.Fatal("registry entry has empty name")
			}
			if _, ok := seen[meta.Name]; ok {
				t.Fatalf("registry entry %q is duplicated", meta.Name)
			}
			seen[meta.Name] = struct{}{}
			if strings.TrimSpace(meta.Family) == "" {
				t.Fatalf("registry entry %q has empty family", meta.Name)
			}
			if strings.TrimSpace(meta.Component) == "" {
				t.Fatalf("registry entry %q has empty component", meta.Name)
			}
			if !ValidOutcome(string(meta.Outcome)) || meta.Outcome == "" {
				t.Fatalf("registry entry %q has invalid outcome %q", meta.Name, meta.Outcome)
			}
			if !meta.EmitsToLogs {
				t.Fatalf("registry entry %q must declare log eligibility", meta.Name)
			}
			lookup, ok := Lookup(meta.Name)
			if !ok {
				t.Fatalf("Lookup(%q) failed", meta.Name)
			}
			if lookup != meta {
				t.Fatalf("Lookup(%q) = %#v, want %#v", meta.Name, lookup, meta)
			}
		}
	})

	t.Run("Should preserve task run underscore events and reject task_run dot family", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{
			TaskRunEnqueued,
			TaskRunClaimed,
			TaskRunStarted,
			TaskRunCompleted,
			TaskRunFailed,
			TaskRunCanceled,
		} {
			if _, ok := Lookup(name); !ok {
				t.Fatalf("Lookup(%q) = false, want true", name)
			}
		}
		if _, ok := Lookup("task_run.completed"); ok {
			t.Fatal("Lookup(task_run.completed) = true, want false")
		}
		if err := ValidatePublicName("task_run.completed"); err == nil {
			t.Fatal("ValidatePublicName(task_run.completed) error = nil, want non-nil")
		}
	})

	t.Run("Should expose metadata consumed by logs and notifications", func(t *testing.T) {
		t.Parallel()

		failed, ok := Lookup(TaskRunFailed)
		if !ok {
			t.Fatal("Lookup(TaskRunFailed) = false")
		}
		if failed.Outcome != OutcomeFailure || failed.Component != ComponentTask || !failed.NotificationEligible {
			t.Fatalf("TaskRunFailed metadata = %#v", failed)
		}
		shadowed, ok := Lookup(SkillShadowed)
		if !ok {
			t.Fatal("Lookup(SkillShadowed) = false")
		}
		if shadowed.Component != ComponentSkill || shadowed.Outcome != OutcomeWarning || !shadowed.GlobalScope {
			t.Fatalf("SkillShadowed metadata = %#v", shadowed)
		}
		if _, ok := Lookup("skills.shadow"); ok {
			t.Fatal("Lookup(skills.shadow) = true, want false after hard cut")
		}
	})

	t.Run("Should keep memory operation projections out of direct global writes", func(t *testing.T) {
		t.Parallel()

		if AllowsGlobalScope(MemoryWriteCommitted) {
			t.Fatal("AllowsGlobalScope(MemoryWriteCommitted) = true, want false for memory_events projection")
		}
		if !AllowsGlobalScope(MemoryProviderCollision) {
			t.Fatal("AllowsGlobalScope(MemoryProviderCollision) = false, want true for extension collision summaries")
		}
	})
}
