package contract

import (
	"encoding/json"
	"testing"
)

func TestMutationResultJSONShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       MutationResult
		wantPresent map[string]any
		wantAbsent  []string
	}{
		{
			name: "ShouldOmitOptionalFieldsWhenUnset",
			input: MutationResult{
				Section:         SettingsSectionGeneral,
				Scope:           SettingsScopeGlobal,
				Behavior:        SettingsMutationBehaviorRestartRequired,
				Applied:         false,
				RestartRequired: true,
			},
			wantPresent: map[string]any{
				"section":          string(SettingsSectionGeneral),
				"scope":            string(SettingsScopeGlobal),
				"behavior":         string(SettingsMutationBehaviorRestartRequired),
				"applied":          false,
				"restart_required": true,
			},
			wantAbsent: []string{"write_target", "workspace_id", "restart_scope", "warnings"},
		},
		{
			name: "ShouldPreserveSemanticWriteTargetAndWorkspaceMetadata",
			input: MutationResult{
				Section:         SettingsSectionHooksExtensions,
				Scope:           SettingsScopeWorkspace,
				WriteTarget:     SettingsWriteTargetWorkspaceMCPSidecar,
				WorkspaceID:     "ws-alpha",
				Behavior:        SettingsMutationBehaviorAppliedNow,
				Applied:         true,
				RestartRequired: false,
				RestartScope:    "daemon",
				Warnings:        []string{"restart deferred"},
			},
			wantPresent: map[string]any{
				"section":          string(SettingsSectionHooksExtensions),
				"scope":            string(SettingsScopeWorkspace),
				"write_target":     string(SettingsWriteTargetWorkspaceMCPSidecar),
				"workspace_id":     "ws-alpha",
				"behavior":         string(SettingsMutationBehaviorAppliedNow),
				"applied":          true,
				"restart_required": false,
				"restart_scope":    "daemon",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			for key, want := range tt.wantPresent {
				got, ok := decoded[key]
				if !ok {
					t.Fatalf("decoded JSON missing key %q in %s", key, string(data))
				}
				switch wantValue := want.(type) {
				case bool:
					gotBool, ok := got.(bool)
					if !ok || gotBool != wantValue {
						t.Fatalf("%s = %#v, want %v", key, got, wantValue)
					}
				default:
					if got != want {
						t.Fatalf("%s = %#v, want %#v", key, got, want)
					}
				}
			}

			for _, key := range tt.wantAbsent {
				if _, ok := decoded[key]; ok {
					t.Fatalf("decoded JSON unexpectedly included %q in %s", key, string(data))
				}
			}

			if tt.input.Warnings != nil {
				warnings, ok := decoded["warnings"].([]any)
				if !ok || len(warnings) != len(tt.input.Warnings) {
					t.Fatalf("warnings = %#v, want %d entries", decoded["warnings"], len(tt.input.Warnings))
				}
				for idx, want := range tt.input.Warnings {
					if warnings[idx] != want {
						t.Fatalf("warnings[%d] = %#v, want %q", idx, warnings[idx], want)
					}
				}
			}
		})
	}
}
