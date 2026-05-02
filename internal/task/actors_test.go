package task

import "testing"

func TestDeriveAgentSessionActorContextForOrigin(t *testing.T) {
	t.Parallel()

	t.Run("Should derive actor context for HTTP agent ingress", func(t *testing.T) {
		t.Parallel()

		actor, err := DeriveAgentSessionActorContextForOrigin(
			"sess-http",
			OriginKindHTTP,
			"agent.context",
		)
		if err != nil {
			t.Fatalf("DeriveAgentSessionActorContextForOrigin() error = %v", err)
		}
		if actor.Actor.Kind != ActorKindAgentSession ||
			actor.Actor.Ref != "sess-http" ||
			actor.Origin.Kind != OriginKindHTTP ||
			actor.Origin.Ref != "agent.context" {
			t.Fatalf("actor = %#v, want HTTP agent-session origin", actor)
		}
	})

	t.Run("Should reject browser web origin for agent-session ingress", func(t *testing.T) {
		t.Parallel()

		_, err := DeriveAgentSessionActorContextForOrigin(
			"sess-web",
			OriginKindWeb,
			"agent.context",
		)
		if err == nil {
			t.Fatal("DeriveAgentSessionActorContextForOrigin() error = nil, want validation error")
		}
	})
}
