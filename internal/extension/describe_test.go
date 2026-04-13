package extension

import (
	"testing"
	"time"
)

func TestDescribeExtension(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 18, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		extension  *Extension
		active     bool
		now        time.Time
		wantType   string
		wantState  string
		wantHealth string
		wantUptime int64
	}{
		{
			name: "Should report active subprocess runtime",
			extension: &Extension{
				Info: ExtensionInfo{
					Name:    "telegram-adapter",
					Version: "1.2.3",
					Source:  SourceUser,
					Enabled: true,
					Capabilities: CapabilitiesConfig{
						Provides: []string{"bridge.adapter"},
					},
					Actions: ActionsConfig{
						Requires: []string{"bridges/messages/ingest"},
					},
				},
				Status: ExtensionStatus{
					Active:        true,
					Healthy:       true,
					PID:           4242,
					LastStartedAt: now.Add(-15 * time.Minute),
				},
			},
			active:     true,
			now:        now,
			wantType:   "subprocess",
			wantState:  "active",
			wantHealth: "healthy",
			wantUptime: 900,
		},
		{
			name: "Should report registered resource health",
			extension: &Extension{
				Info: ExtensionInfo{
					Name:    "workspace-review",
					Version: "0.1.0",
					Source:  SourceWorkspace,
					Enabled: true,
				},
				Status: ExtensionStatus{
					Registered: true,
				},
			},
			active:     true,
			now:        now,
			wantType:   "resource",
			wantState:  "registered",
			wantHealth: "healthy",
			wantUptime: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := DescribeExtension(tt.extension, tt.active, tt.now)
			if payload.Type != tt.wantType {
				t.Fatalf("DescribeExtension().Type = %q, want %q", payload.Type, tt.wantType)
			}
			if payload.State != tt.wantState {
				t.Fatalf("DescribeExtension().State = %q, want %q", payload.State, tt.wantState)
			}
			if payload.Health != tt.wantHealth {
				t.Fatalf("DescribeExtension().Health = %q, want %q", payload.Health, tt.wantHealth)
			}
			if payload.UptimeSeconds != tt.wantUptime {
				t.Fatalf("DescribeExtension().UptimeSeconds = %d, want %d", payload.UptimeSeconds, tt.wantUptime)
			}
		})
	}
}
