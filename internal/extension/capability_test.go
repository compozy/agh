package extension

import (
	"errors"
	"slices"
	"testing"
)

func TestCapabilityCheckerCheckShouldAllowGrantedCapability(t *testing.T) {
	t.Parallel()

	checker := newTestCapabilityChecker(
		"ext",
		SourceUser,
		[]string{"sessions/list"},
		[]string{"session.read"},
	)

	if err := checker.Check("ext", "session.read"); err != nil {
		t.Fatalf("Check() error = %v, want nil", err)
	}
}

func TestCapabilityCheckerCheckShouldReturnCapabilityDenied(t *testing.T) {
	t.Parallel()

	checker := newTestCapabilityChecker(
		"ext",
		SourceUser,
		[]string{"sessions/list"},
		[]string{"session.read"},
	)

	err := checker.Check("ext", "session.write")
	if err == nil {
		t.Fatal("Check() error = nil, want capability denied")
	}

	var denied *ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Fatalf("Check() error type = %T, want *ErrCapabilityDenied", err)
	}
	if denied.Code() != CapabilityDeniedCode {
		t.Fatalf("Code() = %d, want %d", denied.Code(), CapabilityDeniedCode)
	}
	if denied.Data.Method != "session.write" {
		t.Fatalf("Data.Method = %q, want %q", denied.Data.Method, "session.write")
	}
	if !slices.Equal(denied.Data.Required, []string{"session.write"}) {
		t.Fatalf("Data.Required = %v, want %v", denied.Data.Required, []string{"session.write"})
	}
	if !slices.Equal(denied.Data.Granted, []string{"session.read"}) {
		t.Fatalf("Data.Granted = %v, want %v", denied.Data.Granted, []string{"session.read"})
	}
}

func TestCapabilityCheckerCheckHostAPIShouldEnforceDualGates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		actions      []string
		security     []string
		method       string
		wantRequired []string
		wantGranted  []string
		wantErr      bool
	}{
		{
			name:     "succeeds when action and security are granted",
			actions:  []string{"sessions/list"},
			security: []string{"session.read"},
			method:   "sessions/list",
		},
		{
			name:     "allows bridge list method with matching grant",
			actions:  []string{"bridges/instances/list"},
			security: []string{"bridge.read"},
			method:   "bridges/instances/list",
		},
		{
			name:     "allows bridge read method with matching grant",
			actions:  []string{"bridges/instances/get"},
			security: []string{"bridge.read"},
			method:   "bridges/instances/get",
		},
		{
			name:     "ShouldAllowBridgeStateReportWithWriteGrant",
			actions:  []string{"bridges/instances/report_state"},
			security: []string{"bridge.write"},
			method:   "bridges/instances/report_state",
		},
		{
			name:         "ShouldRejectBridgeStateReportWithoutActionGrant",
			actions:      []string{"bridges/instances/get"},
			security:     []string{"bridge.write"},
			method:       "bridges/instances/report_state",
			wantRequired: []string{"bridges/instances/report_state"},
			wantGranted:  []string{"bridges/instances/get"},
			wantErr:      true,
		},
		{
			name:         "ShouldRejectBridgeStateReportWithoutWriteGrant",
			actions:      []string{"bridges/instances/report_state"},
			security:     []string{"bridge.read"},
			method:       "bridges/instances/report_state",
			wantRequired: []string{"bridge.write"},
			wantGranted:  []string{"bridge.read"},
			wantErr:      true,
		},
		{
			name:         "fails when action grant is missing",
			actions:      nil,
			security:     []string{"session.read"},
			method:       "sessions/list",
			wantRequired: []string{"sessions/list"},
			wantGranted:  nil,
			wantErr:      true,
		},
		{
			name:         "fails when security grant is missing",
			actions:      []string{"sessions/list"},
			security:     []string{"observe.read"},
			method:       "sessions/list",
			wantRequired: []string{"session.read"},
			wantGranted:  []string{"observe.read"},
			wantErr:      true,
		},
		{
			name:     "automation read requires action and automation.read capability",
			actions:  []string{"automation/jobs"},
			security: []string{"automation.read"},
			method:   "automation/jobs",
		},
		{
			name:     "automation write requires action and automation.write capability",
			actions:  []string{"automation/jobs/create"},
			security: []string{"automation.write"},
			method:   "automation/jobs/create",
		},
		{
			name:         "fails for bridge write method without bridge security grant",
			actions:      []string{"bridges/messages/ingest"},
			security:     []string{"bridge.read"},
			method:       "bridges/messages/ingest",
			wantRequired: []string{"bridge.write"},
			wantGranted:  []string{"bridge.read"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := newTestCapabilityChecker("ext", SourceUser, tt.actions, tt.security)
			err := checker.CheckHostAPI("ext", tt.method)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("CheckHostAPI() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("CheckHostAPI() error = nil, want capability denied")
			}

			var denied *ErrCapabilityDenied
			if !errors.As(err, &denied) {
				t.Fatalf("CheckHostAPI() error type = %T, want *ErrCapabilityDenied", err)
			}
			if denied.Data.Method != tt.method {
				t.Fatalf("Data.Method = %q, want %q", denied.Data.Method, tt.method)
			}
			if !slices.Equal(denied.Data.Required, tt.wantRequired) {
				t.Fatalf("Data.Required = %v, want %v", denied.Data.Required, tt.wantRequired)
			}
			if !slices.Equal(denied.Data.Granted, tt.wantGranted) {
				t.Fatalf("Data.Granted = %v, want %v", denied.Data.Granted, tt.wantGranted)
			}
		})
	}
}

