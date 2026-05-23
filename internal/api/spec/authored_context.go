package spec

import "github.com/compozy/agh/internal/api/contract"

const (
	authoredContextAPIAgentSoulPath                      = "/api/agent/soul"
	authoredContextAPIAgentsAgentNameHeartbeatPath       = "/api/agents/{agent_name}/heartbeat"
	authoredContextAPIAgentsAgentNameHeartbeatStatusPath = "/api/agents/{agent_name}/heartbeat/status"
	authoredContextAPIAgentsAgentNameSoulPath            = "/api/agents/{agent_name}/soul"
	authoredContextSessionInspectPath                    = "/api/workspaces/{workspace_id}/sessions/" +
		"{session_id}/inspect"
	authoredContextAgentCallerIdentityIsMissingDescription           = "Agent caller identity is missing"
	authoredContextAgentOrWorkspaceNotFoundDescription               = "Agent or workspace not found"
	authoredContextForbiddenWorkspaceOrPermissionMismatchDescription = "Forbidden - workspace or permission mismatch"
	authoredContextHeartbeatAuthoringConflictDescription             = "Heartbeat authoring conflict"
	authoredContextHeartbeatValidationFailedDescription              = "Heartbeat validation failed"
	authoredContextInternalServerErrorDescription                    = "Internal server error"
	authoredContextSessionNotFoundDescription                        = "Session not found"
	authoredContextSoulAuthoringConflictDescription                  = "Soul authoring conflict"
	authoredContextSoulValidationFailedDescription                   = "Soul validation failed"
	authoredContextAgentKey                                          = "agent"
	authoredContextAgentsKey                                         = "agents"
	authoredContextSessionsKey                                       = "sessions"
)

