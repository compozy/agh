package spec

import "github.com/compozy/agh/internal/api/contract"

func networkCoordinationOperations() []OperationSpec {
	return []OperationSpec{
		{
			Method:      httpMethodGet,
			Path:        "/api/workspaces/{workspace_id}/network-coordination",
			OperationID: "getNetworkCoordination",
			Summary:     "Get workspace network coordination settings and invitation state",
			Tags:        []string{specNetworkKey},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("workspace_id", "Workspace id"),
				queryParam("task_id", "Optional task scope for invitation state", false),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.NetworkCoordinationResponse{}},
				{Status: 404, Description: specWorkspaceNotFoundDescription, Body: contract.ErrorPayload{}},
				{Status: 500, Description: specInternalServerErrorDescription, Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      httpMethodPut,
			Path:        "/api/workspaces/{workspace_id}/network-coordination",
			OperationID: "putNetworkCoordination",
			Summary:     "Enable or disable workspace network coordination conversations",
			Tags:        []string{specNetworkKey},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("workspace_id", "Workspace id"),
				queryParam("task_id", "Optional task scope for invitation state", false),
			},
			RequestBody: contract.PutNetworkCoordinationRequest{},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.NetworkCoordinationResponse{}},
				{Status: 400, Description: "Invalid coordination request", Body: contract.ErrorPayload{}},
				{Status: 404, Description: specWorkspaceNotFoundDescription, Body: contract.ErrorPayload{}},
				{Status: 409, Description: "Network participation unavailable", Body: contract.ErrorPayload{}},
				{Status: 500, Description: specInternalServerErrorDescription, Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      httpMethodPut,
			Path:        "/api/workspaces/{workspace_id}/network-coordination/invitation",
			OperationID: "putNetworkCoordinationInvitation",
			Summary:     "Dismiss or reset the coordination invitation for a scope",
			Tags:        []string{specNetworkKey},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("workspace_id", "Workspace id"),
			},
			RequestBody: contract.PutNetworkCoordinationInvitationRequest{},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.NetworkCoordinationResponse{}},
				{Status: 400, Description: "Invalid invitation request", Body: contract.ErrorPayload{}},
				{Status: 404, Description: specWorkspaceNotFoundDescription, Body: contract.ErrorPayload{}},
				{Status: 500, Description: specInternalServerErrorDescription, Body: contract.ErrorPayload{}},
			},
		},
		{
			Method:      httpMethodGet,
			Path:        "/api/workspaces/{workspace_id}/network/usage",
			OperationID: "getNetworkUsage",
			Summary:     "Get workspace-scoped network wake usage from the ledger",
			Tags:        []string{specNetworkKey},
			Transports:  []Transport{TransportHTTP, TransportUDS},
			Parameters: []ParameterSpec{
				pathParam("workspace_id", "Workspace id"),
			},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.NetworkUsageResponse{}},
				{Status: 404, Description: specWorkspaceNotFoundDescription, Body: contract.ErrorPayload{}},
				{Status: 500, Description: specInternalServerErrorDescription, Body: contract.ErrorPayload{}},
			},
		},
	}
}
