package spec

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestAuthoredContextOpenAPIContracts(t *testing.T) {
	t.Run("Should register shared Soul Heartbeat health and wake operations", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		for _, target := range []struct {
			path   string
			method string
			status int
		}{
			{path: "/api/agent/soul", method: "GET", status: 200},
			{path: "/api/agents/{agent_name}/soul", method: "PUT", status: 200},
			{path: "/api/agents/{agent_name}/heartbeat", method: "GET", status: 200},
			{path: "/api/agents/{agent_name}/heartbeat/status", method: "GET", status: 200},
			{path: "/api/agents/{agent_name}/heartbeat/wake", method: "POST", status: 200},
			{path: "/api/sessions/{session_id}/health", method: "GET", status: 200},
			{path: "/api/sessions/{session_id}/inspect", method: "GET", status: 200},
		} {
			operation := operationFor(t, doc, target.path, target.method)
			assertResponseStatus(t, operation, target.status)
		}
	})

	t.Run("Should keep compact agent context Soul projection body-free", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		contextOperation := operationFor(t, doc, "/api/agent/context", "GET")
		contextSchema := jsonResponseSchema(t, contextOperation, 200)
		contextPayloadSchema := propertySchema(t, contextSchema, "context")
		soulSchema := propertySchema(t, contextPayloadSchema, "soul")
		assertRequired(t, soulSchema, "enabled", "present", "active", "valid", "tone", "principles")
		assertEnumValues(t, propertySchema(t, soulSchema, "validation_status"),
			"missing",
			"inactive",
			"valid",
			"invalid",
		)
		if property := soulSchema.Properties["body"]; property != nil {
			t.Fatalf("compact Soul schema exposed full body property: %#v", property)
		}
	})

	t.Run("Should require expected digest inside Soul mutation request bodies", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		putSoul := operationFor(t, doc, "/api/agents/{agent_name}/soul", "PUT")
		putSoulSchema := jsonRequestSchema(t, putSoul)
		assertRequired(t, putSoulSchema, "agent_name", "body", "expected_digest")
		assertNotRequired(t, putSoulSchema, "workspace_id", "idempotency_key")

		deleteSoul := operationFor(t, doc, "/api/agents/{agent_name}/soul", "DELETE")
		deleteSoulSchema := jsonRequestSchema(t, deleteSoul)
		assertRequired(t, deleteSoulSchema, "agent_name", "expected_digest")
		if property := deleteSoulSchema.Properties["if_match"]; property != nil {
			t.Fatalf("Soul delete request schema exposed transport-specific if_match: %#v", property)
		}
	})

	t.Run("Should require expected digest inside Heartbeat mutation request bodies", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		putHeartbeat := operationFor(t, doc, "/api/agents/{agent_name}/heartbeat", "PUT")
		putHeartbeatSchema := jsonRequestSchema(t, putHeartbeat)
		assertRequired(t, putHeartbeatSchema, "agent_name", "body", "expected_digest")
		assertNotRequired(t, putHeartbeatSchema, "workspace_id", "idempotency_key")

		rollbackHeartbeat := operationFor(t, doc, "/api/agents/{agent_name}/heartbeat/rollback", "POST")
		rollbackHeartbeatSchema := jsonRequestSchema(t, rollbackHeartbeat)
		assertRequired(t, rollbackHeartbeatSchema, "agent_name", "expected_digest")
		assertNotRequired(t, rollbackHeartbeatSchema, "revision_id", "target_digest")
		if property := rollbackHeartbeatSchema.Properties["if_match"]; property != nil {
			t.Fatalf("Heartbeat rollback request schema exposed transport-specific if_match: %#v", property)
		}
	})

	t.Run("Should describe closed diagnostics health and wake enums", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		getSoul := operationFor(t, doc, "/api/agent/soul", "GET")
		soulSchema := jsonResponseSchema(t, getSoul, 200)
		assertEnumValues(t, propertySchema(t, soulSchema, "validation_status"),
			"missing",
			"inactive",
			"valid",
			"invalid",
		)
		diagnosticsSchema := propertySchema(t, soulSchema, "diagnostics")
		if diagnosticsSchema.Items == nil || diagnosticsSchema.Items.Value == nil {
			t.Fatal("Soul diagnostics schema missing items")
		}
		assertEnumValues(t, propertySchema(t, diagnosticsSchema.Items.Value, "severity"),
			"info",
			"warning",
			"error",
		)

		healthOperation := operationFor(t, doc, "/api/sessions/{session_id}/health", "GET")
		healthResponseSchema := jsonResponseSchema(t, healthOperation, 200)
		healthSchema := propertySchema(t, healthResponseSchema, "health")
		assertEnumValues(t, propertySchema(t, healthSchema, "state"),
			"idle",
			"prompting",
			"stopped",
			"detached",
		)
		assertEnumValues(t, propertySchema(t, healthSchema, "health"),
			"healthy",
			"degraded",
			"stale",
			"dead",
			"unknown",
		)
		assertEnumValues(t, propertySchema(t, healthSchema, "ineligibility_reason"),
			"session_prompt_active",
			"session_not_attachable",
			"session_unhealthy",
			"session_health_stale",
			"session_health_hung",
			"session_health_dead",
			"session_health_unknown",
		)

		wakeOperation := operationFor(t, doc, "/api/agents/{agent_name}/heartbeat/wake", "POST")
		wakeResponseSchema := jsonResponseSchema(t, wakeOperation, 200)
		decisionSchema := propertySchema(t, wakeResponseSchema, "decision")
		assertEnumValues(t, propertySchema(t, decisionSchema, "result"),
			"sent",
			"skipped",
			"coalesced",
			"rate_limited",
			"failed",
		)
		assertEnumValues(t, propertySchema(t, decisionSchema, "reason"),
			"wake_sent",
			"heartbeat_disabled",
			"heartbeat_invalid",
			"heartbeat_no_policy",
			"heartbeat_rate_limited",
			"heartbeat_no_eligible_session",
			"cooldown_active",
			"quiet_window",
			"session_not_found",
			"session_unhealthy",
			"session_not_attachable",
			"session_prompt_active",
			"session_prompt_active_race",
			"synthetic_prompt_failed",
			"wake_coalesced",
		)
	})

	t.Run("Should expose HTTP and UDS transport parity on new operations", func(t *testing.T) {
		t.Parallel()

		doc := authoredContextDocument(t)
		for _, target := range []struct {
			path   string
			method string
		}{
			{path: "/api/agent/soul", method: "GET"},
			{path: "/api/agents/{agent_name}/heartbeat/status", method: "GET"},
			{path: "/api/sessions/{session_id}/inspect", method: "GET"},
		} {
			operation := operationFor(t, doc, target.path, target.method)
			assertOperationTransports(t, operation, TransportHTTP, TransportUDS)
		}
	})
}

func authoredContextDocument(t *testing.T) *openapi3.T {
	t.Helper()

	doc, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}
	return doc
}
