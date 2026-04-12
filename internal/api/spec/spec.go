package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/tools"
)

const (
	// DefaultPath is the canonical generated OpenAPI output location.
	DefaultPath = "openapi/agh.json"
)

var rawMessageType = reflect.TypeOf(json.RawMessage{})

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
			{Name: "agents"},
			{Name: "automation"},
			{Name: "channels"},
			{Name: "daemon"},
			{Name: "extensions"},
			{Name: "hooks"},
			{Name: "memory"},
			{Name: "observe"},
			{Name: "sessions"},
			{Name: "skills"},
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

// Operations returns the canonical REST operation registry in deterministic order.
func Operations() []OperationSpec {
	ops := []OperationSpec{
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
				enumQueryParam("scope", "Filter by automation scope", false, automationScopeValues()),
				queryParam("workspace_id", "Filter by workspace id", false),
				enumQueryParam("source", "Filter by job source", false, automationSourceValues()),
				intQueryParam("limit", "Maximum number of records to return", false),
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
				enumQueryParam("status", "Filter by run status", false, automationRunStatusValues()),
				dateTimeQueryParam("since", "Only runs started since this timestamp", false),
				dateTimeQueryParam("until", "Only runs started before this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
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
				enumQueryParam("scope", "Filter by automation scope", false, automationScopeValues()),
				queryParam("workspace_id", "Filter by workspace id", false),
				enumQueryParam("source", "Filter by trigger source", false, automationSourceValues()),
				queryParam("event", "Filter by trigger event", false),
				intQueryParam("limit", "Maximum number of records to return", false),
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
				enumQueryParam("status", "Filter by run status", false, automationRunStatusValues()),
				dateTimeQueryParam("since", "Only runs started since this timestamp", false),
				dateTimeQueryParam("until", "Only runs started before this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
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
				enumQueryParam("status", "Filter by run status", false, automationRunStatusValues()),
				dateTimeQueryParam("since", "Only runs started since this timestamp", false),
				dateTimeQueryParam("until", "Only runs started before this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
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
				headerParam("X-AGH-Webhook-Timestamp", "Signed webhook timestamp", true),
				headerParam("X-AGH-Webhook-Signature", "Signed webhook HMAC signature", true),
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
				headerParam("X-AGH-Webhook-Timestamp", "Signed webhook timestamp", true),
				headerParam("X-AGH-Webhook-Signature", "Signed webhook HMAC signature", true),
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
			Path:        "/api/channels",
			OperationID: "listChannels",
			Summary:     "List persisted channel instances",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelsResponse{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/channels",
			OperationID: "createChannel",
			Summary:     "Create a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			RequestBody: contract.CreateChannelRequest{},
			Responses: []ResponseSpec{
				{Status: 201, Description: "Created", Body: contract.ChannelResponse{}},
				{Status: 400, Description: "Invalid channel request", Body: contract.ErrorPayload{}},
				{Status: 404, Description: "Workspace not found", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/channels/{id}",
			OperationID: "getChannel",
			Summary:     "Get one channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelResponse{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "PATCH",
			Path:        "/api/channels/{id}",
			OperationID: "updateChannel",
			Summary:     "Update mutable channel instance fields",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			RequestBody: contract.UpdateChannelRequest{},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelResponse{}},
				{Status: 400, Description: "Invalid channel update", Body: contract.ErrorPayload{}},
				{Status: 404, Description: "Channel instance or workspace not found", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/channels/{id}/enable",
			OperationID: "enableChannel",
			Summary:     "Enable a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelResponse{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 409, Description: "Invalid channel state transition", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/channels/{id}/disable",
			OperationID: "disableChannel",
			Summary:     "Disable a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelResponse{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 409, Description: "Invalid channel state transition", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/channels/{id}/restart",
			OperationID: "restartChannel",
			Summary:     "Restart a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelResponse{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 409, Description: "Invalid channel state transition", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/channels/{id}/routes",
			OperationID: "listChannelRoutes",
			Summary:     "List routes owned by a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelRoutesResponse{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
				{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/channels/{id}/test-delivery",
			OperationID: "testChannelDelivery",
			Summary:     "Resolve a typed outbound delivery target for a channel instance",
			Tags:        []string{"channels"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("id", "Channel instance id"),
			},
			RequestBody: contract.ChannelTestDeliveryRequest{},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.ChannelTestDeliveryResponse{}},
				{Status: 400, Description: "Invalid delivery target request", Body: contract.ErrorPayload{}},
				{Status: 404, Description: "Channel instance not found", Body: contract.ErrorPayload{}},
				{Status: 409, Description: "Channel instance is unavailable", Body: contract.ErrorPayload{}},
				{Status: 503, Description: "Channel service is not configured", Body: contract.ErrorPayload{}},
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
			Path:        "/api/extensions",
			OperationID: "listExtensions",
			Summary:     "List installed extensions",
			Tags:        []string{"extensions"},
			Transports:  []Transport{TransportUDS},
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
			Transports:  []Transport{TransportUDS},
			RequestBody: contract.InstallExtensionRequest{},
			Responses: []ResponseSpec{
				{Status: 201, Description: "Created", Body: contract.ExtensionResponse{}},
				{Status: 400, Description: "Invalid install request", Body: contract.ErrorPayload{}},
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
			Transports:  []Transport{TransportUDS},
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
			Transports:  []Transport{TransportUDS},
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
			Path:        "/api/extensions/{name}/disable",
			OperationID: "disableExtension",
			Summary:     "Disable an installed extension",
			Tags:        []string{"extensions"},
			Transports:  []Transport{TransportUDS},
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
			Method:      "GET",
			Path:        "/api/hooks/catalog",
			OperationID: "getHookCatalog",
			Summary:     "List the resolved hook catalog",
			Tags:        []string{"hooks"},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				queryParam("workspace", "Workspace id or path", false),
				queryParam("agent", "Agent name", false),
				enumQueryParam("event", "Hook event name", false, hookEventValues()),
				enumQueryParam("source", "Hook source", false, hookSourceValues()),
				enumQueryParam("mode", "Hook mode", false, hookModeValues()),
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
				enumQueryParam("event", "Hook event name", false, hookEventValues()),
				enumQueryParam("outcome", "Hook execution outcome", false, hookOutcomeValues()),
				dateTimeQueryParam("since", "Only runs recorded since this timestamp", false),
				intQueryParam("last", "Maximum number of records to return", false),
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
				enumQueryParam("family", "Hook event family", false, hookEventFamilyValues()),
				boolQueryParam("sync_only", "Only return sync-eligible events", false),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.HookEventsResponse{}},
				{Status: 400, Description: "Invalid filter", Body: contract.ErrorPayload{}},
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
				enumQueryParam("scope", "Memory scope", false, memoryScopeValues()),
				queryParam("workspace", "Workspace id or path", false),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: []memory.MemoryHeader{}},
				{Status: 400, Description: "Invalid memory filter", Body: contract.ErrorPayload{}},
				{Status: 404, Description: "Workspace or memory not found", Body: contract.ErrorPayload{}},
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
				enumQueryParam("scope", "Memory scope", false, memoryScopeValues()),
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
				enumQueryParam("scope", "Memory scope", false, memoryScopeValues()),
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
				dateTimeQueryParam("since", "Only events emitted since this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
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
			OperationID: "stopSession",
			Summary:     "Stop a session",
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
				dateTimeQueryParam("since", "Only events emitted since this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
				int64QueryParam("after_sequence", "Only return events after this sequence number", false),
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
				dateTimeQueryParam("since", "Only events emitted since this timestamp", false),
				intQueryParam("limit", "Maximum number of records to return", false),
				int64QueryParam("after_sequence", "Only return events after this sequence number", false),
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
	return os.WriteFile(path, data, 0o644)
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
			resp.WithContent(openapi3.NewContentWithJSONSchemaRef(schemaRef))
		}
		operation.AddResponse(response.Status, resp)
	}

	return operation, nil
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
	switch t {
	case rawMessageType:
		*schema = *openapi3.NewSchema()
		return nil
	case reflect.TypeOf(automationpkg.AutomationScope("")):
		setStringEnum(schema, automationScopeValues())
		return nil
	case reflect.TypeOf(automationpkg.JobSource("")):
		setStringEnum(schema, automationSourceValues())
		return nil
	case reflect.TypeOf(automationpkg.ScheduleMode("")):
		setStringEnum(schema, automationScheduleModeValues())
		return nil
	case reflect.TypeOf(automationpkg.RetryStrategy("")):
		setStringEnum(schema, automationRetryStrategyValues())
		return nil
	case reflect.TypeOf(automationpkg.RunStatus("")):
		setStringEnum(schema, automationRunStatusValues())
		return nil
	case reflect.TypeOf(hooks.HookEvent("")):
		setStringEnum(schema, hookEventValues())
		return nil
	case reflect.TypeOf(hooks.HookEventFamily("")):
		setStringEnum(schema, hookEventFamilyValues())
		return nil
	case reflect.TypeOf(hooks.HookMode("")):
		setStringEnum(schema, hookModeValues())
		return nil
	case reflect.TypeOf(hooks.HookRunOutcome("")):
		setStringEnum(schema, hookOutcomeValues())
		return nil
	case reflect.TypeOf(hooks.HookSkillSource("")):
		setStringEnum(schema, hookSkillSourceValues())
		return nil
	case reflect.TypeOf(hooks.HookExecutorKind("")):
		setStringEnum(schema, hookExecutorKindValues())
		return nil
	case reflect.TypeOf(hooks.HookSource(0)):
		setStringEnum(schema, hookSourceValues())
		return nil
	case reflect.TypeOf(memory.MemoryType("")):
		setStringEnum(schema, memoryTypeValues())
		return nil
	case reflect.TypeOf(memory.Scope("")):
		setStringEnum(schema, memoryScopeValues())
		return nil
	case reflect.TypeOf(channelspkg.Scope("")):
		setStringEnum(schema, channelScopeValues())
		return nil
	case reflect.TypeOf(channelspkg.ChannelStatus("")):
		setStringEnum(schema, channelStatusValues())
		return nil
	case reflect.TypeOf(channelspkg.DeliveryMode("")):
		setStringEnum(schema, deliveryModeValues())
		return nil
	case reflect.TypeOf(session.SessionState("")):
		setStringEnum(schema, sessionStateValues())
		return nil
	case reflect.TypeOf(store.StopReason("")):
		setStringEnum(schema, stopReasonValues())
		return nil
	case reflect.TypeOf(tools.ToolSource(0)):
		setStringEnum(schema, toolSourceValues())
		return nil
	case reflect.TypeOf(extensionprotocol.HostAPIMethod("")):
		setStringEnum(schema, hostAPIMethodValues())
		return nil
	default:
		return nil
	}
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
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitEmpty = true
			break
		}
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

func pathParam(name string, description string) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInPath, Description: description, Required: true}
}

func headerParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInHeader, Description: description, Required: required}
}

func queryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{Name: name, In: openapi3.ParameterInQuery, Description: description, Required: required}
}

func enumQueryParam(name string, description string, required bool, values []string) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    required,
		Enum:        values,
	}
}

func boolQueryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    required,
		Kind:        "boolean",
	}
}

func intQueryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    required,
		Kind:        "integer",
		Format:      "int32",
	}
}

func int64QueryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    required,
		Kind:        "integer",
		Format:      "int64",
	}
}

func dateTimeQueryParam(name string, description string, required bool) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          openapi3.ParameterInQuery,
		Description: description,
		Required:    required,
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
		string(automationpkg.RunCompleted),
		string(automationpkg.RunFailed),
		string(automationpkg.RunCancelled),
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

func channelScopeValues() []string {
	return []string{string(channelspkg.ScopeGlobal), string(channelspkg.ScopeWorkspace)}
}

func channelStatusValues() []string {
	return []string{
		string(channelspkg.ChannelStatusAuthRequired),
		string(channelspkg.ChannelStatusDegraded),
		string(channelspkg.ChannelStatusDisabled),
		string(channelspkg.ChannelStatusError),
		string(channelspkg.ChannelStatusReady),
		string(channelspkg.ChannelStatusStarting),
	}
}

func deliveryModeValues() []string {
	return []string{
		string(channelspkg.DeliveryModeDirectSend),
		string(channelspkg.DeliveryModeReply),
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
