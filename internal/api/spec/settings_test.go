package spec

import (
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSettingsRoutesAndSchemas(t *testing.T) {
	t.Parallel()

	doc, err := Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	t.Run("ShouldRegisterSettingsRoutesAndHTTPExtensionParity", func(t *testing.T) {
		t.Parallel()

		operations := []struct {
			path       string
			method     string
			transports []Transport
		}{
			{path: "/api/settings/general", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/general", method: "PATCH", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/memory", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/memory", method: "PATCH", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/skills", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/skills", method: "PATCH", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/automation", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/automation", method: "PATCH", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/network", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/network", method: "PATCH", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/observability", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/observability",
				method:     "PATCH",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/observability/log-tail",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/hooks-extensions",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/hooks-extensions",
				method:     "PATCH",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{path: "/api/settings/providers", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/providers/{name}",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/providers/{name}",
				method:     "PUT",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/providers/{name}",
				method:     "DELETE",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{path: "/api/settings/mcp-servers", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/mcp-servers/{name}",
				method:     "PUT",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/mcp-servers/{name}",
				method:     "DELETE",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{path: "/api/settings/environments", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/environments/{name}",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/environments/{name}",
				method:     "PUT",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/environments/{name}",
				method:     "DELETE",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{path: "/api/settings/hooks", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/hooks/{name}", method: "PUT", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/hooks/{name}",
				method:     "DELETE",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/actions/restart",
				method:     "POST",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/actions/restart/{operation_id}",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{path: "/api/extensions", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/extensions", method: "POST", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/extensions/{name}", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/extensions/{name}/enable",
				method:     "POST",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/extensions/{name}/disable",
				method:     "POST",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
		}

		for _, operation := range operations {
			t.Run(operation.method+" "+operation.path, func(t *testing.T) {
				t.Parallel()

				op := operationFor(t, doc, operation.path, operation.method)
				assertOperationTransports(t, op, operation.transports...)
			})
		}
	})

	t.Run("ShouldDescribeSettingsMutationAndRestartSchemas", func(t *testing.T) {
		t.Parallel()

		updateGeneral := operationFor(t, doc, "/api/settings/general", "PATCH")
		updateGeneralSchema := jsonRequestSchema(t, updateGeneral)
		assertRequired(t, updateGeneralSchema, "config")

		generalConfig := propertySchema(t, updateGeneralSchema, "config")
		assertRequired(t, generalConfig, "defaults", "limits", "permissions", "session_timeout", "http", "daemon")
		assertEnumValues(t, propertySchema(t, propertySchema(t, generalConfig, "permissions"), "mode"),
			"approve-all",
			"approve-reads",
			"deny-all",
		)

		mutationSchema := jsonResponseSchema(t, updateGeneral, 200)
		assertRequired(t, mutationSchema, "section", "scope", "behavior", "applied", "restart_required")
		assertNotRequired(t, mutationSchema, "write_target", "workspace_id", "restart_scope", "warnings")
		assertEnumValues(t, propertySchema(t, mutationSchema, "section"),
			"automation",
			"general",
			"hooks-extensions",
			"memory",
			"network",
			"observability",
			"skills",
		)
		assertEnumValues(t, propertySchema(t, mutationSchema, "scope"), "global", "workspace")
		assertEnumValues(t, propertySchema(t, mutationSchema, "behavior"),
			"action_trigger",
			"applied_now",
			"restart_required",
		)
		assertEnumValues(t, propertySchema(t, mutationSchema, "write_target"),
			"global-config",
			"global-mcp-sidecar",
			"workspace-config",
			"workspace-mcp-sidecar",
		)

		restartAction := operationFor(t, doc, "/api/settings/actions/restart", "POST")
		restartActionSchema := jsonResponseSchema(t, restartAction, 202)
		assertRequired(t, restartActionSchema, "operation_id", "status", "status_url", "active_session_count")
		assertEnumValues(t, propertySchema(t, restartActionSchema, "status"),
			"failed",
			"pending",
			"ready",
			"starting",
			"stopping",
			"waiting_release",
		)

		restartStatus := operationFor(t, doc, "/api/settings/actions/restart/{operation_id}", "GET")
		restartStatusSchema := jsonResponseSchema(t, restartStatus, 200)
		assertRequired(
			t,
			restartStatusSchema,
			"operation_id",
			"status",
			"old_pid",
			"old_started_at",
			"old_socket_path",
			"active_session_count",
			"started_at",
			"updated_at",
		)
		assertNotRequired(t, restartStatusSchema, "new_pid", "failure_reason", "completed_at")
		assertEnumValues(t, propertySchema(t, restartStatusSchema, "status"),
			"failed",
			"pending",
			"ready",
			"starting",
			"stopping",
			"waiting_release",
		)
	})

	t.Run("ShouldDescribeSettingsCollectionsAndLogTail", func(t *testing.T) {
		t.Parallel()

		mcpList := operationFor(t, doc, "/api/settings/mcp-servers", "GET")
		assertParameter(t, mcpList, "scope", openapi3.ParameterInQuery, false)
		assertParameter(t, mcpList, "workspace_id", openapi3.ParameterInQuery, false)
		assertParameterEnumValues(t, mcpList, "scope", "global", "workspace")

		putMCP := operationFor(t, doc, "/api/settings/mcp-servers/{name}", "PUT")
		assertParameter(t, putMCP, "name", openapi3.ParameterInPath, true)
		assertParameter(t, putMCP, "scope", openapi3.ParameterInQuery, false)
		assertParameter(t, putMCP, "workspace_id", openapi3.ParameterInQuery, false)
		assertParameter(t, putMCP, "target", openapi3.ParameterInQuery, false)
		assertParameterEnumValues(t, putMCP, "scope", "global", "workspace")
		assertParameterEnumValues(t, putMCP, "target", "auto", "config", "sidecar")

		putMCPSchema := jsonRequestSchema(t, putMCP)
		assertRequired(t, putMCPSchema, "server")
		serverSchema := propertySchema(t, putMCPSchema, "server")
		assertRequired(t, serverSchema, "name")
		assertNotRequired(t, serverSchema, "transport", "command", "args", "env", "url", "auth")

		mcpListSchema := jsonResponseSchema(t, mcpList, 200)
		assertRequired(t, mcpListSchema, "collection", "scope", "available_scopes", "mcp_servers")
		assertNotRequired(t, mcpListSchema, "workspace_id")
		mcpItemRootSchema := propertySchema(t, mcpListSchema, "mcp_servers").Items.Value
		assertRequired(t, mcpItemRootSchema, "name", "transport", "scope", "source_metadata")
		assertNotRequired(t, mcpItemRootSchema, "command", "args", "env", "url", "auth", "auth_status")
		mcpItemSchema := propertySchema(t, mcpItemRootSchema, "source_metadata")
		assertRequired(t, mcpItemSchema, "effective_source", "available_targets")
		assertNotRequired(t, mcpItemSchema, "shadowed_sources")

		getObservability := operationFor(t, doc, "/api/settings/observability", "GET")
		observabilitySchema := jsonResponseSchema(t, getObservability, 200)
		assertRequired(t, observabilitySchema, "section", "scope", "available_scopes", "config", "runtime", "log_tail")
		logTailSchema := propertySchema(t, observabilitySchema, "log_tail")
		assertRequired(t, logTailSchema, "available")
		assertNotRequired(t, logTailSchema, "stream_url", "transport")
		assertEnumValues(t, propertySchema(t, logTailSchema, "transport"), "sse")

		logTail := operationFor(t, doc, "/api/settings/observability/log-tail", "GET")
		response := logTail.Responses.Status(200)
		if response == nil || response.Value == nil {
			t.Fatal("expected 200 response for log-tail stream")
		}
		if response.Value.Content != nil {
			t.Fatalf("log-tail 200 response content = %#v, want no JSON body for SSE route", response.Value.Content)
		}

		installExtension := operationFor(t, doc, "/api/extensions", "POST")
		if installExtension.Responses.Status(403) == nil {
			t.Fatal("expected 403 response on POST /api/extensions")
		}
	})
}

func assertOperationTransports(t *testing.T, operation *openapi3.Operation, want ...Transport) {
	t.Helper()

	raw, ok := operation.Extensions["x-agh-transports"]
	if !ok {
		t.Fatal("missing x-agh-transports extension")
	}

	got, ok := raw.([]Transport)
	if !ok {
		t.Fatalf("x-agh-transports = %#v, want []Transport", raw)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("x-agh-transports = %v, want %v", got, want)
	}
}

func assertParameterEnumValues(t *testing.T, operation *openapi3.Operation, name string, values ...string) {
	t.Helper()

	for _, ref := range operation.Parameters {
		if ref == nil || ref.Value == nil || ref.Value.Name != name {
			continue
		}
		schema := ref.Value.Schema
		if schema == nil || schema.Value == nil {
			t.Fatalf("parameter %q schema = nil", name)
		}
		assertEnumValues(t, schema.Value, values...)
		return
	}

	t.Fatalf("parameter %q not found", name)
}
