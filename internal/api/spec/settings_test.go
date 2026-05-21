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

	t.Run("Should register settings routes and HTTP extension parity", func(t *testing.T) {
		t.Parallel()

		operations := []struct {
			path       string
			method     string
			transports []Transport
		}{
			{path: "/api/settings/general", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: "/api/settings/update", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
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
			{path: "/api/settings/sandboxes", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       "/api/settings/sandboxes/{name}",
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/sandboxes/{name}",
				method:     "PUT",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       "/api/settings/sandboxes/{name}",
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
			{path: "/api/extensions/marketplace", method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: specAPIExtensionsNamePath, method: "GET", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: specAPIExtensionsNamePath, method: "PUT", transports: []Transport{TransportHTTP, TransportUDS}},
			{path: specAPIExtensionsNamePath, method: "DELETE", transports: []Transport{TransportHTTP, TransportUDS}},
			{
				path:       specAPIExtensionsNameProvenancePath,
				method:     "GET",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       specAPIExtensionsNameEnablePath,
				method:     "POST",
				transports: []Transport{TransportHTTP, TransportUDS},
			},
			{
				path:       specAPIExtensionsNameDisablePath,
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

	t.Run("Should describe settings mutation and restart schemas", func(t *testing.T) {
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
		assertRequired(
			t,
			mutationSchema,
			"active_config_hash",
			"active_generation",
			"applied",
			"apply_record_id",
			"lifecycle",
			"next_action",
		)
		assertNotRequired(
			t,
			mutationSchema,
			"section",
			"scope",
			"write_target",
			"workspace_id",
			"agent_name",
			"restart_required",
			"restart_scope",
			"warnings",
			"partial_failures",
			"skipped",
			"skipped_reason",
		)
		assertEnumValues(t, propertySchema(t, mutationSchema, "lifecycle"),
			"live",
			"live-add",
			"live-remove-if-unused",
			"restart-required",
			"session-rebind",
		)
		assertEnumValues(t, propertySchema(t, mutationSchema, "next_action"),
			"new-session",
			"none",
			"restart-daemon",
			"retry",
		)
		assertEnumValues(t, propertySchema(t, mutationSchema, "write_target"),
			"global-agent-file",
			"global-config",
			"global-mcp-sidecar",
			"workspace-agent-file",
			"workspace-config",
			"workspace-mcp-sidecar",
		)

		updateSkills := operationFor(t, doc, "/api/settings/skills", "PATCH")
		assertParameterEnumValues(t, updateSkills, "scope", "agent", "global")
		skillsMutationSchema := jsonResponseSchema(t, updateSkills, 200)
		assertRequired(
			t,
			skillsMutationSchema,
			"active_config_hash",
			"active_generation",
			"applied",
			"apply_record_id",
			"lifecycle",
			"next_action",
		)
		assertNotRequired(
			t,
			skillsMutationSchema,
			"section",
			"scope",
			"write_target",
			"workspace_id",
			"agent_name",
			"restart_required",
			"restart_scope",
			"warnings",
		)
		assertEnumValues(t, propertySchema(t, skillsMutationSchema, "lifecycle"),
			"live",
			"live-add",
			"live-remove-if-unused",
			"restart-required",
			"session-rebind",
		)

		reloadSettings := operationFor(t, doc, "/api/settings/reload", "POST")
		assertOperationTransports(t, reloadSettings, TransportHTTP, TransportUDS)
		reloadSchema := jsonResponseSchema(t, reloadSettings, 200)
		assertRequired(
			t,
			reloadSchema,
			"active_config_hash",
			"active_generation",
			"applied",
			"apply_record_id",
			"lifecycle",
			"next_action",
		)

		applyRecords := operationFor(t, doc, "/api/settings/apply", "GET")
		assertOperationTransports(t, applyRecords, TransportHTTP, TransportUDS)
		assertParameterEnumValues(t, applyRecords, "status", "applied", "blocked", "failed", "pending_apply")
		applyRecordsSchema := jsonResponseSchema(t, applyRecords, 200)
		assertRequired(t, applyRecordsSchema, "entries")
		applyRecordSchema := propertySchema(t, applyRecordsSchema, "entries").Items.Value
		assertRequired(
			t,
			applyRecordSchema,
			"id",
			"desired_config_hash",
			"active_config_hash",
			"generation",
			"actor",
			"diff_class",
			"status",
			"lifecycle",
			"next_action",
			"created_at",
			"updated_at",
		)
		assertEnumValues(t, propertySchema(t, applyRecordSchema, "status"),
			"applied",
			"blocked",
			"failed",
			"pending_apply",
		)

		readSkills := operationFor(t, doc, "/api/settings/skills", "GET")
		skillsResponseSchema := jsonResponseSchema(t, readSkills, 200)
		diagnosticsSchema := propertySchema(t, skillsResponseSchema, "diagnostics").Items.Value
		assertRequired(t, diagnosticsSchema, "name", "state", "verification_status")
		assertEnumValues(
			t,
			propertySchema(t, diagnosticsSchema, "state"),
			"shadowed",
			"valid",
			"verification_failed",
		)
		assertEnumValues(
			t,
			propertySchema(t, diagnosticsSchema, "verification_status"),
			"failed",
			"passed",
			"warning",
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

		updateStatus := operationFor(t, doc, "/api/settings/update", "GET")
		updateStatusSchema := jsonResponseSchema(t, updateStatus, 200)
		assertRequired(
			t,
			updateStatusSchema,
			"supported",
			"managed",
			"install_method",
			"current_version",
			"available",
			"status",
		)
		assertNotRequired(
			t,
			updateStatusSchema,
			"latest_version",
			"recommendation",
			"release_url",
			"checked_at",
			"last_error",
		)
		assertEnumValues(t, propertySchema(t, updateStatusSchema, "status"),
			"available",
			"current",
			"deferred",
			"failed",
			"unsupported",
			"updated",
		)
	})

	t.Run("Should describe settings collections and log tail", func(t *testing.T) {
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
		assertEnumValues(t, propertySchema(t, mcpListSchema, "scope"), "global", "workspace")
		mcpItemRootSchema := propertySchema(t, mcpListSchema, "mcp_servers").Items.Value
		assertRequired(t, mcpItemRootSchema, "name", "transport", "scope", "source_metadata")
		assertNotRequired(
			t,
			mcpItemRootSchema,
			"command",
			"args",
			"env",
			"url",
			"auth",
			"auth_status",
			"runtime_status",
		)
		mcpRuntimeSchema := propertySchema(t, mcpItemRootSchema, "runtime_status")
		assertRequired(t, mcpRuntimeSchema, "configured", "initialized", "state", "probe", "tool_count")
		assertNotRequired(t, mcpRuntimeSchema, "reason", "diagnostic")
		mcpItemSchema := propertySchema(t, mcpItemRootSchema, "source_metadata")
		assertRequired(t, mcpItemSchema, "effective_source", "available_targets")
		assertNotRequired(t, mcpItemSchema, "shadowed_sources")

		getObservability := operationFor(t, doc, "/api/settings/observability", "GET")
		observabilitySchema := jsonResponseSchema(t, getObservability, 200)
		assertRequired(t, observabilitySchema, "section", "scope", "available_scopes", "config", "runtime", "log_tail")
		assertNotRequired(t, observabilitySchema, "workspace_id", "agent_name")
		assertEnumValues(t, propertySchema(t, observabilitySchema, "scope"), "global")
		logTailSchema := propertySchema(t, observabilitySchema, "log_tail")
		assertRequired(t, logTailSchema, "available")
		assertNotRequired(t, logTailSchema, "stream_url", "transport")
		assertEnumValues(t, propertySchema(t, logTailSchema, "transport"), "sse")

		getProviders := operationFor(t, doc, "/api/settings/providers", "GET")
		providersSchema := jsonResponseSchema(t, getProviders, 200)
		assertRequired(t, providersSchema, "collection", "scope", "available_scopes", "providers")
		assertNotRequired(t, providersSchema, "workspace_id", "agent_name")
		assertEnumValues(t, propertySchema(t, providersSchema, "scope"), "global")
		providerItemSchema := propertySchema(t, providersSchema, "providers").Items.Value
		authStatusSchema := propertySchema(t, providerItemSchema, "auth_status")
		assertRequired(t, authStatusSchema, "mode", "env_policy", "home_policy", "state")
		assertNotRequired(
			t,
			authStatusSchema,
			"message",
			"status_command",
			"login_command",
			"login_env",
			"native_cli",
		)
		nativeCLISchema := propertySchema(t, authStatusSchema, "native_cli")
		assertRequired(t, nativeCLISchema, "present")
		assertNotRequired(t, nativeCLISchema, "command", "path", "source", "error")

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
