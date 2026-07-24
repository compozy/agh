package daemon

import (
	"errors"
	"reflect"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
)

func TestRoleStatusProjection(t *testing.T) {
	t.Parallel()

	t.Run("Should return the closed roster sorted with truthful provenance", func(t *testing.T) {
		t.Parallel()

		cfg := roleResolverConfig()
		statuses, err := newRoleResolver(&cfg, nil, nil).RoleStatuses(t.Context(), "")
		if err != nil {
			t.Fatalf("RoleStatuses() error = %v", err)
		}
		roles := make([]string, 0, len(statuses))
		for _, status := range statuses {
			roles = append(roles, status.Role)
			assertRoleStatusProvenance(t, status)
		}
		want := []string{
			"auto_title",
			"checkpoint_summary",
			"coordinator",
			"dream",
			"memory_controller",
			"memory_extractor",
		}
		if !reflect.DeepEqual(roles, want) {
			t.Fatalf("RoleStatuses() roles = %#v, want %#v", roles, want)
		}
		if statuses[0].ResolutionMode != contract.RoleResolutionModeInherit || statuses[0].Agent != nil {
			t.Fatalf("auto_title status = %#v, want invocation-time inheritance", statuses[0])
		}
	})

	t.Run("Should report a missing catalog agent without failing projection", func(t *testing.T) {
		t.Parallel()

		cfg := roleResolverConfig()
		cfg.Roles.Dream.Agent = "missing-curator"
		status, err := newRoleResolver(&cfg, nil, roleAgentResolverStub{}).RoleStatus(
			t.Context(),
			"",
			string(aghconfig.RoleDream),
		)
		if err != nil {
			t.Fatalf("RoleStatus(dream) error = %v", err)
		}
		if len(status.Diagnostics) != 1 || status.Diagnostics[0].Code != contract.CodeRoleAgentNotFound ||
			status.Diagnostics[0].Agent != "missing-curator" {
			t.Fatalf("RoleStatus(dream).Diagnostics = %#v", status.Diagnostics)
		}
		if status.Agent == nil || *status.Agent != "missing-curator" ||
			status.ResolutionMode != contract.RoleResolutionModeCatalog {
			t.Fatalf("RoleStatus(dream) = %#v, want missing catalog projection", status)
		}
	})

	t.Run("Should return the stable unknown role code", func(t *testing.T) {
		t.Parallel()

		cfg := roleResolverConfig()
		_, err := newRoleResolver(&cfg, nil, nil).RoleStatus(t.Context(), "", "judge")
		var resolutionErr *RoleResolutionError
		if !errors.As(err, &resolutionErr) || resolutionErr.DiagnosticCode() != contract.CodeRoleUnknown {
			t.Fatalf("RoleStatus(judge) error = %v, want role_unknown", err)
		}
	})
}

func assertRoleStatusProvenance(t *testing.T, status contract.RoleStatus) {
	t.Helper()
	for _, field := range []struct {
		name    string
		present bool
	}{
		{name: roleFieldEnabled, present: true},
		{name: "fallback_chain", present: true},
		{name: "agent", present: status.Agent != nil},
		{name: "provider", present: status.Provider != nil},
		{name: "model", present: status.Model != nil},
		{name: "reasoning_effort", present: status.ReasoningEffort != nil},
		{name: "timeout", present: status.Timeout != nil},
	} {
		_, exists := status.Provenance[field.name]
		if exists != field.present {
			t.Fatalf(
				"RoleStatus(%s).Provenance[%s] exists = %t, want %t: %#v",
				status.Role,
				field.name,
				exists,
				field.present,
				status.Provenance,
			)
		}
	}
}
