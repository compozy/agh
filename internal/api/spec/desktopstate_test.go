package spec

import (
	"slices"
	"testing"

	"github.com/compozy/agh/internal/api/contract"
)

func TestDesktopStateOperations(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		httpMethodGet + " " + desktopStateCollectionPath:             "listDesktopState",
		httpMethodGet + " " + desktopStateItemPath:                   "getDesktopState",
		httpMethodPut + " " + desktopStateItemPath:                   "putDesktopState",
		httpMethodPost + " " + desktopStateCollectionPath + "/apply": "applyDesktopState",
		httpMethodDelete + " " + desktopStateItemPath:                "deleteDesktopState",
		httpMethodGet + " " + desktopStateCollectionPath + "/stream": "streamDesktopState",
	}
	seen := make(map[string]struct{}, len(expected))
	for _, operation := range Operations() {
		key := operation.Method + " " + operation.Path
		operationID, ok := expected[key]
		if !ok {
			continue
		}
		seen[key] = struct{}{}
		if operation.OperationID != operationID {
			t.Fatalf("%s operation id = %q, want %q", key, operation.OperationID, operationID)
		}
		if !slices.Equal(operation.Transports, []Transport{TransportHTTP, TransportUDS}) {
			t.Fatalf("%s transports = %#v, want HTTP and UDS", key, operation.Transports)
		}
		for _, parameter := range operation.Parameters {
			if parameter.Name == "domain" {
				t.Fatalf("%s exposes forbidden public domain parameter", key)
			}
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("desktop-state operations seen = %d, want %d; seen=%v", len(seen), len(expected), seen)
	}
}

func TestDesktopStateOpenAPIContract(t *testing.T) {
	t.Parallel()
	document, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	for _, component := range desktopStateComponentSchemas {
		if schema := document.Components.Schemas[component.name]; schema == nil || schema.Value == nil {
			t.Fatalf("component %q is missing", component.name)
		}
	}
	entry := document.Components.Schemas["DesktopStateEntry"].Value
	assertRequired(t, entry, "key", "value", "rev", "seq", "deleted", "updated_at")
	value := propertySchema(t, entry, "value")
	if !value.Nullable {
		t.Fatal("DesktopStateEntry.value is not nullable")
	}
	for _, field := range []string{"rev", "seq"} {
		number := propertySchema(t, entry, field)
		if number.Max == nil || *number.Max != float64(contract.DesktopStateMaxSafeNumber) {
			t.Fatalf(
				"DesktopStateEntry.%s maximum = %v, want %d",
				field,
				number.Max,
				contract.DesktopStateMaxSafeNumber,
			)
		}
	}

	frameOps := map[string]string{
		"DesktopStateSubscribeFrame": "sub",
		"DesktopStateApplyFrame":     "apply",
		"DesktopStatePingFrame":      "ping",
		"DesktopStateSnapshotFrame":  "snapshot",
		"DesktopStateEventFrame":     "event",
		"DesktopStateAckFrame":       "ack",
		"DesktopStateErrorFrame":     "error",
		"DesktopStatePongFrame":      "pong",
	}
	for component, op := range frameOps {
		schema := document.Components.Schemas[component].Value
		assertEnumValues(t, propertySchema(t, schema, "op"), op)
	}

	stream := operationFor(t, document, desktopStateCollectionPath+"/stream", httpMethodGet)
	assertResponseStatus(t, stream, 101)
	assertResponseStatus(t, stream, 404)
	deleteOperation := operationFor(t, document, desktopStateItemPath, httpMethodDelete)
	ifRev := parameterSchema(t, deleteOperation, "if_rev", "query")
	if ifRev.Max == nil || *ifRev.Max != float64(contract.DesktopStateMaxSafeNumber) {
		t.Fatalf("delete if_rev maximum = %v, want %d", ifRev.Max, contract.DesktopStateMaxSafeNumber)
	}
}
