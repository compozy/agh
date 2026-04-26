package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/tools"
)

const (
	// DefaultPath is the canonical generated OpenAPI output location.
	DefaultPath = "openapi/agh.json"
)

var rawMessageType = reflect.TypeFor[json.RawMessage]()

var schemaEnumValues = map[reflect.Type][]string{
	reflect.TypeFor[automationpkg.Scope]():               automationScopeValues(),
	reflect.TypeFor[automationpkg.JobSource]():           automationSourceValues(),
	reflect.TypeFor[automationpkg.ScheduleMode]():        automationScheduleModeValues(),
	reflect.TypeFor[automationpkg.RetryStrategy]():       automationRetryStrategyValues(),
	reflect.TypeFor[automationpkg.RunStatus]():           automationRunStatusValues(),
	reflect.TypeFor[taskpkg.Scope]():                     taskScopeValues(),
	reflect.TypeFor[taskpkg.Status]():                    taskStatusValues(),
	reflect.TypeFor[taskpkg.Priority]():                  taskPriorityValues(),
	reflect.TypeFor[taskpkg.ApprovalPolicy]():            taskApprovalPolicyValues(),
	reflect.TypeFor[taskpkg.ApprovalState]():             taskApprovalStateValues(),
	reflect.TypeFor[taskpkg.RunStatus]():                 taskRunStatusValues(),
	reflect.TypeFor[taskpkg.ActorKind]():                 taskActorKindValues(),
	reflect.TypeFor[taskpkg.OwnerKind]():                 taskOwnerKindValues(),
	reflect.TypeFor[taskpkg.OriginKind]():                taskOriginKindValues(),
	reflect.TypeFor[taskpkg.DependencyKind]():            taskDependencyKindValues(),
	reflect.TypeFor[contract.TaskInboxLane]():            taskInboxLaneValues(),
	reflect.TypeFor[contract.CoordinationMessageKind]():  coordinationMessageKindValues(),
	reflect.TypeFor[contract.CoordinatorConfigSource]():  coordinatorConfigSourceValues(),
	reflect.TypeFor[hooks.HookEvent]():                   hookEventValues(),
	reflect.TypeFor[hooks.HookEventFamily]():             hookEventFamilyValues(),
	reflect.TypeFor[hooks.HookMode]():                    hookModeValues(),
	reflect.TypeFor[hooks.HookRunOutcome]():              hookOutcomeValues(),
	reflect.TypeFor[hooks.HookSkillSource]():             hookSkillSourceValues(),
	reflect.TypeFor[hooks.HookExecutorKind]():            hookExecutorKindValues(),
	reflect.TypeFor[hooks.HookSource]():                  hookSourceValues(),
	reflect.TypeFor[memory.Type]():                       memoryTypeValues(),
	reflect.TypeFor[memory.Scope]():                      memoryScopeValues(),
	reflect.TypeFor[contract.SettingsScopeKind]():        settingsScopeValues(),
	reflect.TypeFor[contract.SettingsSectionName]():      settingsSectionValues(),
	reflect.TypeFor[contract.SettingsCollectionName]():   settingsCollectionValues(),
	reflect.TypeFor[contract.SettingsWriteTargetKind]():  settingsWriteTargetValues(),
	reflect.TypeFor[contract.SettingsTargetSelector]():   settingsTargetSelectorValues(),
	reflect.TypeFor[contract.SettingsMutationBehavior](): settingsMutationBehaviorValues(),
	reflect.TypeFor[contract.SettingsPermissionMode]():   settingsPermissionModeValues(),
	reflect.TypeFor[contract.SettingsSourceKind]():       settingsSourceKindValues(),
	reflect.TypeFor[contract.RestartOperationStatus]():   restartOperationStatusValues(),
	reflect.TypeFor[contract.SettingsStreamTransport]():  settingsStreamTransportValues(),
	reflect.TypeFor[resources.ResourceScopeKind]():       resourceScopeKindValues(),
	reflect.TypeFor[bridgepkg.Scope]():                   bridgeScopeValues(),
	reflect.TypeFor[bridgepkg.BridgeInstanceSource]():    bridgeInstanceSourceValues(),
	reflect.TypeFor[bridgepkg.BridgeStatus]():            bridgeStatusValues(),
	reflect.TypeFor[bridgepkg.BridgeDMPolicy]():          bridgeDMPolicyValues(),
	reflect.TypeFor[bridgepkg.BridgeDegradationReason](): bridgeDegradationReasonValues(),
	reflect.TypeFor[bridgepkg.DeliveryMode]():            deliveryModeValues(),
	reflect.TypeFor[session.State]():                     sessionStateValues(),
	reflect.TypeFor[store.StopReason]():                  stopReasonValues(),
	reflect.TypeFor[tools.ToolSource]():                  toolSourceValues(),
	reflect.TypeFor[extensionprotocol.HostAPIMethod]():   hostAPIMethodValues(),
}

var schemaCustomizers = map[reflect.Type]func(*openapi3.Schema){
	rawMessageType: func(schema *openapi3.Schema) {
		*schema = *openapi3.NewSchema()
	},
	reflect.TypeFor[contract.BridgeProviderConfigPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeProviderConfigSchema()
	},
	reflect.TypeFor[contract.BridgeDeliveryDefaultsPayload](): func(schema *openapi3.Schema) {
		*schema = *bridgeDeliveryDefaultsSchema()
	},
}

// Transport identifies which daemon transport exposes a route.
type Transport string

const (
	TransportHTTP Transport = "http"
	TransportUDS  Transport = "uds"
)

// ParameterSpec describes one OpenAPI parameter.
type ParameterSpec struct {
	Name        string
	In          string
	Description string
	Required    bool
	Kind        string
	Format      string
	Enum        []string
}

// ResponseSpec describes one OpenAPI response.
type ResponseSpec struct {
	Status      int
	Description string
	Body        any
	ContentType string
}

// OperationSpec describes one canonical REST operation.
type OperationSpec struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Tags        []string
	Transports  []Transport
	Parameters  []ParameterSpec
	RequestBody any
	Responses   []ResponseSpec
}

// Document builds the canonical OpenAPI specification document.
func Document() (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "AGH API",
			Version: "1.0.0",
		},
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
		Paths: openapi3.NewPaths(),
		Tags: openapi3.Tags{
			{Name: "agent"},
			{Name: "agents"},
			{Name: "automation"},
			{Name: "bridges"},
			{Name: "daemon"},
			{Name: "network"},
			{Name: "extensions"},
			{Name: "hooks"},
			{Name: "memory"},
			{Name: "observe"},
			{Name: "resources"},
			{Name: "sessions"},
			{Name: "settings"},
			{Name: "skills"},
			{Name: "tasks"},
			{Name: "workspaces"},
		},
	}

	for _, opSpec := range Operations() {
		operation, err := buildOperation(doc.Components.Schemas, opSpec)
		if err != nil {
			return nil, fmt.Errorf("build %s %s: %w", opSpec.Method, opSpec.Path, err)
		}
		doc.AddOperation(opSpec.Path, opSpec.Method, operation)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi: %w", err)
	}

	return doc, nil
}

