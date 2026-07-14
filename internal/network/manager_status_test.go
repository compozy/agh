// Suite: Network manager status
// Invariant: status is disabled, ready, or active according to availability and Live participation.
// Boundary IN: the real in-process Network manager and participant registry.
// Boundary OUT: HTTP/OpenAPI projection, owned by internal/api/core/network_test.go.
package network

import (
	"context"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
)

func TestManagerStatusReportsAvailabilityAndParticipation(t *testing.T) {
	t.Parallel()

	t.Run("Should report disabled ready and active without transport states", func(t *testing.T) {
		t.Parallel()

		manager, err := NewManager(
			t.Context(),
			aghconfig.DefaultNetworkConfig(),
			"",
			nil,
			WithManagerLogger(discardManagerLogger()),
			WithManagerAuditWriter(managerAuditWriterStub{}),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Errorf("Shutdown() error = %v", err)
			}
		})

		ready, err := manager.Status(t.Context())
		if err != nil {
			t.Fatalf("Status(ready) error = %v", err)
		}
		if ready.Status != StatusReady || ready.LocalPeers != 0 {
			t.Fatalf("Status(ready) = %#v, want ready with zero Live participants", ready)
		}

		joinManagerSendParticipant(t, manager, "sess-live", "coder.sess-live")
		active, err := manager.Status(t.Context())
		if err != nil {
			t.Fatalf("Status(active) error = %v", err)
		}
		if active.Status != StatusActive || active.LocalPeers != 1 {
			t.Fatalf("Status(active) = %#v, want active with one Live participant", active)
		}

		manager.SetEnabled(false)
		disabled, err := manager.Status(t.Context())
		if err != nil {
			t.Fatalf("Status(disabled) error = %v", err)
		}
		if disabled.Status != StatusDisabled || disabled.Enabled {
			t.Fatalf("Status(disabled) = %#v, want disabled availability", disabled)
		}
	})
}
