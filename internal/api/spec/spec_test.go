package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/tools"
)

func TestDocumentTracksRequiredFieldsAndEnums(t *testing.T) {
	t.Parallel()

	doc, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	tests := []struct {
		name  string
		check func(t *testing.T, doc *openapi3.T)
	}{
		{
			name: "ShouldDescribeSessionListRequiredFieldsAndEnums",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				listSessions := operationFor(t, doc, "/api/sessions", "GET")
				listSessionsSchema := jsonResponseSchema(t, listSessions, 200)
				assertRequired(t, listSessionsSchema, "sessions")

				sessionsSchema := propertySchema(t, listSessionsSchema, "sessions")
				if sessionsSchema.Items == nil || sessionsSchema.Items.Value == nil {
					t.Fatal("expected sessions to define an items schema")
				}

				sessionSchema := sessionsSchema.Items.Value
				assertRequired(t, sessionSchema, "id", "agent_name", "state", "created_at", "updated_at")
				assertNotRequired(t, sessionSchema, "workspace_id", "workspace_path", "stop_reason", "stop_detail")
				assertEnumValues(t, propertySchema(t, sessionSchema, "state"), "starting", "active", "stopping", "stopped")
				assertEnumValues(t, propertySchema(t, sessionSchema, "stop_reason"),
					"completed",
					"user_canceled",
					"max_iterations",
					"loop_detected",
					"timeout",
					"budget_exceeded",
					"error",
					"agent_crashed",
					"hook_stopped",
					"shutdown",
				)
			},
		},
		{
			name: "ShouldDescribeCreateSessionOptionalFields",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				createSession := operationFor(t, doc, "/api/sessions", "POST")
				createSessionSchema := jsonRequestSchema(t, createSession)
				assertNotRequired(t, createSessionSchema, "agent_name", "name", "workspace", "workspace_path")
			},
		},
		{
			name: "ShouldDescribeApproveSessionRequiredFields",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				approveSession := operationFor(t, doc, "/api/sessions/{id}/approve", "POST")
				approveSchema := jsonRequestSchema(t, approveSession)
				assertRequired(t, approveSchema, "request_id", "turn_id", "decision")
			},
		},
		{
			name: "ShouldDescribeWriteMemoryRequiredAndOptionalFields",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				writeMemory := operationFor(t, doc, "/api/memory/{filename}", "PUT")
				writeMemorySchema := jsonRequestSchema(t, writeMemory)
				assertRequired(t, writeMemorySchema, "content")
				assertNotRequired(t, writeMemorySchema, "scope", "workspace")
			},
		},
		{
			name: "ShouldDescribeAutomationJobSchemasAndEnums",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				createJob := operationFor(t, doc, "/api/automation/jobs", "POST")
				createJobSchema := jsonRequestSchema(t, createJob)
				assertRequired(t, createJobSchema, "scope", "name", "agent_name", "prompt", "schedule")
				assertNotRequired(t, createJobSchema, "workspace_id", "enabled", "retry", "fire_limit")
				assertEnumValues(t, propertySchema(t, createJobSchema, "scope"), "global", "workspace")

				scheduleSchema := propertySchema(t, createJobSchema, "schedule")
				assertRequired(t, scheduleSchema, "mode")
				assertEnumValues(t, propertySchema(t, scheduleSchema, "mode"), "at", "cron", "every")

				retrySchema := propertySchema(t, createJobSchema, "retry")
				assertRequired(t, retrySchema, "strategy", "max_retries", "base_delay")
				assertEnumValues(t, propertySchema(t, retrySchema, "strategy"), "backoff", "none")
			},
		},
		{
			name: "ShouldDescribeAutomationTriggerAndHealthSchemas",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				createTrigger := operationFor(t, doc, "/api/automation/triggers", "POST")
				createTriggerSchema := jsonRequestSchema(t, createTrigger)
				assertRequired(t, createTriggerSchema, "scope", "name", "agent_name", "prompt", "event")
				assertNotRequired(t, createTriggerSchema, "workspace_id", "filter", "enabled", "retry", "fire_limit", "webhook_id", "endpoint_slug", "webhook_secret")
				assertEnumValues(t, propertySchema(t, createTriggerSchema, "scope"), "global", "workspace")

				healthOperation := operationFor(t, doc, "/api/observe/health", "GET")
				healthSchema := jsonResponseSchema(t, healthOperation, 200)
				assertRequired(t, healthSchema, "health", "memory", "automation")

				automationSchema := propertySchema(t, healthSchema, "automation")
				assertRequired(t, automationSchema, "enabled", "jobs", "triggers", "scheduler_running")
				assertNotRequired(t, automationSchema, "next_fire")
			},
		},
		{
			name: "ShouldDescribeWebhookHeadersAndAutomationRunEnums",
			check: func(t *testing.T, doc *openapi3.T) {
				t.Helper()

				webhookOperation := operationFor(t, doc, "/api/webhooks/global/{endpoint}", "POST")
				assertParameter(t, webhookOperation, "endpoint", openapi3.ParameterInPath, true)
				assertParameter(t, webhookOperation, "X-AGH-Webhook-Timestamp", openapi3.ParameterInHeader, true)
				assertParameter(t, webhookOperation, "X-AGH-Webhook-Signature", openapi3.ParameterInHeader, true)

				runOperation := operationFor(t, doc, "/api/automation/runs/{id}", "GET")
				runSchema := jsonResponseSchema(t, runOperation, 200)
				runPayloadSchema := propertySchema(t, runSchema, "run")
				assertEnumValues(t, propertySchema(t, runPayloadSchema, "status"), "cancelled", "completed", "failed", "running", "scheduled")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, doc)
		})
	}
}

