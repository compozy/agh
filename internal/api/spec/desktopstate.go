package spec

import "github.com/compozy/agh/internal/api/contract"

const (
	desktopStateCollectionPath = "/api/workspaces/{workspace_id}/desktop-state"
	desktopStateItemPath       = desktopStateCollectionPath + "/{key}"
)

func desktopStateOperations() []OperationSpec {
	transports := []Transport{TransportHTTP, TransportUDS}
	workspaceParam := pathParam("workspace_id", "Workspace id")
	keyParam := pathParam("key", "Desktop-state key")
	errorBody := contract.DesktopStateErrorPayload{}
	mutationErrors := []ResponseSpec{
		{Status: 404, Description: "Desktop state or workspace not found", Body: errorBody},
		{Status: 409, Description: "Revision conflict", Body: errorBody},
		{Status: 413, Description: specPayloadTooLargeDescription, Body: errorBody},
		{Status: 422, Description: "Invalid desktop-state mutation", Body: errorBody},
		{Status: 500, Description: specInternalServerErrorDescription, Body: errorBody},
	}

	return []OperationSpec{
		{
			Method: httpMethodGet, Path: desktopStateCollectionPath,
			OperationID: "listDesktopState", Summary: "List one workspace desktop-state snapshot",
			Tags: []string{specWorkspacesKey}, Transports: transports,
			Parameters: []ParameterSpec{workspaceParam},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.DesktopStateListResponse{}},
				{Status: 404, Description: specWorkspaceNotFoundDescription, Body: errorBody},
				{Status: 500, Description: specInternalServerErrorDescription, Body: errorBody},
			},
		},
		{
			Method: httpMethodGet, Path: desktopStateItemPath,
			OperationID: "getDesktopState", Summary: "Read one workspace desktop-state value",
			Tags: []string{specWorkspacesKey}, Transports: transports,
			Parameters: []ParameterSpec{workspaceParam, keyParam},
			Responses: []ResponseSpec{
				{Status: 200, Description: "OK", Body: contract.DesktopStateEntry{}},
				{Status: 404, Description: "Desktop state or workspace not found", Body: errorBody},
				{Status: 500, Description: specInternalServerErrorDescription, Body: errorBody},
			},
		},
		{
			Method: httpMethodPut, Path: desktopStateItemPath,
			OperationID: "putDesktopState", Summary: "Create or replace one workspace desktop-state value",
			Tags: []string{specWorkspacesKey}, Transports: transports,
			Parameters:  []ParameterSpec{workspaceParam, keyParam},
			RequestBody: contract.DesktopStatePutRequest{},
			Responses: append(
				[]ResponseSpec{{Status: 200, Description: "Updated", Body: contract.DesktopStateEntry{}}},
				mutationErrors...,
			),
		},
		{
			Method: httpMethodPost, Path: desktopStateCollectionPath + "/apply",
			OperationID: "applyDesktopState", Summary: "Atomically mutate workspace desktop state",
			Tags: []string{specWorkspacesKey}, Transports: transports,
			Parameters:  []ParameterSpec{workspaceParam},
			RequestBody: contract.DesktopStateApplyRequest{},
			Responses: append(
				[]ResponseSpec{{Status: 200, Description: "Applied", Body: contract.DesktopStateApplyResponse{}}},
				mutationErrors...,
			),
		},
		{
			Method: httpMethodDelete, Path: desktopStateItemPath,
			OperationID: "deleteDesktopState", Summary: "Delete one workspace desktop-state value",
			Tags: []string{specWorkspacesKey}, Transports: transports,
			Parameters: []ParameterSpec{
				workspaceParam,
				keyParam,
				desktopStateRevisionQueryParam(),
			},
			Responses: append(
				[]ResponseSpec{{Status: 204, Description: specNoContentDescription}},
				mutationErrors...,
			),
		},
		desktopStateStreamOperation(transports, workspaceParam, errorBody),
	}
}

func desktopStateStreamOperation(
	transports []Transport,
	workspaceParam ParameterSpec,
	errorBody contract.DesktopStateErrorPayload,
) OperationSpec {
	return OperationSpec{
		Method: httpMethodGet, Path: desktopStateCollectionPath + "/stream",
		OperationID: "streamDesktopState", Summary: "Stream one workspace desktop-state snapshot and deltas",
		Tags: []string{specWorkspacesKey}, Transports: transports,
		Parameters: []ParameterSpec{workspaceParam},
		Responses: []ResponseSpec{
			{
				Status: 101, Description: "WebSocket upgrade and frame contract",
				Body: contract.DesktopStateWebSocketContract{},
			},
			{Status: 404, Description: specWorkspaceNotFoundDescription, Body: errorBody},
		},
	}
}

func desktopStateRevisionQueryParam() ParameterSpec {
	parameter := intQueryParam("if_rev", "Expected current revision")
	maximum := float64(contract.DesktopStateMaxSafeNumber)
	parameter.Maximum = &maximum
	return parameter
}
