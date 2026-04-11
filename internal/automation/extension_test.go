package automation

import "testing"

func TestExtensionTriggerRequestValidateRequiresExtPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request ExtensionTriggerRequest
		wantErr bool
	}{
		{
			name: "accepts workspace scoped ext event",
			request: ExtensionTriggerRequest{
				Event:       "ext.github.push",
				Scope:       AutomationScopeWorkspace,
				WorkspaceID: "ws-1",
				Payload:     map[string]any{"repo": "acme/api"},
			},
		},
		{
			name: "rejects built in event names",
			request: ExtensionTriggerRequest{
				Event: "session.stopped",
				Scope: AutomationScopeGlobal,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate("trigger_fire")
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}