func TestWriteFileAndEnumHelpers(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "openapi", "agh.json")
	if err := WriteFile(path); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("WriteFile() output is not valid JSON: %s", string(data))
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("WriteFile() output must end with newline: %q", string(data))
	}

	if got := hookSkillSourceValues(); len(got) == 0 {
		t.Fatal("hookSkillSourceValues() returned no values")
	}
	if got := hookExecutorKindValues(); len(got) == 0 {
		t.Fatal("hookExecutorKindValues() returned no values")
	}
	if got := toolSourceValues(); len(got) == 0 {
		t.Fatal("toolSourceValues() returned no values")
	}
	if got := hostAPIMethodValues(); len(got) == 0 || !slices.IsSorted(got) {
		t.Fatalf("hostAPIMethodValues() = %v, want non-empty sorted values", got)
	}
}

func TestSchemaCustomizerCoversAdditionalEnums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  any
	}{
		{name: "HookSkillSource", typ: hooks.HookSkillSource(hooks.HookSkillSourceBundled)},
		{name: "HookExecutorKind", typ: hooks.HookExecutorKind(hooks.HookExecutorNative)},
		{name: "ToolSource", typ: tools.ToolSource(0)},
		{name: "HostAPIMethod", typ: extensionprotocol.HostAPIMethod("memory.read")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema := openapi3.NewStringSchema()
			if err := schemaCustomizer("", reflect.TypeOf(tt.typ), "", schema); err != nil {
				t.Fatalf("schemaCustomizer() error = %v", err)
			}
			if len(schema.Enum) == 0 {
				t.Fatalf("schemaCustomizer() enum = %v, want non-empty", schema.Enum)
			}
		})
	}
}

func operationFor(t *testing.T, doc *openapi3.T, path string, method string) *openapi3.Operation {
	t.Helper()

	pathItem := doc.Paths.Value(path)
	if pathItem == nil {
		t.Fatalf("missing path %q", path)
	}
	operation := pathItem.GetOperation(method)
	if operation == nil {
		t.Fatalf("missing operation %s %s", method, path)
	}
	return operation
}

func jsonResponseSchema(t *testing.T, operation *openapi3.Operation, status int) *openapi3.Schema {
	t.Helper()

	responseRef := operation.Responses.Status(status)
	if responseRef == nil || responseRef.Value == nil {
		t.Fatalf("missing %d response", status)
	}
	mediaType := responseRef.Value.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
		t.Fatalf("missing application/json schema for %d response", status)
	}
	return mediaType.Schema.Value
}

func jsonRequestSchema(t *testing.T, operation *openapi3.Operation) *openapi3.Schema {
	t.Helper()

	if operation.RequestBody == nil || operation.RequestBody.Value == nil {
		t.Fatal("missing request body")
	}
	mediaType := operation.RequestBody.Value.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
		t.Fatal("missing application/json request schema")
	}
	return mediaType.Schema.Value
}

func propertySchema(t *testing.T, schema *openapi3.Schema, name string) *openapi3.Schema {
	t.Helper()

	propertyRef := schema.Properties[name]
	if propertyRef == nil || propertyRef.Value == nil {
		t.Fatalf("missing property %q", name)
	}
	return propertyRef.Value
}

func assertParameter(t *testing.T, operation *openapi3.Operation, name string, in string, required bool) {
	t.Helper()

	for _, ref := range operation.Parameters {
		if ref == nil || ref.Value == nil {
			continue
		}
		if ref.Value.Name == name && ref.Value.In == in {
			if ref.Value.Required != required {
				t.Fatalf("parameter %s in %s required = %v, want %v", name, in, ref.Value.Required, required)
			}
			return
		}
	}
	t.Fatalf("missing parameter %q in %s", name, in)
}

func assertRequired(t *testing.T, schema *openapi3.Schema, names ...string) {
	t.Helper()
	for _, name := range names {
		if !contains(schema.Required, name) {
			t.Fatalf("expected %q to be required, got %v", name, schema.Required)
		}
	}
}

func assertNotRequired(t *testing.T, schema *openapi3.Schema, names ...string) {
	t.Helper()
	for _, name := range names {
		if contains(schema.Required, name) {
			t.Fatalf("expected %q to be optional, got %v", name, schema.Required)
		}
	}
}

func assertEnumValues(t *testing.T, schema *openapi3.Schema, values ...string) {
	t.Helper()

	got := make([]string, 0, len(schema.Enum))
	for idx, value := range schema.Enum {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("expected enum[%d] to be string, got %T", idx, value)
		}
		got = append(got, text)
	}

	want := append([]string(nil), values...)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("expected enum values %v, got %v", want, got)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
