package spec

import (
	"slices"
	"testing"
)

func TestResourceOperationsSupportHTTPAndUDS(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"GET /api/resources":                "listResources",
		"GET /api/resources/{kind}":         "listResourcesByKind",
		"GET /api/resources/{kind}/{id}":    "getResource",
		"PUT /api/resources/{kind}/{id}":    "putResource",
		"DELETE /api/resources/{kind}/{id}": "deleteResource",
	}

	seen := make(map[string]OperationSpec, len(want))
	for _, op := range Operations() {
		key := op.Method + " " + op.Path
		if _, ok := want[key]; ok {
			seen[key] = op
		}
	}

	if len(seen) != len(want) {
		t.Fatalf("resource operations found = %d, want %d", len(seen), len(want))
	}

	for key, op := range seen {
		if op.OperationID != want[key] {
			t.Fatalf("%s operation_id = %q, want %q", key, op.OperationID, want[key])
		}
		if !slices.Equal(op.Transports, []Transport{TransportHTTP, TransportUDS}) {
			t.Fatalf("%s transports = %#v, want [http uds]", key, op.Transports)
		}
	}
}

func TestDocumentDescribesResourceCRUDSchemas(t *testing.T) {
	t.Parallel()

	doc, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	listResources := operationFor(t, doc, "/api/resources", "GET")
	assertParameter(t, listResources, "kind", "query", false)
	assertParameter(t, listResources, "scope_kind", "query", false)
	assertParameter(t, listResources, "limit", "query", false)

	putResource := operationFor(t, doc, "/api/resources/{kind}/{id}", "PUT")
	putSchema := jsonRequestSchema(t, putResource)
	assertRequired(t, putSchema, "scope", "spec")
	assertNotRequired(t, putSchema, "expected_version")
	scopeSchema := propertySchema(t, putSchema, "scope")
	assertEnumValues(t, propertySchema(t, scopeSchema, "kind"), "global", "workspace")

	deleteResource := operationFor(t, doc, "/api/resources/{kind}/{id}", "DELETE")
	deleteSchema := jsonRequestSchema(t, deleteResource)
	assertRequired(t, deleteSchema, "expected_version")
	assertNotRequired(t, deleteSchema, "scope", "spec")
}