var operationRegistry = []OperationSpec{
	{
		Method:      "GET",
		Path:        "/api/resources",
		OperationID: "listResources",
		Summary:     "List desired-state resources on the local operator control plane",
		Tags:        []string{"resources"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("kind", "Filter by resource kind", false),
			enumQueryParam("scope_kind", "Filter by resource scope kind", resourceScopeKindValues()),
			queryParam("scope_id", "Filter by workspace scope id", false),
			queryParam("owner_kind", "Filter by stamped owner kind", false),
			queryParam("owner_id", "Filter by stamped owner id", false),
			queryParam("source_kind", "Filter by stamped source kind", false),
			queryParam("source_id", "Filter by stamped source id", false),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ResourcesResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid resource filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/resources/{kind}",
		OperationID: "listResourcesByKind",
		Summary:     "List one desired-state resource kind on the local operator control plane",
		Tags:        []string{"resources"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("kind", "Resource kind"),
			enumQueryParam("scope_kind", "Filter by resource scope kind", resourceScopeKindValues()),
			queryParam("scope_id", "Filter by workspace scope id", false),
			queryParam("owner_kind", "Filter by stamped owner kind", false),
			queryParam("owner_id", "Filter by stamped owner id", false),
			queryParam("source_kind", "Filter by stamped source kind", false),
			queryParam("source_id", "Filter by stamped source id", false),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ResourcesResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid resource filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/resources/{kind}/{id}",
		OperationID: "getResource",
		Summary:     "Read one desired-state resource on the local operator control plane",
		Tags:        []string{"resources"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("kind", "Resource kind"),
			pathParam("id", "Resource id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ResourceResponse{}},
			{Status: 404, Description: "Resource not found", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid resource identifier", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/resources/{kind}/{id}",
		OperationID: "putResource",
		Summary:     "Create or replace one desired-state resource on the local operator control plane",
		Tags:        []string{"resources"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("kind", "Resource kind"),
			pathParam("id", "Resource id"),
		},
		RequestBody: contract.PutResourceRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "Updated", Body: contract.ResourceResponse{}},
			{Status: 201, Description: "Created", Body: contract.ResourceResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflict", Body: contract.ErrorPayload{}},
			{Status: 413, Description: "Payload too large", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid resource payload", Body: contract.ErrorPayload{}},
			{Status: 429, Description: "Rate limited", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/resources/{kind}/{id}",
		OperationID: "deleteResource",
		Summary:     "Delete one desired-state resource on the local operator control plane",
		Tags:        []string{"resources"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("kind", "Resource kind"),
			pathParam("id", "Resource id"),
		},
		RequestBody: contract.DeleteResourceRequest{},
		Responses: []ResponseSpec{
			{Status: 204, Description: "Deleted"},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Resource not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid delete request", Body: contract.ErrorPayload{}},
			{Status: 429, Description: "Rate limited", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agents",
		OperationID: "listAgents",
		Summary:     "List all readable agent definitions",
		Tags:        []string{"agents"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentsResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agents/{name}",
		OperationID: "getAgent",
		Summary:     "Get one agent definition by name",
		Tags:        []string{"agents"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Agent name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentResponse{}},
			{Status: 404, Description: "Agent not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/jobs",
		OperationID: "listAutomationJobs",
		Summary:     "List automation jobs",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Filter by automation scope", automationScopeValues()),
			queryParam("workspace_id", "Filter by workspace id", false),
			enumQueryParam("source", "Filter by job source", automationSourceValues()),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.JobsResponse{}},
			{Status: 400, Description: "Invalid automation filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/automation/jobs",
		OperationID: "createAutomationJob",
		Summary:     "Create an automation job",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateJobRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.JobResponse{}},
			{Status: 400, Description: "Invalid automation job request", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Automation job conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/jobs/{id}",
		OperationID: "getAutomationJob",
		Summary:     "Get one automation job",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation job id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.JobResponse{}},
			{Status: 404, Description: "Automation job not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/automation/jobs/{id}",
		OperationID: "updateAutomationJob",
		Summary:     "Update one automation job",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation job id"),
		},
		RequestBody: contract.UpdateJobRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.JobResponse{}},
			{Status: 400, Description: "Invalid automation job update", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation job not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Automation job conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/automation/jobs/{id}",
		OperationID: "deleteAutomationJob",
		Summary:     "Delete one automation job",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation job id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 400, Description: "Invalid automation job delete request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation job not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/automation/jobs/{id}/trigger",
		OperationID: "triggerAutomationJob",
		Summary:     "Trigger one automation job immediately",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation job id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RunResponse{}},
			{Status: 404, Description: "Automation job not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Automation run conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/jobs/{id}/runs",
		OperationID: "listAutomationJobRuns",
		Summary:     "List run history for one automation job",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation job id"),
			enumQueryParam("status", "Filter by run status", automationRunStatusValues()),
			dateTimeQueryParam("since", "Only runs started since this timestamp"),
			dateTimeQueryParam("until", "Only runs started before this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RunsResponse{}},
			{Status: 400, Description: "Invalid automation run filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation job not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/triggers",
		OperationID: "listAutomationTriggers",
		Summary:     "List automation triggers",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Filter by automation scope", automationScopeValues()),
			queryParam("workspace_id", "Filter by workspace id", false),
			enumQueryParam("source", "Filter by trigger source", automationSourceValues()),
			queryParam("event", "Filter by trigger event", false),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TriggersResponse{}},
			{Status: 400, Description: "Invalid automation filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/automation/triggers",
		OperationID: "createAutomationTrigger",
		Summary:     "Create an automation trigger",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateTriggerRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.TriggerResponse{}},
			{Status: 400, Description: "Invalid automation trigger request", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Automation trigger conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/triggers/{id}",
		OperationID: "getAutomationTrigger",
		Summary:     "Get one automation trigger",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation trigger id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TriggerResponse{}},
			{Status: 404, Description: "Automation trigger not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/automation/triggers/{id}",
		OperationID: "updateAutomationTrigger",
		Summary:     "Update one automation trigger",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation trigger id"),
		},
		RequestBody: contract.UpdateTriggerRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TriggerResponse{}},
			{Status: 400, Description: "Invalid automation trigger update", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation trigger not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Automation trigger conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/automation/triggers/{id}",
		OperationID: "deleteAutomationTrigger",
		Summary:     "Delete one automation trigger",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation trigger id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 400, Description: "Invalid automation trigger delete request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation trigger not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/triggers/{id}/runs",
		OperationID: "listAutomationTriggerRuns",
		Summary:     "List run history for one automation trigger",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation trigger id"),
			enumQueryParam("status", "Filter by run status", automationRunStatusValues()),
			dateTimeQueryParam("since", "Only runs started since this timestamp"),
			dateTimeQueryParam("until", "Only runs started before this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RunsResponse{}},
			{Status: 400, Description: "Invalid automation run filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Automation trigger not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/runs",
		OperationID: "listAutomationRuns",
		Summary:     "List automation runs",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("job_id", "Filter by automation job id", false),
			queryParam("trigger_id", "Filter by automation trigger id", false),
			enumQueryParam("status", "Filter by run status", automationRunStatusValues()),
			dateTimeQueryParam("since", "Only runs started since this timestamp"),
			dateTimeQueryParam("until", "Only runs started before this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RunsResponse{}},
			{Status: 400, Description: "Invalid automation run filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/automation/runs/{id}",
		OperationID: "getAutomationRun",
		Summary:     "Get one automation run",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Automation run id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RunResponse{}},
			{Status: 404, Description: "Automation run not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/webhooks/global/{endpoint}",
		OperationID: "deliverGlobalWebhook",
		Summary:     "Deliver one global automation webhook",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP},
		Parameters: []ParameterSpec{
			pathParam("endpoint", "Webhook endpoint slug and id"),
			headerParam("X-AGH-Webhook-Timestamp", "Signed webhook timestamp"),
			headerParam("X-AGH-Webhook-Signature", "Signed webhook HMAC signature"),
		},
		RequestBody: map[string]any{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WebhookDeliveryResponse{}},
			{Status: 400, Description: "Invalid webhook request", Body: contract.ErrorPayload{}},
			{Status: 401, Description: "Webhook authentication failed", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Webhook trigger not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/webhooks/workspaces/{workspace_id}/{endpoint}",
		OperationID: "deliverWorkspaceWebhook",
		Summary:     "Deliver one workspace-scoped automation webhook",
		Tags:        []string{"automation"},
		Transports:  []Transport{TransportHTTP},
		Parameters: []ParameterSpec{
			pathParam("workspace_id", "Workspace id"),
			pathParam("endpoint", "Webhook endpoint slug and id"),
			headerParam("X-AGH-Webhook-Timestamp", "Signed webhook timestamp"),
			headerParam("X-AGH-Webhook-Signature", "Signed webhook HMAC signature"),
		},
		RequestBody: map[string]any{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WebhookDeliveryResponse{}},
			{Status: 400, Description: "Invalid webhook request", Body: contract.ErrorPayload{}},
			{Status: 401, Description: "Webhook authentication failed", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Webhook trigger not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Automation manager is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/bridges",
		OperationID: "listBridges",
		Summary:     "List persisted bridge instances",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgesResponse{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/bridges",
		OperationID: "createBridge",
		Summary:     "Create a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateBridgeRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.BridgeResponse{}},
			{Status: 400, Description: "Invalid bridge request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/bridges/providers",
		OperationID: "listBridgeProviders",
		Summary:     "List installed bridge-capable providers",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeProvidersResponse{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/bridges/{id}",
		OperationID: "getBridge",
		Summary:     "Get one bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/bridges/{id}",
		OperationID: "updateBridge",
		Summary:     "Update mutable bridge instance fields",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		RequestBody: contract.UpdateBridgeRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeResponse{}},
			{Status: 400, Description: "Invalid bridge update", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Bridge instance or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/bridges/{id}/enable",
		OperationID: "enableBridge",
		Summary:     "Enable a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Invalid bridge state transition", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/bridges/{id}/disable",
		OperationID: "disableBridge",
		Summary:     "Disable a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Invalid bridge state transition", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/bridges/{id}/restart",
		OperationID: "restartBridge",
		Summary:     "Restart a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Invalid bridge state transition", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/bridges/{id}/routes",
		OperationID: "listBridgeRoutes",
		Summary:     "List routes owned by a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeRoutesResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/bridges/{id}/secret-bindings",
		OperationID: "listBridgeSecretBindings",
		Summary:     "List persisted secret bindings for a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeSecretBindingsResponse{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/bridges/{id}/secret-bindings/{binding_name}",
		OperationID: "putBridgeSecretBinding",
		Summary:     "Create or update one bridge secret binding",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
			pathParam("binding_name", "Bridge provider secret slot name"),
		},
		RequestBody: contract.PutBridgeSecretBindingRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeSecretBindingResponse{}},
			{Status: 400, Description: "Invalid bridge secret binding request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Bridge secret binding conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/bridges/{id}/secret-bindings/{binding_name}",
		OperationID: "deleteBridgeSecretBinding",
		Summary:     "Delete one bridge secret binding",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
			pathParam("binding_name", "Bridge provider secret slot name"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{
				Status:      404,
				Description: "Bridge instance or secret binding not found",
				Body:        contract.ErrorPayload{},
			},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/bridges/{id}/test-delivery",
		OperationID: "testBridgeDelivery",
		Summary:     "Resolve a typed outbound delivery target for a bridge instance",
		Tags:        []string{"bridges"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Bridge instance id"),
		},
		RequestBody: contract.BridgeTestDeliveryRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.BridgeTestDeliveryResponse{}},
			{Status: 400, Description: "Invalid delivery target request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Bridge instance not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Bridge instance is unavailable", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Bridge service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/daemon/status",
		OperationID: "getDaemonStatus",
		Summary:     "Get the daemon status snapshot",
		Tags:        []string{"daemon"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.DaemonStatusResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/status",
		OperationID: "getNetworkStatus",
		Summary:     "Get the network runtime status snapshot",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkStatusResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/peers",
		OperationID: "listNetworkPeers",
		Summary:     "List visible network peers",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("channel", "Filter peers by channel", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkPeersResponse{}},
			{Status: 400, Description: "Invalid network filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/peers/{peer_id}",
		OperationID: "getNetworkPeer",
		Summary:     "Get one visible network peer detail",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("peer_id", "Network peer id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkPeerResponse{}},
			{Status: 404, Description: "Network peer not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/peers/{peer_id}/messages",
		OperationID: "listNetworkPeerMessages",
		Summary:     "List the directed room timeline for one network peer",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("peer_id", "Network peer id"),
			queryParam("before", "Return messages before the specified message id", false),
			queryParam("after", "Return messages after the specified message id", false),
			boolQueryParam("include_presence", "Include coalesced presence episodes in the timeline"),
			intQueryParam("limit", "Maximum number of timeline messages to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkPeerMessagesResponse{}},
			{Status: 400, Description: "Invalid peer timeline request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Network peer not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/channels",
		OperationID: "listNetworkChannels",
		Summary:     "List materialized network channels",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkChannelsResponse{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/network/channels",
		OperationID: "createNetworkChannel",
		Summary:     "Create a network channel by spawning agent sessions",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateNetworkChannelRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.CreateNetworkChannelResponse{}},
			{Status: 400, Description: "Invalid network channel request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/channels/{channel}",
		OperationID: "getNetworkChannel",
		Summary:     "Get one network channel detail",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("channel", "Network channel"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkChannelResponse{}},
			{Status: 400, Description: "Invalid network channel", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Network channel not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/channels/{channel}/messages",
		OperationID: "listNetworkChannelMessages",
		Summary:     "List the read-only timeline for one network channel",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("channel", "Network channel"),
			queryParam("before", "Return messages before the specified message id", false),
			queryParam("after", "Return messages after the specified message id", false),
			boolQueryParam("include_presence", "Include coalesced presence episodes in the timeline"),
			intQueryParam("limit", "Maximum number of timeline messages to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkChannelMessagesResponse{}},
			{Status: 400, Description: "Invalid network channel", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Network channel not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/network/send",
		OperationID: "sendNetworkMessage",
		Summary:     "Send one network message",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.NetworkSendRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkSendResponse{}},
			{Status: 400, Description: "Invalid network send request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Network target not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/network/inbox",
		OperationID: "listNetworkInbox",
		Summary:     "List queued network inbox messages for one local session",
		Tags:        []string{"network"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("session_id", "Target local session id", true),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.NetworkInboxResponse{}},
			{Status: 400, Description: "Invalid inbox request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Network target not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Network runtime is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/extensions",
		OperationID: "listExtensions",
		Summary:     "List installed extensions",
		Tags:        []string{"extensions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ExtensionsResponse{}},
			{Status: 503, Description: "Extension service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/extensions",
		OperationID: "installExtension",
		Summary:     "Install an extension by path and checksum",
		Tags:        []string{"extensions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.InstallExtensionRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.ExtensionResponse{}},
			{Status: 400, Description: "Invalid install request", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Extension service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/extensions/{name}",
		OperationID: "getExtension",
		Summary:     "Get one installed extension",
		Tags:        []string{"extensions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Extension name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ExtensionResponse{}},
			{Status: 404, Description: "Extension not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Extension service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/extensions/{name}/enable",
		OperationID: "enableExtension",
		Summary:     "Enable an installed extension",
		Tags:        []string{"extensions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Extension name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ExtensionResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Extension not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Extension service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/extensions/{name}/disable",
		OperationID: "disableExtension",
		Summary:     "Disable an installed extension",
		Tags:        []string{"extensions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Extension name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ExtensionResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Extension not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Extension service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/hooks/catalog",
		OperationID: "getHookCatalog",
		Summary:     "List the resolved hook catalog",
		Tags:        []string{"hooks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("workspace", "Workspace id or path", false),
			queryParam("agent", "Agent name", false),
			enumQueryParam("event", "Hook event name", hookEventValues()),
			enumQueryParam("source", "Hook source", hookSourceValues()),
			enumQueryParam("mode", "Hook mode", hookModeValues()),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HookCatalogResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/hooks/runs",
		OperationID: "getHookRuns",
		Summary:     "List hook run history for one session",
		Tags:        []string{"hooks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("session", "Session id", true),
			enumQueryParam("event", "Hook event name", hookEventValues()),
			enumQueryParam("outcome", "Hook execution outcome", hookOutcomeValues()),
			dateTimeQueryParam("since", "Only runs recorded since this timestamp"),
			intQueryParam("last", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HookRunsResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/hooks/events",
		OperationID: "getHookEvents",
		Summary:     "List supported hook taxonomy metadata",
		Tags:        []string{"hooks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("family", "Hook event family", hookEventFamilyValues()),
			boolQueryParam("sync_only", "Only return sync-eligible events"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HookEventsResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agent/me",
		OperationID: "getAgentMe",
		Summary:     "Resolve the calling agent session",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentMeResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Caller session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agent/context",
		OperationID: "getAgentContext",
		Summary:     "Return the bounded calling-agent situation context",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentContextResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Caller session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agent/channels",
		OperationID: "listAgentChannels",
		Summary:     "List coordination channels visible to the calling agent",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentChannelsResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agent/channels/{channel}/recv",
		OperationID: "receiveAgentChannelMessages",
		Summary:     "Receive task-bound coordination channel messages",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("channel", "Coordination channel id"),
			boolQueryParam("wait", "Wait for the next message when no messages are immediately available"),
			intQueryParam("limit", "Maximum number of messages to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentChannelMessagesResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Coordination channel not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid channel receive request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/channels/{channel}/send",
		OperationID: "sendAgentChannelMessage",
		Summary:     "Send one task-bound coordination channel message",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("channel", "Coordination channel id"),
		},
		RequestBody: contract.AgentChannelSendRequest{},
		Responses: []ResponseSpec{
			{Status: 202, Description: "Accepted", Body: contract.AgentChannelMessageResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Coordination channel not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid channel send request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/channels/reply",
		OperationID: "replyAgentChannelMessage",
		Summary:     "Reply to one delivered coordination channel message",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.AgentChannelReplyRequest{},
		Responses: []ResponseSpec{
			{Status: 202, Description: "Accepted", Body: contract.AgentChannelMessageResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Coordination message not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid channel reply request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/tasks/claim-next",
		OperationID: "claimNextAgentTask",
		Summary:     "Atomically claim the next matching task run for the calling agent",
		Tags:        []string{"agent", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.AgentTaskClaimNextRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentTaskClaimResponse{}},
			{Status: 204, Description: "No matching task run is currently claimable"},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run claim conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid claim criteria", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/tasks/{run_id}/heartbeat",
		OperationID: "heartbeatAgentTaskRun",
		Summary:     "Extend a claimed task-run lease for the calling agent",
		Tags:        []string{"agent", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("run_id", "Task run id"),
		},
		RequestBody: contract.AgentTaskHeartbeatRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentTaskLeaseResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run lease conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid heartbeat request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/tasks/{run_id}/complete",
		OperationID: "completeAgentTaskRun",
		Summary:     "Complete a claimed task run for the calling agent",
		Tags:        []string{"agent", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("run_id", "Task run id"),
		},
		RequestBody: contract.AgentTaskCompleteRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentTaskLeaseResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run completion conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid completion request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/tasks/{run_id}/fail",
		OperationID: "failAgentTaskRun",
		Summary:     "Fail a claimed task run for the calling agent",
		Tags:        []string{"agent", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("run_id", "Task run id"),
		},
		RequestBody: contract.AgentTaskFailRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentTaskLeaseResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run failure conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid failure request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/tasks/{run_id}/release",
		OperationID: "releaseAgentTaskRun",
		Summary:     "Release a claimed task run for the calling agent",
		Tags:        []string{"agent", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("run_id", "Task run id"),
		},
		RequestBody: contract.AgentTaskReleaseRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentTaskLeaseResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run release conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid release request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/agent/spawn",
		OperationID: "spawnAgentSession",
		Summary:     "Spawn a narrowed child session for the calling agent",
		Tags:        []string{"agent", "sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.AgentSpawnRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.AgentSpawnResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Spawn permission denied", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Spawn limit conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid spawn request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/agent/coordinator/config",
		OperationID: "getAgentCoordinatorConfig",
		Summary:     "Read resolved coordinator config for the calling agent workspace",
		Tags:        []string{"agent"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentCoordinatorConfigResponse{}},
			{Status: 401, Description: "Agent caller identity is missing", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/memory",
		OperationID: "listMemory",
		Summary:     "List memory document headers",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Memory scope", memoryScopeValues()),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: []memory.Header{}},
			{Status: 400, Description: "Invalid memory filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace or memory not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/memory/health",
		OperationID: "getMemoryHealth",
		Summary:     "Get memory health",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryHealthPayload{}},
			{Status: 400, Description: "Invalid memory health filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/memory/history",
		OperationID: "listMemoryHistory",
		Summary:     "List redacted memory operation history",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Memory scope", memoryScopeValues()),
			queryParam("workspace", "Workspace id or path", false),
			queryParam("operation", "Memory operation type", false),
			dateTimeQueryParam("since", "Only operations since this timestamp"),
			intQueryParam("limit", "Maximum number of operations to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryHistoryResponse{}},
			{Status: 400, Description: "Invalid memory history filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/memory/{filename}",
		OperationID: "readMemory",
		Summary:     "Read one memory document",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("filename", "Memory filename"),
			enumQueryParam("scope", "Memory scope", memoryScopeValues()),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryReadResponse{}},
			{Status: 400, Description: "Invalid memory reference", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Memory not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/memory/{filename}",
		OperationID: "writeMemory",
		Summary:     "Write one memory document",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("filename", "Memory filename"),
		},
		RequestBody: contract.MemoryWriteRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryMutationResponse{}},
			{Status: 400, Description: "Invalid memory write request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/memory/{filename}",
		OperationID: "deleteMemory",
		Summary:     "Delete one memory document",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("filename", "Memory filename"),
			enumQueryParam("scope", "Memory scope", memoryScopeValues()),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryMutationResponse{}},
			{Status: 400, Description: "Invalid memory reference", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Memory not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/memory/consolidate",
		OperationID: "consolidateMemory",
		Summary:     "Trigger dream consolidation",
		Tags:        []string{"memory"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.MemoryConsolidateRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MemoryConsolidateResponse{}},
			{Status: 400, Description: "Invalid consolidate request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/observe/events",
		OperationID: "listObserveEvents",
		Summary:     "List observability events",
		Tags:        []string{"observe"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("session_id", "Session id", false),
			queryParam("agent_name", "Agent name", false),
			queryParam("type", "Event type", false),
			dateTimeQueryParam("since", "Only events emitted since this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.ObserveEventsResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/observe/health",
		OperationID: "getObserveHealth",
		Summary:     "Get daemon health and memory health",
		Tags:        []string{"observe"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HealthResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/sessions",
		OperationID: "listSessions",
		Summary:     "List sessions",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionsResponse{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/sessions",
		OperationID: "createSession",
		Summary:     "Create a session",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateSessionRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.SessionResponse{}},
			{Status: 400, Description: "Invalid create request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Session creation conflict", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/sessions/{id}",
		OperationID: "getSession",
		Summary:     "Get one session snapshot",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionResponse{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/sessions/{id}",
		OperationID: "deleteSession",
		Summary:     "Delete one session and remove it from persisted history",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/sessions/{id}/stop",
		OperationID: "stopSession",
		Summary:     "Stop a session without deleting persisted history",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/sessions/{id}/resume",
		OperationID: "resumeSession",
		Summary:     "Resume a stopped session",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionResponse{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/sessions/{id}/events",
		OperationID: "listSessionEvents",
		Summary:     "List persisted session events",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
			dateTimeQueryParam("since", "Only events emitted since this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
			afterSequenceQueryParam("Only return events after this sequence number"),
			queryParam("type", "Event type", false),
			queryParam("agent_name", "Agent name", false),
			queryParam("turn_id", "Turn id", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionEventsResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/sessions/{id}/history",
		OperationID: "getSessionHistory",
		Summary:     "List grouped session turn history",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
			dateTimeQueryParam("since", "Only events emitted since this timestamp"),
			intQueryParam("limit", "Maximum number of records to return"),
			afterSequenceQueryParam("Only return events after this sequence number"),
			queryParam("type", "Event type", false),
			queryParam("agent_name", "Agent name", false),
			queryParam("turn_id", "Turn id", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionHistoryResponse{}},
			{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/sessions/{id}/transcript",
		OperationID: "getSessionTranscript",
		Summary:     "Get the canonical transcript for one session",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionTranscriptResponse{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/sessions/{id}/approve",
		OperationID: "approveSession",
		Summary:     "Approve or deny an interactive permission request",
		Tags:        []string{"sessions"},
		Transports:  []Transport{TransportHTTP},
		Parameters: []ParameterSpec{
			pathParam("id", "Session id"),
		},
		RequestBody: contract.ApproveSessionRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionApprovalResponse{}},
			{Status: 400, Description: "Invalid approval request", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Session not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks",
		OperationID: "listTasks",
		Summary:     "List enriched tasks",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Filter by task scope", taskScopeValues()),
			queryParam("workspace", "Filter by workspace path, name, or ID", false),
			enumQueryParam("status", "Filter by task status", taskStatusValues()),
			enumQueryParam("priority", "Filter by task priority", taskPriorityValues()),
			boolQueryParam("include_drafts", "Include draft tasks in list results"),
			enumQueryParam("approval_state", "Filter by task approval state", taskApprovalStateValues()),
			enumQueryParam("owner_kind", "Filter by owner kind", taskOwnerKindValues()),
			queryParam("owner_ref", "Filter by owner reference", false),
			queryParam("parent_task_id", "Filter by parent task ID", false),
			queryParam("network_channel", "Filter by network channel", false),
			queryParam("query", "Filter by task title or identifier", false),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TasksResponse{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks",
		OperationID: "createTask",
		Summary:     "Create a task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateTaskRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task conflict", Body: contract.ErrorPayload{}},
			{Status: 413, Description: "Payload too large", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks/{id}",
		OperationID: "getTask",
		Summary:     "Get one task with detail",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskDetailResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task id", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/tasks/{id}",
		OperationID: "deleteTask",
		Summary:     "Delete one task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 400, Description: "Invalid task delete", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/tasks/{id}",
		OperationID: "updateTask",
		Summary:     "Update one task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		RequestBody: contract.UpdateTaskRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task update conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task update", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/publish",
		OperationID: "publishTask",
		Summary:     "Publish one draft task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task publish conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task publish request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/cancel",
		OperationID: "cancelTask",
		Summary:     "Cancel one task tree",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		RequestBody: contract.CancelTaskRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task cancel conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task cancel request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/children",
		OperationID: "createChildTask",
		Summary:     "Create one child task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Parent task id"),
		},
		RequestBody: contract.CreateTaskChildRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Child task conflict", Body: contract.ErrorPayload{}},
			{Status: 413, Description: "Payload too large", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid child task request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/dependencies",
		OperationID: "addTaskDependency",
		Summary:     "Add one task dependency",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		RequestBody: contract.AddTaskDependencyRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskDetailResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Dependency conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid dependency request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/tasks/{id}/dependencies/{depends_on_id}",
		OperationID: "removeTaskDependency",
		Summary:     "Remove one task dependency",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
			pathParam("depends_on_id", "Dependency task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskDetailResponse{}},
			{Status: 404, Description: "Task or dependency not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid dependency request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks/{id}/runs",
		OperationID: "listTaskRuns",
		Summary:     "List runs for one task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
			enumQueryParam("status", "Filter by run status", taskRunStatusValues()),
			queryParam("session_id", "Filter by attached session id", false),
			intQueryParam("limit", "Maximum number of records to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunsResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run filter", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/runs",
		OperationID: "enqueueTaskRun",
		Summary:     "Enqueue one task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		RequestBody: contract.EnqueueTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run enqueue conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run enqueue request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/task-runs/{id}",
		OperationID: "getTaskRun",
		Summary:     "Get one task run detail",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunDetailResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run id", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/claim",
		OperationID: "claimTaskRun",
		Summary:     "Claim one queued task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.ClaimTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run claim conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run claim request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/start",
		OperationID: "startTaskRun",
		Summary:     "Start one claimed task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.StartTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run start conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run start request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/attach-session",
		OperationID: "attachTaskRunSession",
		Summary:     "Attach an existing session to one task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.AttachTaskRunSessionRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run or session not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Attach-session conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid attach-session request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/complete",
		OperationID: "completeTaskRun",
		Summary:     "Complete one running task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.CompleteTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run completion conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run completion request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/fail",
		OperationID: "failTaskRun",
		Summary:     "Fail one task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.FailTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run failure conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run failure request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/task-runs/{id}/cancel",
		OperationID: "cancelTaskRun",
		Summary:     "Cancel one task run",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task run id"),
		},
		RequestBody: contract.CancelTaskRunRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskRunResponse{}},
			{Status: 404, Description: "Task run not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task-run cancel conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task-run cancel request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks/{id}/timeline",
		OperationID: "getTaskTimeline",
		Summary:     "Get one task timeline",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
			afterSequenceQueryParam("Return only events after the supplied sequence"),
			intQueryParam("limit", "Maximum number of timeline items to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskTimelineResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid timeline query", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks/{id}/stream",
		OperationID: "streamTask",
		Summary:     "Stream task-native live events for one task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
			afterSequenceQueryParam("Replay events after the supplied task stream sequence"),
		},
		Responses: []ResponseSpec{
			{
				Status:      200,
				Description: "Task event stream",
				Body:        contract.TaskStreamEventPayload{},
				ContentType: "text/event-stream",
			},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task stream query", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/tasks/{id}/tree",
		OperationID: "getTaskTree",
		Summary:     "Get one task tree live view",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskTreeResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task id", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/approve",
		OperationID: "approveTask",
		Summary:     "Approve one approval-gated task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task approval conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task approval request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/reject",
		OperationID: "rejectTask",
		Summary:     "Reject one approval-gated task",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task rejection conflict", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task rejection request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/triage/read",
		OperationID: "markTaskRead",
		Summary:     "Mark one task inbox item as read",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskTriageStateResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task triage request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/triage/archive",
		OperationID: "archiveTask",
		Summary:     "Archive one task inbox item",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskTriageStateResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task triage request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/tasks/{id}/triage/dismiss",
		OperationID: "dismissTask",
		Summary:     "Dismiss one task inbox item",
		Tags:        []string{"tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Task id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskTriageStateResponse{}},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task triage request", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/observe/tasks/dashboard",
		OperationID: "getTaskDashboard",
		Summary:     "Get the observer-backed task dashboard",
		Tags:        []string{"observe", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Filter by task scope", taskScopeValues()),
			queryParam("workspace", "Filter by workspace path, name, or ID", false),
			enumQueryParam("owner_kind", "Filter by owner kind", taskOwnerKindValues()),
			queryParam("owner_ref", "Filter by owner reference", false),
			queryParam("network_channel", "Filter by network channel", false),
			enumQueryParam("origin_kind", "Filter by task origin kind", taskOriginKindValues()),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskDashboardResponse{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task dashboard query", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Observe service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/observe/tasks/inbox",
		OperationID: "getTaskInbox",
		Summary:     "Get the observer-backed task inbox",
		Tags:        []string{"observe", "tasks"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Filter by task scope", taskScopeValues()),
			queryParam("workspace", "Filter by workspace path, name, or ID", false),
			enumQueryParam("owner_kind", "Filter by owner kind", taskOwnerKindValues()),
			queryParam("owner_ref", "Filter by owner reference", false),
			enumQueryParam("lane", "Filter by inbox lane", taskInboxLaneValues()),
			boolQueryParam("unread", "Return only unread inbox items"),
			queryParam("query", "Filter by task title or identifier", false),
			intQueryParam("limit", "Maximum number of inbox items to return"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.TaskInboxResponse{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid task inbox query", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Observe service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/skills",
		OperationID: "listSkills",
		Summary:     "List skills for one workspace",
		Tags:        []string{"skills"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			queryParam("workspace", "Workspace id or path", true),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SkillsResponse{}},
			{Status: 400, Description: "Invalid workspace filter", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Skills registry is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/skills/{name}",
		OperationID: "getSkill",
		Summary:     "Get one skill definition",
		Tags:        []string{"skills"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Skill name"),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SkillResponse{}},
			{Status: 400, Description: "Invalid skill lookup", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Skill or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Skills registry is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/skills/{name}/content",
		OperationID: "getSkillContent",
		Summary:     "Get the raw content for one skill",
		Tags:        []string{"skills"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Skill name"),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SkillContentResponse{}},
			{Status: 400, Description: "Invalid skill lookup", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Skill or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Skills registry is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/skills/{name}/enable",
		OperationID: "enableSkill",
		Summary:     "Enable one skill",
		Tags:        []string{"skills"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Skill name"),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SkillActionResponse{}},
			{Status: 400, Description: "Invalid skill lookup", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Skill or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Skills registry is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/skills/{name}/disable",
		OperationID: "disableSkill",
		Summary:     "Disable one skill",
		Tags:        []string{"skills"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Skill name"),
			queryParam("workspace", "Workspace id or path", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SkillActionResponse{}},
			{Status: 400, Description: "Invalid skill lookup", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Skill or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Skills registry is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/actions/restart/{operation_id}",
		OperationID: "getSettingsRestartStatus",
		Summary:     "Get the persisted status for one daemon restart operation",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("operation_id", "Restart operation id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.RestartActionStatus{}},
			{Status: 404, Description: "Restart operation not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/settings/actions/restart",
		OperationID: "triggerSettingsRestart",
		Summary:     "Trigger a daemon restart using the persisted relaunch helper flow",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 202, Description: "Accepted", Body: contract.RestartActionResponse{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/automation",
		OperationID: "getSettingsAutomation",
		Summary:     "Read the automation settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsAutomationResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/automation",
		OperationID: "updateSettingsAutomation",
		Summary:     "Update the automation settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsAutomationRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/environments",
		OperationID: "listSettingsEnvironments",
		Summary:     "List settings-backed execution environments",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsEnvironmentsResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/environments/{name}",
		OperationID: "getSettingsEnvironment",
		Summary:     "Read one settings-backed execution environment",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Environment name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsEnvironmentResponse{}},
			{Status: 404, Description: "Environment not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/settings/environments/{name}",
		OperationID: "putSettingsEnvironment",
		Summary:     "Create or replace one settings-backed execution environment",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Environment name"),
		},
		RequestBody: contract.PutSettingsEnvironmentRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid environment payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting environment change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/settings/environments/{name}",
		OperationID: "deleteSettingsEnvironment",
		Summary:     "Delete one settings-backed execution environment overlay",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Environment name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Environment not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/general",
		OperationID: "getSettingsGeneral",
		Summary:     "Read the general settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsGeneralResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/general",
		OperationID: "updateSettingsGeneral",
		Summary:     "Update the general settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsGeneralRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/hooks",
		OperationID: "listSettingsHooks",
		Summary:     "List settings-backed hook declarations",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsHooksResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/settings/hooks/{name}",
		OperationID: "putSettingsHook",
		Summary:     "Create or replace one settings-backed hook declaration",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Hook name"),
		},
		RequestBody: contract.PutSettingsHookRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid hook payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting hook change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/settings/hooks/{name}",
		OperationID: "deleteSettingsHook",
		Summary:     "Delete one settings-backed hook declaration",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Hook name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Hook not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/hooks-extensions",
		OperationID: "getSettingsHooksExtensions",
		Summary:     "Read the hooks and extensions settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsHooksExtensionsResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/hooks-extensions",
		OperationID: "updateSettingsHooksExtensions",
		Summary:     "Update the hooks and extensions settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsHooksExtensionsRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/mcp-servers",
		OperationID: "listSettingsMCPServers",
		Summary:     "List settings-backed MCP servers",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			enumQueryParam("scope", "Select the settings scope", settingsScopeValues()),
			queryParam("workspace_id", "Select the workspace id for workspace scope", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsMCPServersResponse{}},
			{Status: 400, Description: "Invalid settings scope", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/settings/mcp-servers/{name}",
		OperationID: "putSettingsMCPServer",
		Summary:     "Create or replace one settings-backed MCP server",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "MCP server name"),
			enumQueryParam("scope", "Select the settings scope", settingsScopeValues()),
			queryParam("workspace_id", "Select the workspace id for workspace scope", false),
			enumQueryParam("target", "Select the persistence target", settingsTargetSelectorValues()),
		},
		RequestBody: contract.PutSettingsMCPServerRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid MCP server payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting MCP server change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/settings/mcp-servers/{name}",
		OperationID: "deleteSettingsMCPServer",
		Summary:     "Delete one settings-backed MCP server",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "MCP server name"),
			enumQueryParam("scope", "Select the settings scope", settingsScopeValues()),
			queryParam("workspace_id", "Select the workspace id for workspace scope", false),
			enumQueryParam("target", "Select the persistence target", settingsTargetSelectorValues()),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid MCP server request", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "MCP server or workspace not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting MCP server change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/memory",
		OperationID: "getSettingsMemory",
		Summary:     "Read the memory settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsMemoryResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/memory",
		OperationID: "updateSettingsMemory",
		Summary:     "Update the memory settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsMemoryRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/network",
		OperationID: "getSettingsNetwork",
		Summary:     "Read the network settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsNetworkResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/network",
		OperationID: "updateSettingsNetwork",
		Summary:     "Update the network settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsNetworkRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/observability",
		OperationID: "getSettingsObservability",
		Summary:     "Read the observability settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsObservabilityResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/observability",
		OperationID: "updateSettingsObservability",
		Summary:     "Update the observability settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsObservabilityRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/observability/log-tail",
		OperationID: "streamSettingsObservabilityLogTail",
		Summary:     "Stream daemon log output for the observability settings screen",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "SSE stream established"},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/providers",
		OperationID: "listSettingsProviders",
		Summary:     "List settings-backed providers",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsProvidersResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/providers/{name}",
		OperationID: "getSettingsProvider",
		Summary:     "Read one settings-backed provider",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Provider name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsProviderResponse{}},
			{Status: 404, Description: "Provider not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PUT",
		Path:        "/api/settings/providers/{name}",
		OperationID: "putSettingsProvider",
		Summary:     "Create or replace one settings-backed provider overlay",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Provider name"),
		},
		RequestBody: contract.PutSettingsProviderRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid provider payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting provider change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/settings/providers/{name}",
		OperationID: "deleteSettingsProvider",
		Summary:     "Delete one settings-backed provider overlay",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("name", "Provider name"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Provider not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/settings/skills",
		OperationID: "getSettingsSkills",
		Summary:     "Read the skills settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SettingsSkillsResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/settings/skills",
		OperationID: "updateSettingsSkills",
		Summary:     "Update the skills settings section",
		Tags:        []string{"settings"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.UpdateSettingsSkillsRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.MutationResult{}},
			{Status: 400, Description: "Invalid settings payload", Body: contract.ErrorPayload{}},
			{Status: 403, Description: "Forbidden", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Conflicting settings change", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/workspaces",
		OperationID: "listWorkspaces",
		Summary:     "List registered workspaces",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WorkspacesResponse{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/workspaces",
		OperationID: "createWorkspace",
		Summary:     "Register a workspace",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.CreateWorkspaceRequest{},
		Responses: []ResponseSpec{
			{Status: 201, Description: "Created", Body: contract.WorkspaceResponse{}},
			{Status: 400, Description: "Invalid workspace request", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Workspace conflict", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "GET",
		Path:        "/api/workspaces/{id}",
		OperationID: "getWorkspace",
		Summary:     "Get one resolved workspace with related data",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Workspace id or path"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WorkspaceDetailPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "PATCH",
		Path:        "/api/workspaces/{id}",
		OperationID: "updateWorkspace",
		Summary:     "Update a registered workspace",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Workspace id"),
		},
		RequestBody: contract.UpdateWorkspaceRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WorkspaceResponse{}},
			{Status: 400, Description: "Invalid workspace update", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "DELETE",
		Path:        "/api/workspaces/{id}",
		OperationID: "deleteWorkspace",
		Summary:     "Delete a registered workspace",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("id", "Workspace id"),
		},
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      "POST",
		Path:        "/api/workspaces/resolve",
		OperationID: "resolveWorkspace",
		Summary:     "Resolve or register a workspace from a path",
		Tags:        []string{"workspaces"},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.ResolveWorkspaceRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.WorkspaceResponse{}},
			{Status: 400, Description: "Invalid workspace path", Body: contract.ErrorPayload{}},
			{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
		},
	},
}

// Operations returns the canonical REST operation registry in deterministic order.
func Operations() []OperationSpec {
	ops := append([]OperationSpec(nil), operationRegistry...)
	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].Path == ops[j].Path {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})

	return ops
}

// WriteFile renders the canonical OpenAPI document to the supplied path.
func WriteFile(path string) error {
	doc, err := Document()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal openapi: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func buildOperation(schemas openapi3.Schemas, spec OperationSpec) (*openapi3.Operation, error) {
	operation := openapi3.NewOperation()
	operation.OperationID = spec.OperationID
	operation.Summary = spec.Summary
	operation.Tags = append([]string(nil), spec.Tags...)
	operation.Extensions = map[string]any{
		"x-agh-transports": spec.Transports,
	}

	for _, param := range spec.Parameters {
		operation.AddParameter(buildParameter(param))
	}

	if spec.RequestBody != nil {
		schemaRef, err := schemaRefForValue(spec.RequestBody, schemas)
		if err != nil {
			return nil, fmt.Errorf("request body schema: %w", err)
		}
		operation.RequestBody = &openapi3.RequestBodyRef{
			Value: openapi3.NewRequestBody().
				WithContent(openapi3.NewContentWithJSONSchemaRef(schemaRef)).
				WithDescription("JSON request body"),
		}
		operation.RequestBody.Value.Required = true
	}

	for _, response := range spec.Responses {
		resp := openapi3.NewResponse().WithDescription(response.Description)
		if response.Body != nil {
			schemaRef, err := schemaRefForValue(response.Body, schemas)
			if err != nil {
				return nil, fmt.Errorf("response %d schema: %w", response.Status, err)
			}
			contentType := response.ContentType
			if contentType == "" {
				contentType = "application/json"
			}
			resp.WithContent(openapi3.Content{
				contentType: &openapi3.MediaType{Schema: schemaRef},
			})
		}
		operation.AddResponse(response.Status, resp)
	}

	return operation, nil
}

func resourceScopeKindValues() []string {
	return []string{
		string(resources.ResourceScopeKindGlobal),
		string(resources.ResourceScopeKindWorkspace),
	}
}

func settingsScopeValues() []string {
	return []string{
		string(contract.SettingsScopeGlobal),
		string(contract.SettingsScopeWorkspace),
	}
}

func settingsSectionValues() []string {
	return []string{
		string(contract.SettingsSectionGeneral),
		string(contract.SettingsSectionMemory),
		string(contract.SettingsSectionSkills),
		string(contract.SettingsSectionAutomation),
		string(contract.SettingsSectionNetwork),
		string(contract.SettingsSectionObservability),
		string(contract.SettingsSectionHooksExtensions),
	}
}

func settingsCollectionValues() []string {
	return []string{
		string(contract.SettingsCollectionProviders),
		string(contract.SettingsCollectionMCPServers),
		string(contract.SettingsCollectionEnvironments),
		string(contract.SettingsCollectionHooks),
	}
}

func settingsWriteTargetValues() []string {
	return []string{
		string(contract.SettingsWriteTargetGlobalConfig),
		string(contract.SettingsWriteTargetWorkspaceConfig),
		string(contract.SettingsWriteTargetGlobalMCPSidecar),
		string(contract.SettingsWriteTargetWorkspaceMCPSidecar),
	}
}

func settingsTargetSelectorValues() []string {
	return []string{
		string(contract.SettingsTargetAuto),
		string(contract.SettingsTargetConfig),
		string(contract.SettingsTargetSidecar),
	}
}

func settingsMutationBehaviorValues() []string {
	return []string{
		string(contract.SettingsMutationBehaviorAppliedNow),
		string(contract.SettingsMutationBehaviorRestartRequired),
		string(contract.SettingsMutationBehaviorActionTrigger),
	}
}

func settingsPermissionModeValues() []string {
	return []string{
		string(contract.SettingsPermissionModeDenyAll),
		string(contract.SettingsPermissionModeApproveReads),
		string(contract.SettingsPermissionModeApproveAll),
	}
}

func settingsSourceKindValues() []string {
	return []string{
		string(contract.SettingsSourceBuiltinProvider),
		string(contract.SettingsSourceGlobalConfig),
		string(contract.SettingsSourceWorkspaceConfig),
		string(contract.SettingsSourceGlobalMCPSidecar),
		string(contract.SettingsSourceWorkspaceMCPSidecar),
	}
}

func restartOperationStatusValues() []string {
	return []string{
		string(contract.RestartOperationPending),
		string(contract.RestartOperationStopping),
		string(contract.RestartOperationWaitingRelease),
		string(contract.RestartOperationStarting),
		string(contract.RestartOperationReady),
		string(contract.RestartOperationFailed),
	}
}

func settingsStreamTransportValues() []string {
	return []string{
		string(contract.SettingsStreamTransportSSE),
	}
}

func schemaRefForValue(value any, schemas openapi3.Schemas) (*openapi3.SchemaRef, error) {
	var rootType reflect.Type
	if value != nil {
		rootType = reflect.TypeOf(value)
		switch rootType.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array:
			value = reflect.New(rootType).Interface()
		}
	}
	schemaRef, err := openapi3gen.NewSchemaRefForValue(value, schemas, openapi3gen.SchemaCustomizer(schemaCustomizer))
	if err != nil {
		return nil, err
	}
	applySchemaRequirements(schemaRef, rootType)
	return schemaRef, nil
}

func buildParameter(spec ParameterSpec) *openapi3.Parameter {
	var param *openapi3.Parameter
	switch spec.In {
	case openapi3.ParameterInPath:
		param = openapi3.NewPathParameter(spec.Name)
	case openapi3.ParameterInHeader:
		param = &openapi3.Parameter{Name: spec.Name, In: openapi3.ParameterInHeader}
	default:
		param = openapi3.NewQueryParameter(spec.Name)
	}
	param.WithRequired(spec.Required)
	if spec.Description != "" {
		param.WithDescription(spec.Description)
	}
	param.Schema = schemaRefForParameter(spec)
	return param
}

func schemaRefForParameter(spec ParameterSpec) *openapi3.SchemaRef {
	var schema *openapi3.Schema
	switch spec.Kind {
	case "boolean":
		schema = openapi3.NewBoolSchema()
	case "integer":
		schema = openapi3.NewIntegerSchema()
		if spec.Format != "" {
			schema.Format = spec.Format
		}
	default:
		schema = openapi3.NewStringSchema()
		if spec.Format != "" {
			schema.Format = spec.Format
		}
	}
	if len(spec.Enum) > 0 {
		schema.Enum = make([]any, 0, len(spec.Enum))
		for _, value := range spec.Enum {
			schema.Enum = append(schema.Enum, value)
		}
	}
	return openapi3.NewSchemaRef("", schema)
}

func schemaCustomizer(_ string, t reflect.Type, _ reflect.StructTag, schema *openapi3.Schema) error {
	if customizer, ok := schemaCustomizers[t]; ok {
		customizer(schema)
		return nil
	}
	if values, ok := schemaEnumValues[t]; ok {
		setStringEnum(schema, values)
	}
	return nil
}

func applySchemaRequirements(schemaRef *openapi3.SchemaRef, t reflect.Type) {
	if schemaRef == nil || schemaRef.Value == nil || t == nil {
		return
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Array, reflect.Slice:
		applySchemaRequirements(schemaRef.Value.Items, t.Elem())
	case reflect.Map:
		if schemaRef.Value.AdditionalProperties.Schema != nil {
			applySchemaRequirements(schemaRef.Value.AdditionalProperties.Schema, t.Elem())
		}
	case reflect.Struct:
		applyStructRequirements(schemaRef.Value, t)
	}
}

func applyStructRequirements(schema *openapi3.Schema, t reflect.Type) {
	if schema == nil || t.Kind() != reflect.Struct {
		return
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return
	}
	if schema.Properties == nil {
		return
	}

	required := make(map[string]struct{}, len(schema.Properties))
	collectStructRequirements(schema, t, required)
	if len(required) == 0 {
		schema.Required = nil
		return
	}

	schema.Required = schema.Required[:0]
	for name := range required {
		schema.Required = append(schema.Required, name)
	}
	sort.Strings(schema.Required)
}

func collectStructRequirements(schema *openapi3.Schema, t reflect.Type, required map[string]struct{}) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() && !field.Anonymous {
			continue
		}

		jsonName, omitEmpty, skip := parseJSONField(field)
		if skip {
			continue
		}

		if field.Anonymous && field.Tag.Get("json") == "" {
			collectStructRequirements(schema, field.Type, required)
			continue
		}

		propertyRef, ok := schema.Properties[jsonName]
		if !ok {
			continue
		}

		if !omitEmpty {
			required[jsonName] = struct{}{}
		}
		applySchemaRequirements(propertyRef, field.Type)
	}
}

func parseJSONField(field reflect.StructField) (name string, omitEmpty bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}

	if tag == "" {
		return field.Name, false, false
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		name = parts[0]
	} else {
		name = field.Name
	}
	if slices.Contains(parts[1:], "omitempty") {
		omitEmpty = true
	}
	return name, omitEmpty, false
}

func setStringEnum(schema *openapi3.Schema, values []string) {
	*schema = *openapi3.NewStringSchema()
	schema.Enum = make([]any, 0, len(values))
	for _, value := range values {
		schema.Enum = append(schema.Enum, value)
	}
}

func enumAsAny(values []string) []any {
	converted := make([]any, 0, len(values))
	for _, value := range values {
		converted = append(converted, value)
	}
	return converted
}

func pathParam(name string, description string) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInPath, Description: description, Required: true}
}

func headerParam(name string, description string) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInHeader, Description: description, Required: true}
}

func queryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInQuery, Description: description, Required: required}
}

func enumQueryParam(name string, description string, values []string) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    false,
		Enum:        values,
	}
}

func boolQueryParam(name string, description string) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    false,
		Kind:        "boolean",
	}
}

func intQueryParam(name string, description string) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    false,
		Kind:        "integer",
		Format:      "int32",
	}
}

func afterSequenceQueryParam(description string) ParameterSpec {
	return ParameterSpec{
		Name:        "after_sequence",
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    false,
		Kind:        "integer",
		Format:      "int64",
	}
}

func dateTimeQueryParam(name string, description string) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    false,
		Format:      "date-time",
	}
}

func automationScopeValues() []string {
	return []string{
		string(automationpkg.AutomationScopeGlobal),
		string(automationpkg.AutomationScopeWorkspace),
	}
}

func automationSourceValues() []string {
	return []string{
		string(automationpkg.JobSourceConfig),
		string(automationpkg.JobSourceDynamic),
	}
}

func automationScheduleModeValues() []string {
	return []string{
		string(automationpkg.ScheduleModeCron),
		string(automationpkg.ScheduleModeEvery),
		string(automationpkg.ScheduleModeAt),
	}
}

func automationRetryStrategyValues() []string {
	return []string{
		string(automationpkg.RetryStrategyNone),
		string(automationpkg.RetryStrategyBackoff),
	}
}

func automationRunStatusValues() []string {
	return []string{
		string(automationpkg.RunScheduled),
		string(automationpkg.RunRunning),
		string(automationpkg.RunDelegated),
		string(automationpkg.RunCompleted),
		string(automationpkg.RunFailed),
		string(automationpkg.RunCancelled),
	}
}

func taskScopeValues() []string {
	return []string{
		string(taskpkg.ScopeGlobal),
		string(taskpkg.ScopeWorkspace),
	}
}

func taskStatusValues() []string {
	return []string{
		string(taskpkg.TaskStatusDraft),
		string(taskpkg.TaskStatusPending),
		string(taskpkg.TaskStatusBlocked),
		string(taskpkg.TaskStatusReady),
		string(taskpkg.TaskStatusInProgress),
		string(taskpkg.TaskStatusCompleted),
		string(taskpkg.TaskStatusFailed),
		string(taskpkg.TaskStatusCanceled),
	}
}

func taskPriorityValues() []string {
	return []string{
		string(taskpkg.PriorityLow),
		string(taskpkg.PriorityMedium),
		string(taskpkg.PriorityHigh),
		string(taskpkg.PriorityUrgent),
	}
}

func taskApprovalPolicyValues() []string {
	return []string{
		string(taskpkg.ApprovalPolicyNone),
		string(taskpkg.ApprovalPolicyManual),
	}
}

func taskApprovalStateValues() []string {
	return []string{
		string(taskpkg.ApprovalStateNotRequired),
		string(taskpkg.ApprovalStatePending),
		string(taskpkg.ApprovalStateApproved),
		string(taskpkg.ApprovalStateRejected),
	}
}

func taskRunStatusValues() []string {
	return []string{
		string(taskpkg.TaskRunStatusQueued),
		string(taskpkg.TaskRunStatusClaimed),
		string(taskpkg.TaskRunStatusStarting),
		string(taskpkg.TaskRunStatusRunning),
		string(taskpkg.TaskRunStatusCompleted),
		string(taskpkg.TaskRunStatusFailed),
		string(taskpkg.TaskRunStatusCanceled),
	}
}

func taskActorKindValues() []string {
	return []string{
		string(taskpkg.ActorKindHuman),
		string(taskpkg.ActorKindAgentSession),
		string(taskpkg.ActorKindAutomation),
		string(taskpkg.ActorKindExtension),
		string(taskpkg.ActorKindNetworkPeer),
		string(taskpkg.ActorKindDaemon),
	}
}

func taskOwnerKindValues() []string {
	return []string{
		string(taskpkg.OwnerKindHuman),
		string(taskpkg.OwnerKindAgentSession),
		string(taskpkg.OwnerKindAutomation),
		string(taskpkg.OwnerKindExtension),
		string(taskpkg.OwnerKindNetworkPeer),
		string(taskpkg.OwnerKindPool),
	}
}

func taskOriginKindValues() []string {
	return []string{
		string(taskpkg.OriginKindCLI),
		string(taskpkg.OriginKindWeb),
		string(taskpkg.OriginKindUDS),
		string(taskpkg.OriginKindHTTP),
		string(taskpkg.OriginKindAutomation),
		string(taskpkg.OriginKindExtension),
		string(taskpkg.OriginKindNetwork),
		string(taskpkg.OriginKindAgentSession),
		string(taskpkg.OriginKindDaemon),
	}
}

func taskDependencyKindValues() []string {
	return []string{
		string(taskpkg.DependencyKindBlocks),
	}
}

func taskInboxLaneValues() []string {
	return []string{
		string(contract.TaskInboxLaneMyWork),
		string(contract.TaskInboxLaneApprovals),
		string(contract.TaskInboxLaneFailedRuns),
		string(contract.TaskInboxLaneBlocked),
		string(contract.TaskInboxLaneArchived),
	}
}

func coordinationMessageKindValues() []string {
	kinds := contract.CoordinationMessageKinds()
	values := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, string(kind))
	}
	return values
}

func coordinatorConfigSourceValues() []string {
	return []string{
		string(contract.CoordinatorConfigSourceWorkspace),
		string(contract.CoordinatorConfigSourceGlobal),
		string(contract.CoordinatorConfigSourceDefault),
	}
}

func hookEventValues() []string {
	events := hooks.AllHookEvents()
	values := make([]string, 0, len(events))
	for _, event := range events {
		values = append(values, string(event))
	}
	return values
}

func hookEventFamilyValues() []string {
	return []string{
		string(hooks.HookEventFamilySession),
		string(hooks.HookEventFamilyInput),
		string(hooks.HookEventFamilyPrompt),
		string(hooks.HookEventFamilyEvent),
		string(hooks.HookEventFamilyAgent),
		string(hooks.HookEventFamilyTurn),
		string(hooks.HookEventFamilyMessage),
		string(hooks.HookEventFamilyTool),
		string(hooks.HookEventFamilyPermission),
		string(hooks.HookEventFamilyContext),
	}
}

func hookModeValues() []string {
	return []string{string(hooks.HookModeSync), string(hooks.HookModeAsync)}
}

func hookOutcomeValues() []string {
	return []string{
		string(hooks.HookRunOutcomeApplied),
		string(hooks.HookRunOutcomeDenied),
		string(hooks.HookRunOutcomeFailed),
		string(hooks.HookRunOutcomeSkipped),
		string(hooks.HookRunOutcomeDropped),
		string(hooks.HookRunOutcomeRejected),
	}
}

func hookSkillSourceValues() []string {
	return []string{
		string(hooks.HookSkillSourceBundled),
		string(hooks.HookSkillSourceMarketplace),
		string(hooks.HookSkillSourceUser),
		string(hooks.HookSkillSourceAdditional),
		string(hooks.HookSkillSourceWorkspace),
	}
}

func hookExecutorKindValues() []string {
	return []string{
		string(hooks.HookExecutorNative),
		string(hooks.HookExecutorSubprocess),
		string(hooks.HookExecutorWASM),
	}
}

func hookSourceValues() []string {
	return []string{"native", "config", "agent_definition", "skill"}
}

func memoryTypeValues() []string {
	return []string{
		string(memory.MemoryTypeUser),
		string(memory.MemoryTypeFeedback),
		string(memory.MemoryTypeProject),
		string(memory.MemoryTypeReference),
	}
}

func memoryScopeValues() []string {
	return []string{string(memory.ScopeGlobal), string(memory.ScopeWorkspace)}
}

func bridgeScopeValues() []string {
	return []string{string(bridgepkg.ScopeGlobal), string(bridgepkg.ScopeWorkspace)}
}

func bridgeInstanceSourceValues() []string {
	return []string{
		string(bridgepkg.BridgeInstanceSourceDynamic),
		string(bridgepkg.BridgeInstanceSourcePackage),
	}
}

func bridgeStatusValues() []string {
	return []string{
		string(bridgepkg.BridgeStatusAuthRequired),
		string(bridgepkg.BridgeStatusDegraded),
		string(bridgepkg.BridgeStatusDisabled),
		string(bridgepkg.BridgeStatusError),
		string(bridgepkg.BridgeStatusReady),
		string(bridgepkg.BridgeStatusStarting),
	}
}

func bridgeDMPolicyValues() []string {
	return []string{
		string(bridgepkg.BridgeDMPolicyOpen),
		string(bridgepkg.BridgeDMPolicyAllowlist),
		string(bridgepkg.BridgeDMPolicyPairing),
	}
}

func bridgeDegradationReasonValues() []string {
	return []string{
		string(bridgepkg.BridgeDegradationReasonAuthFailed),
		string(bridgepkg.BridgeDegradationReasonRateLimited),
		string(bridgepkg.BridgeDegradationReasonWebhookInvalid),
		string(bridgepkg.BridgeDegradationReasonProviderTimeout),
		string(bridgepkg.BridgeDegradationReasonTenantConfigInvalid),
	}
}

func deliveryModeValues() []string {
	return []string{
		string(bridgepkg.DeliveryModeDirectSend),
		string(bridgepkg.DeliveryModeReply),
	}
}

func sessionStateValues() []string {
	return []string{
		string(session.StateStarting),
		string(session.StateActive),
		string(session.StateStopping),
		string(session.StateStopped),
	}
}

func stopReasonValues() []string {
	return []string{
		string(store.StopCompleted),
		string(store.StopUserCanceled),
		string(store.StopMaxIterations),
		string(store.StopLoopDetected),
		string(store.StopTimeout),
		string(store.StopBudgetExceeded),
		string(store.StopError),
		string(store.StopAgentCrashed),
		string(store.StopHookStopped),
		string(store.StopShutdown),
	}
}

func bridgeProviderConfigSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithNullable().
		WithAdditionalProperties(openapi3.NewSchema())
}

func bridgeDeliveryDefaultsSchema() *openapi3.Schema {
	return openapi3.NewObjectSchema().
		WithNullable().
		WithProperty("peer_id", openapi3.NewStringSchema()).
		WithProperty("thread_id", openapi3.NewStringSchema()).
		WithProperty("group_id", openapi3.NewStringSchema()).
		WithProperty("mode", openapi3.NewStringSchema().WithEnum(enumAsAny(deliveryModeValues())...)).
		WithoutAdditionalProperties()
}

func toolSourceValues() []string {
	return []string{"builtin", "mcp", "extension", "dynamic"}
}

func hostAPIMethodValues() []string {
	specs := extensioncontract.HostAPIMethodSpecs()
	values := make([]string, 0, len(specs))
	for _, spec := range specs {
		values = append(values, string(spec.Method))
	}
	sort.Strings(values)
	return values
}
