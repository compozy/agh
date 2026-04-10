package spec

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestDocumentTracksRequiredFieldsAndEnums(t *testing.T) {
	t.Parallel()

	doc, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	t.Run("ShouldDescribeSessionListRequiredFieldsAndEnums", func(t *testing.T) {
		listSessions := operationFor(t, doc, "/api/sessions", "GET")
		listSessionsSchema := jsonResponseSchema(t, listSessions, 200)
		assertRequired(t, listSessionsSchema, "sessions")

		sessionSchema := propertySchema(t, listSessionsSchema, "sessions").Items.Value
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
	})

	t.Run("ShouldDescribeCreateSessionOptionalFields", func(t *testing.T) {
		createSession := operationFor(t, doc, "/api/sessions", "POST")
		createSessionSchema := jsonRequestSchema(t, createSession)
		assertNotRequired(t, createSessionSchema, "agent_name", "name", "workspace", "workspace_path")
	})

	t.Run("ShouldDescribeApproveSessionRequiredFields", func(t *testing.T) {
		approveSession := operationFor(t, doc, "/api/sessions/{id}/approve", "POST")
		approveSchema := jsonRequestSchema(t, approveSession)
		assertRequired(t, approveSchema, "request_id", "turn_id", "decision")
	})

	t.Run("ShouldDescribeWriteMemoryRequiredAndOptionalFields", func(t *testing.T) {
		writeMemory := operationFor(t, doc, "/api/memory/{filename}", "PUT")
		writeMemorySchema := jsonRequestSchema(t, writeMemory)
		assertRequired(t, writeMemorySchema, "content")
		assertNotRequired(t, writeMemorySchema, "scope", "workspace")
	})
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
	for _, value := range schema.Enum {
		text, ok := value.(string)
		if ok {
			got = append(got, text)
		}
	}
	for _, value := range values {
		if !contains(got, value) {
			t.Fatalf("expected enum %q in %v", value, got)
		}
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
