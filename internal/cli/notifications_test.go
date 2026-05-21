package cli

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNotificationPresetCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should list presets through daemon client filters", func(t *testing.T) {
		t.Parallel()

		var captured NotificationPresetQuery
		deps := newTestDeps(t, &stubClient{
			listNotificationPresetsFn: func(
				_ context.Context,
				query NotificationPresetQuery,
			) (NotificationPresetListRecord, error) {
				captured = query
				return NotificationPresetListRecord{
					Presets: []NotificationPresetRecord{notificationPresetRecordForTest("task_terminal")},
					Total:   1,
				}, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"notifications", "presets", "list", "--enabled", "--built-in", "--name", "task_terminal", "-o", "json",
		)
		if err != nil {
			t.Fatalf("notifications presets list error = %v", err)
		}
		if captured.Enabled == nil || !*captured.Enabled || captured.BuiltIn == nil || !*captured.BuiltIn ||
			captured.Name != "task_terminal" {
			t.Fatalf("captured query = %#v, want enabled built-in task_terminal", captured)
		}
		var payload NotificationPresetListRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(list stdout) error = %v\nstdout=%s", err, stdout)
		}
		if payload.Total != 1 || payload.Presets[0].Name != "task_terminal" {
			t.Fatalf("payload = %#v, want task_terminal list", payload)
		}
	})

	t.Run("Should create preset with event and target payload", func(t *testing.T) {
		t.Parallel()

		var captured CreateNotificationPresetRequest
		deps := newTestDeps(t, &stubClient{
			createNotificationPresetFn: func(
				_ context.Context,
				request CreateNotificationPresetRequest,
			) (NotificationPresetRecord, error) {
				captured = request
				record := notificationPresetRecordForTest(request.Name)
				record.Events = request.Events
				record.Targets = request.Targets
				record.Filter = request.Filter
				record.Enabled = request.Enabled
				return record, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"notifications", "preset", "create", "provider_failure_copy",
			"--event", "provider.*",
			"--target", "brg-1:#ops",
			"--filter", "severity >= warning",
			"--enabled",
			"-o", "json",
		)
		if err != nil {
			t.Fatalf("notifications preset create error = %v", err)
		}
		if captured.Name != "provider_failure_copy" || len(captured.Events) != 1 ||
			len(captured.Targets) != 1 || captured.Targets[0].BridgeID != "brg-1" ||
			captured.Targets[0].CanonicalRoute != "#ops" || captured.Filter != "severity >= warning" ||
			!captured.Enabled {
			t.Fatalf("captured request = %#v", captured)
		}
		var payload NotificationPresetRecord
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("json.Unmarshal(create stdout) error = %v\nstdout=%s", err, stdout)
		}
		if payload.Name != "provider_failure_copy" || !payload.Enabled {
			t.Fatalf("payload = %#v, want enabled provider_failure_copy", payload)
		}
	})

	t.Run("Should enable preset with replacement targets", func(t *testing.T) {
		t.Parallel()

		var capturedName string
		var captured UpdateNotificationPresetRequest
		deps := newTestDeps(t, &stubClient{
			updateNotificationPresetFn: func(
				_ context.Context,
				name string,
				request UpdateNotificationPresetRequest,
			) (NotificationPresetRecord, error) {
				capturedName = name
				captured = request
				record := notificationPresetRecordForTest(name)
				record.Enabled = request.Enabled != nil && *request.Enabled
				if request.Targets != nil {
					record.Targets = *request.Targets
				}
				return record, nil
			},
		})

		if _, _, err := executeRootCommand(
			t,
			deps,
			"notifications", "preset", "enable", "task_terminal", "--target", "brg-1:#ops", "-o", "json",
		); err != nil {
			t.Fatalf("notifications preset enable error = %v", err)
		}
		if capturedName != "task_terminal" || captured.Enabled == nil || !*captured.Enabled ||
			captured.Targets == nil || len(*captured.Targets) != 1 {
			t.Fatalf("captured update = name %q request %#v", capturedName, captured)
		}
	})
}

func notificationPresetRecordForTest(name string) NotificationPresetRecord {
	return NotificationPresetRecord{
		Name:           name,
		Events:         []string{"task.run_*"},
		Targets:        []NotificationPresetTarget{{BridgeID: "brg-1", CanonicalRoute: "#ops"}},
		Enabled:        false,
		BuiltIn:        true,
		DefaultVersion: "1",
		DefaultHash:    "sha256:default",
		CreatedAt:      time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC),
	}
}
