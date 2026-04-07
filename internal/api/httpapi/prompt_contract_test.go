package httpapi

import (
	"reflect"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
)

func TestPromptStreamPayloadsRemainTransportLocal(t *testing.T) {
	t.Parallel()

	t.Run("Should keep transport payloads local and separate from shared contract", func(t *testing.T) {
		t.Parallel()

		promptPkg := reflect.TypeOf(promptRequest{}).PkgPath()
		transportPkg := reflect.TypeOf(agentEventPayload{}).PkgPath()
		sharedPkg := reflect.TypeOf(contract.AgentEventPayload{}).PkgPath()

		if promptPkg != transportPkg {
			t.Fatalf("prompt payload package = %q, agent event package = %q", promptPkg, transportPkg)
		}
		if transportPkg == sharedPkg {
			t.Fatalf("transport-local payload unexpectedly uses shared contract package %q", sharedPkg)
		}

		transportTimestamp, ok := reflect.TypeOf(agentEventPayload{}).FieldByName("Timestamp")
		if !ok {
			t.Fatal("agentEventPayload.Timestamp field is missing")
		}
		if transportTimestamp.Type.Kind() != reflect.String {
			t.Fatalf("transport timestamp type = %v, want string", transportTimestamp.Type)
		}

		sharedTimestamp, ok := reflect.TypeOf(contract.AgentEventPayload{}).FieldByName("Timestamp")
		if !ok {
			t.Fatal("contract.AgentEventPayload.Timestamp field is missing")
		}
		if sharedTimestamp.Type != reflect.TypeOf(time.Time{}) {
			t.Fatalf("shared timestamp type = %v, want time.Time", sharedTimestamp.Type)
		}
	})
}