func TestCapabilityCheckerAutomationMethodsMapToExpectedCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method     string
		capability string
	}{
		{method: "automation/jobs", capability: "automation.read"},
		{method: "automation/jobs/get", capability: "automation.read"},
		{method: "automation/jobs/create", capability: "automation.write"},
		{method: "automation/jobs/update", capability: "automation.write"},
		{method: "automation/jobs/delete", capability: "automation.write"},
		{method: "automation/jobs/trigger", capability: "automation.write"},
		{method: "automation/jobs/runs", capability: "automation.read"},
		{method: "automation/triggers", capability: "automation.read"},
		{method: "automation/triggers/get", capability: "automation.read"},
		{method: "automation/triggers/create", capability: "automation.write"},
		{method: "automation/triggers/update", capability: "automation.write"},
		{method: "automation/triggers/delete", capability: "automation.write"},
		{method: "automation/triggers/runs", capability: "automation.read"},
		{method: "automation/triggers/fire", capability: "automation.write"},
		{method: "automation/runs", capability: "automation.read"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			checker := newTestCapabilityChecker("ext", SourceUser, []string{tt.method}, []string{tt.capability})
			if err := checker.CheckHostAPI("ext", tt.method); err != nil {
				t.Fatalf("CheckHostAPI(%q) error = %v, want nil", tt.method, err)
			}
		})
	}
}

func TestCapabilityCheckerRegisterShouldGrantRequestedCapabilitiesForTrustedSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source ExtensionSource
	}{
		{name: "bundled", source: SourceBundled},
		{name: "user", source: SourceUser},
		{name: "workspace", source: SourceWorkspace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := newTestCapabilityChecker(
				"ext",
				tt.source,
				[]string{"memory/store", "sessions/create"},
				[]string{"agent.pre_start", "memory.write", "permission.request", "session.write"},
			)

			for _, capability := range []string{"agent.pre_start", "memory.write", "permission.request", "session.write"} {
				if err := checker.Check("ext", capability); err != nil {
					t.Fatalf("Check(%q) error = %v, want nil", capability, err)
				}
			}
			for _, method := range []string{"memory/store", "sessions/create"} {
				if err := checker.CheckHostAPI("ext", method); err != nil {
					t.Fatalf("CheckHostAPI(%q) error = %v, want nil", method, err)
				}
			}
		})
	}
}