var authoredContextOperationRegistry = []OperationSpec{
	{
		Method:      httpMethodGet,
		Path:        authoredContextAPIAgentSoulPath,
		OperationID: "getAgentSoul",
		Summary:     "Inspect the resolved Soul read model for the calling agent",
		Tags:        []string{authoredContextAgentKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulPayload{}},
			{
				Status:      401,
				Description: authoredContextAgentCallerIdentityIsMissingDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Caller session or agent not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Soul is invalid", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agent/soul/validate",
		OperationID: "validateAgentSoul",
		Summary:     "Validate a proposed Soul body or the calling agent's current Soul",
		Tags:        []string{authoredContextAgentKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		RequestBody: contract.AgentSoulValidateRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulPayload{}},
			{
				Status:      401,
				Description: authoredContextAgentCallerIdentityIsMissingDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Caller session or agent not found", Body: contract.ErrorPayload{}},
			{
				Status:      422,
				Description: authoredContextSoulValidationFailedDescription,
				Body:        contract.AgentSoulPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        authoredContextAPIAgentsAgentNameSoulPath,
		OperationID: "getAgentDefinitionSoul",
		Summary:     "Inspect the resolved Soul read model for an agent definition",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
			queryParam("workspace_id", "Workspace id", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulPayload{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 422, Description: "Soul is invalid", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agents/{agent_name}/soul/validate",
		OperationID: "validateAgentDefinitionSoul",
		Summary:     "Validate a proposed Soul body for an agent definition",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.AgentSoulValidateByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulPayload{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      422,
				Description: authoredContextSoulValidationFailedDescription,
				Body:        contract.AgentSoulPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPut,
		Path:        authoredContextAPIAgentsAgentNameSoulPath,
		OperationID: "putAgentSoul",
		Summary:     "Create or replace SOUL.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.AgentSoulPutByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 409, Description: authoredContextSoulAuthoringConflictDescription, Body: contract.ErrorPayload{}},
			{
				Status:      422,
				Description: authoredContextSoulValidationFailedDescription,
				Body:        contract.AgentSoulPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodDelete,
		Path:        authoredContextAPIAgentsAgentNameSoulPath,
		OperationID: "deleteAgentSoul",
		Summary:     "Delete SOUL.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.AgentSoulDeleteByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Agent, workspace, or Soul file not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: authoredContextSoulAuthoringConflictDescription, Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Invalid Soul delete request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        "/api/agents/{agent_name}/soul/history",
		OperationID: "listAgentSoulHistory",
		Summary:     "List managed SOUL.md authoring revisions",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
			queryParam("workspace_id", "Workspace id", false),
			intQueryParam("limit", "Maximum number of revisions to return"),
			queryParam("cursor", "Revision cursor", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulHistoryResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 422, Description: "Invalid Soul history request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agents/{agent_name}/soul/rollback",
		OperationID: "rollbackAgentSoul",
		Summary:     "Rollback SOUL.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.AgentSoulRollbackByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Agent, workspace, or revision not found", Body: contract.ErrorPayload{}},
			{Status: 409, Description: authoredContextSoulAuthoringConflictDescription, Body: contract.ErrorPayload{}},
			{
				Status:      422,
				Description: authoredContextSoulValidationFailedDescription,
				Body:        contract.AgentSoulPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/workspaces/{workspace_id}/sessions/{session_id}/soul/refresh",
		OperationID: "refreshSessionSoul",
		Summary:     "Refresh an idle session's Soul snapshot through body-level CAS",
		Tags:        []string{authoredContextSessionsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("workspace_id", "Workspace id"),
			pathParam("session_id", "Session id"),
		},
		RequestBody: contract.SessionSoulRefreshRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.AgentSoulPayload{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: authoredContextSessionNotFoundDescription, Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Session is not idle or Soul digest is stale", Body: contract.ErrorPayload{}},
			{
				Status:      422,
				Description: authoredContextSoulValidationFailedDescription,
				Body:        contract.AgentSoulPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        authoredContextAPIAgentsAgentNameHeartbeatPath,
		OperationID: "getAgentHeartbeat",
		Summary:     "Inspect the resolved Heartbeat policy for an agent definition",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
			queryParam("workspace_id", "Workspace id", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatPolicyPayload{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 422, Description: "Heartbeat policy is invalid", Body: contract.HeartbeatPolicyPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agents/{agent_name}/heartbeat/validate",
		OperationID: "validateAgentHeartbeat",
		Summary:     "Validate a proposed HEARTBEAT.md body",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.HeartbeatValidateByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatPolicyPayload{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      422,
				Description: authoredContextHeartbeatValidationFailedDescription,
				Body:        contract.HeartbeatPolicyPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPut,
		Path:        authoredContextAPIAgentsAgentNameHeartbeatPath,
		OperationID: "putAgentHeartbeat",
		Summary:     "Create or replace HEARTBEAT.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.HeartbeatPutByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      409,
				Description: authoredContextHeartbeatAuthoringConflictDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      422,
				Description: authoredContextHeartbeatValidationFailedDescription,
				Body:        contract.HeartbeatPolicyPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodDelete,
		Path:        authoredContextAPIAgentsAgentNameHeartbeatPath,
		OperationID: "deleteAgentHeartbeat",
		Summary:     "Delete HEARTBEAT.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.HeartbeatDeleteByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Agent, workspace, or policy not found", Body: contract.ErrorPayload{}},
			{
				Status:      409,
				Description: authoredContextHeartbeatAuthoringConflictDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 422, Description: "Invalid Heartbeat delete request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        "/api/agents/{agent_name}/heartbeat/history",
		OperationID: "listAgentHeartbeatHistory",
		Summary:     "List managed HEARTBEAT.md authoring revisions",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
			queryParam("workspace_id", "Workspace id", false),
			intQueryParam("limit", "Maximum number of revisions to return"),
			queryParam("cursor", "Revision cursor", false),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatHistoryResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: authoredContextAgentOrWorkspaceNotFoundDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 422, Description: "Invalid Heartbeat history request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agents/{agent_name}/heartbeat/rollback",
		OperationID: "rollbackAgentHeartbeat",
		Summary:     "Rollback HEARTBEAT.md through managed authoring",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.HeartbeatRollbackByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatMutationResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      404,
				Description: "Agent, workspace, revision, or snapshot not found",
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      409,
				Description: authoredContextHeartbeatAuthoringConflictDescription,
				Body:        contract.ErrorPayload{},
			},
			{
				Status:      422,
				Description: authoredContextHeartbeatValidationFailedDescription,
				Body:        contract.HeartbeatPolicyPayload{},
			},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        authoredContextAPIAgentsAgentNameHeartbeatStatusPath,
		OperationID: "getAgentHeartbeatStatus",
		Summary:     "Read Heartbeat policy status, wake state, and optional session health",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
			queryParam("workspace_id", "Workspace id", false),
			queryParam("session_id", "Session id for wake state and health", false),
			boolQueryParam("include_session_health", "Include session health when a session id is supplied"),
			boolQueryParam("include_recent_wake_events", "Include recent wake audit rows"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatStatusResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Agent, workspace, or session not found", Body: contract.ErrorPayload{}},
			{Status: 422, Description: "Heartbeat status request is invalid", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodPost,
		Path:        "/api/agents/{agent_name}/heartbeat/wake",
		OperationID: "wakeAgentHeartbeat",
		Summary:     "Request one advisory Heartbeat wake for an eligible session",
		Tags:        []string{authoredContextAgentsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("agent_name", "Agent name"),
		},
		RequestBody: contract.HeartbeatWakeByPathRequest{},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.HeartbeatWakeResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: "Agent, workspace, policy, or session not found", Body: contract.ErrorPayload{}},
			{
				Status:      409,
				Description: "Wake skipped or coalesced by policy and health gates",
				Body:        contract.HeartbeatWakeResponse{},
			},
			{Status: 422, Description: "Invalid Heartbeat wake request", Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        "/api/workspaces/{workspace_id}/sessions/{session_id}/health",
		OperationID: "getSessionHealth",
		Summary:     "Read metadata-only session health and wake eligibility",
		Tags:        []string{authoredContextSessionsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("workspace_id", "Workspace id"),
			pathParam("session_id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionHealthResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: authoredContextSessionNotFoundDescription, Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        "/api/workspaces/{workspace_id}/sessions/{session_id}/status",
		OperationID: "getSessionStatus",
		Summary:     "Read compact session status and wake eligibility",
		Tags:        []string{authoredContextSessionsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("workspace_id", "Workspace id"),
			pathParam("session_id", "Session id"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionStatusResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: authoredContextSessionNotFoundDescription, Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
	{
		Method:      httpMethodGet,
		Path:        authoredContextSessionInspectPath,
		OperationID: "inspectSession",
		Summary:     "Inspect session health, wake audit, and policy correlation metadata",
		Tags:        []string{authoredContextSessionsKey},
		Transports:  []Transport{TransportHTTP, TransportUDS},
		Parameters: []ParameterSpec{
			pathParam("workspace_id", "Workspace id"),
			pathParam("session_id", "Session id"),
			boolQueryParam("include_recent_wake_events", "Include recent wake audit rows"),
		},
		Responses: []ResponseSpec{
			{Status: 200, Description: "OK", Body: contract.SessionInspectResponse{}},
			{
				Status:      403,
				Description: authoredContextForbiddenWorkspaceOrPermissionMismatchDescription,
				Body:        contract.ErrorPayload{},
			},
			{Status: 404, Description: authoredContextSessionNotFoundDescription, Body: contract.ErrorPayload{}},
			{Status: 500, Description: authoredContextInternalServerErrorDescription, Body: contract.ErrorPayload{}},
		},
	},
}

func authoredContextOperations() []OperationSpec {
	return cloneOperationSpecs(authoredContextOperationRegistry)
}
