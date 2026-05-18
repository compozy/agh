package bridgesdk

import (
	"testing"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestSessionInitializeRequestClonesHandshakeSlicesClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should clone supported protocol and granted resource slices", func(t *testing.T) {
		t.Parallel()

		session := &Session{
			request: subprocess.InitializeRequest{
				SupportedProtocolVersion: []string{"1"},
				Capabilities: subprocess.InitializeCapabilities{
					Provides: []string{"bridge.adapter"},
					GrantedActions: []extensionprotocol.HostAPIMethod{
						extensionprotocol.HostAPIMethodBridgesInstancesList,
					},
					GrantedSecurity:       []string{"bridge.read"},
					GrantedResourceKinds:  []resources.ResourceKind{resources.ResourceKind("agent")},
					GrantedResourceScopes: []resources.ResourceScopeKind{resources.ResourceScopeKindWorkspace},
				},
			},
		}

		request := session.InitializeRequest()
		request.SupportedProtocolVersion[0] = "mutated"
		request.Capabilities.GrantedResourceKinds[0] = resources.ResourceKind("mutated")
		request.Capabilities.GrantedResourceScopes[0] = resources.ResourceScopeKind("mutated")

		again := session.InitializeRequest()
		if got, want := again.SupportedProtocolVersion[0], "1"; got != want {
			t.Fatalf("SupportedProtocolVersion[0] = %q, want %q", got, want)
		}
		if got, want := again.Capabilities.GrantedResourceKinds[0], resources.ResourceKind("agent"); got != want {
			t.Fatalf("GrantedResourceKinds[0] = %q, want %q", got, want)
		}
		if got, want := again.Capabilities.GrantedResourceScopes[0], resources.ResourceScopeKindWorkspace; got != want {
			t.Fatalf("GrantedResourceScopes[0] = %q, want %q", got, want)
		}
	})
}