func TestCapabilityCheckerMarketplaceShouldDenyRestrictedCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		capability string
	}{
		{name: "permission family", capability: "permission.request"},
		{name: "session write", capability: "session.write"},
		{name: "memory write", capability: "memory.write"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := newTestCapabilityChecker("ext", SourceMarketplace, nil, []string{tt.capability})
			err := checker.Check("ext", tt.capability)
			if err == nil {
				t.Fatalf("Check(%q) error = nil, want capability denied", tt.capability)
			}
			var denied *ErrCapabilityDenied
			if !errors.As(err, &denied) {
				t.Fatalf("Check(%q) error type = %T, want *ErrCapabilityDenied", tt.capability, err)
			}
		})
	}
}

func TestCapabilityCheckerMarketplaceShouldAllowDefaultReadCapabilities(t *testing.T) {
	t.Parallel()

	checker := newTestCapabilityChecker(
		"ext",
		SourceMarketplace,
		[]string{"memory/recall", "observe/events", "sessions/list"},
		[]string{"*"},
	)

	for _, capability := range []string{"memory.read", "observe.read", "session.read"} {
		if err := checker.Check("ext", capability); err != nil {
			t.Fatalf("Check(%q) error = %v, want nil", capability, err)
		}
	}
	for _, method := range []string{"memory/recall", "observe/events", "sessions/list"} {
		if err := checker.CheckHostAPI("ext", method); err != nil {
			t.Fatalf("CheckHostAPI(%q) error = %v, want nil", method, err)
		}
	}
}

func TestCapabilityCheckerRegisterShouldApplyMarketplaceTierCeiling(t *testing.T) {
	t.Parallel()

	checker := &CapabilityChecker{}
	checker.Register("ext", SourceMarketplace, &Manifest{
		Actions: ActionsConfig{
			Requires: []string{
				"memory/recall",
				"memory/store",
				"sessions/create",
				"sessions/list",
				"skills/list",
			},
		},
		Security: SecurityConfig{
			Capabilities: []string{"*"},
		},
	})

	grant := checker.grants["ext"]
	if !slices.Equal(grant.actions, []string{"memory/recall", "sessions/list", "skills/list"}) {
		t.Fatalf("grant.actions = %v, want %v", grant.actions, []string{"memory/recall", "sessions/list", "skills/list"})
	}
	if !slices.Equal(grant.security, []string{"memory.read", "observe.read", "session.read", "skills.read", "tool.read"}) {
		t.Fatalf(
			"grant.security = %v, want %v",
			grant.security,
			[]string{"memory.read", "observe.read", "session.read", "skills.read", "tool.read"},
		)
	}
}

func TestCapabilityCheckerCheckShouldHonorGlobalWildcardGrant(t *testing.T) {
	t.Parallel()

	checker := newTestCapabilityChecker("ext", SourceUser, nil, []string{"*"})
	for _, capability := range []string{"agent.pre_start", "permission.request", "session.write"} {
		if err := checker.Check("ext", capability); err != nil {
			t.Fatalf("Check(%q) error = %v, want nil", capability, err)
		}
	}
}

func TestCapabilityCheckerCheckShouldHonorFamilyWildcardGrant(t *testing.T) {
	t.Parallel()

	checker := newTestCapabilityChecker("ext", SourceUser, nil, []string{"session.*"})
	for _, capability := range []string{"session.read", "session.write"} {
		if err := checker.Check("ext", capability); err != nil {
			t.Fatalf("Check(%q) error = %v, want nil", capability, err)
		}
	}

	if err := checker.Check("ext", "memory.read"); err == nil {
		t.Fatal("Check(memory.read) error = nil, want capability denied")
	}
}

func newTestCapabilityChecker(extName string, source ExtensionSource, actions []string, security []string) *CapabilityChecker {
	checker := &CapabilityChecker{}
	checker.Register(extName, source, &Manifest{
		Actions: ActionsConfig{
			Requires: actions,
		},
		Security: SecurityConfig{
			Capabilities: security,
		},
	})
	return checker
}
