package cli

import "strconv"

func sessionResolvedChannel(info SessionRecord) string {
	if info.ResolvedNetworkParticipation == nil {
		return ""
	}
	return stringOrDash(info.ResolvedNetworkParticipation.ChannelID)
}

func sessionResolvedChannelRaw(info SessionRecord) string {
	if info.ResolvedNetworkParticipation == nil {
		return ""
	}
	return info.ResolvedNetworkParticipation.ChannelID
}

func coordinationOutputBundle(payload NetworkCoordinationRecord) outputBundle {
	return outputBundle{
		jsonValue: map[string]any{"coordination": payload},
		human: func() (string, error) {
			rows := []keyValue{
				{Label: configWorkspaceValue, Value: stringOrDash(payload.WorkspaceID)},
				{Label: "Scope", Value: stringOrDash(payload.Scope)},
				{Label: "Task", Value: stringOrDash(payload.TaskID)},
				{Label: networkEnabledValue, Value: formatBool(payload.Enabled)},
				{Label: "Revision", Value: strconv.FormatInt(payload.Revision, 10)},
				{Label: "Updated By", Value: stringOrDash(payload.UpdatedBy)},
				{Label: "Invitation Eligible", Value: formatBool(payload.Eligibility.Eligible)},
				{Label: "Eligibility Reason", Value: stringOrDash(payload.Eligibility.Reason)},
				{Label: "Coordinator Run", Value: stringOrDash(payload.Eligibility.RunID)},
				{Label: "Worker Count", Value: strconv.Itoa(payload.Eligibility.WorkerCount)},
			}
			if payload.Invitation != nil {
				rows = append(rows,
					keyValue{Label: "Invitation Scope", Value: stringOrDash(payload.Invitation.Scope)},
					keyValue{Label: "Invitation Dismissed", Value: formatBool(payload.Invitation.Dismissed)},
				)
			}
			return renderHumanSection("Coordination", rows), nil
		},
		toon: func() (string, error) {
			fields := []string{
				taskWorkspaceIDKey,
				automationScopeKey,
				taskTaskIDKey,
				networkEnabledKey,
				"revision",
				automationUpdatedAtKey,
				"updated_by",
				"invitation_eligible", "eligibility_reason", "coordinator_run_id", "coordinator", "worker_count",
				"invitation_scope", "invitation_task_id", "invitation_dismissed", "invitation_dismissed_at",
				"invitation_dismissed_by",
			}
			values := []string{
				payload.WorkspaceID,
				payload.Scope,
				payload.TaskID,
				strconv.FormatBool(payload.Enabled),
				strconv.FormatInt(payload.Revision, 10),
				formatTimePtr(payload.UpdatedAt),
				payload.UpdatedBy,
				strconv.FormatBool(payload.Eligibility.Eligible),
				payload.Eligibility.Reason,
				payload.Eligibility.RunID,
				strconv.FormatBool(payload.Eligibility.Coordinator),
				strconv.Itoa(payload.Eligibility.WorkerCount),
				"", "", toolBoolFalse, "", "",
			}
			if payload.Invitation != nil {
				values[12] = payload.Invitation.Scope
				values[13] = payload.Invitation.TaskID
				values[14] = strconv.FormatBool(payload.Invitation.Dismissed)
				values[15] = formatTimePtr(payload.Invitation.DismissedAt)
				values[16] = payload.Invitation.DismissedBy
			}
			return renderToonObject("network_coordination", fields, values), nil
		},
	}
}

func networkUsageOutputBundle(payload NetworkUsageRecord) outputBundle {
	return outputBundle{
		jsonValue: payload,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: configWorkspaceValue, Value: stringOrDash(payload.WorkspaceID)},
				{Label: "Wake Count", Value: strconv.Itoa(payload.Total.WakeCount)},
				{Label: "Reserved Wakes", Value: strconv.Itoa(payload.Total.ReservedWakeCount)},
				{Label: "Actual Wakes", Value: strconv.Itoa(payload.Total.ActualWakeCount)},
				{Label: "Unavailable Wakes", Value: strconv.Itoa(payload.Total.UnavailableWakeCount)},
				{Label: "Charged Wall Time", Value: stringOrDash(payload.Total.ChargedWallTime)},
				{Label: "Input Tokens", Value: strconv.FormatInt(payload.Total.InputTokens, 10)},
				{Label: "Output Tokens", Value: strconv.FormatInt(payload.Total.OutputTokens, 10)},
				{Label: "Next Cursor", Value: stringOrDash(payload.NextCursor)},
			}
			if payload.Budget != nil {
				rows = append(rows,
					keyValue{Label: "Budget Owner", Value: stringOrDash(payload.Budget.OwnerKey)},
					keyValue{Label: "Budget Exhausted", Value: formatBool(payload.Budget.ExhaustedReason != "")},
					keyValue{Label: "Exhausted Reason", Value: stringOrDash(payload.Budget.ExhaustedReason)},
				)
			}
			if len(payload.Details) > 0 {
				latest := payload.Details[0]
				rows = append(rows,
					keyValue{Label: "Latest Wake State", Value: stringOrDash(latest.State)},
					keyValue{Label: "Latest Usage State", Value: stringOrDash(latest.UsageState)},
				)
			}
			return renderHumanSection("Network Usage", rows), nil
		},
		toon: func() (string, error) {
			return networkUsageToon(payload), nil
		},
	}
}

func networkUsageToon(payload NetworkUsageRecord) string {
	fields := []string{
		taskWorkspaceIDKey, "wake_count", "reserved_wake_count", "actual_wake_count",
		"unavailable_wake_count", "charged_wall_time", "input_tokens", "output_tokens", "next_cursor",
		"budget_owner", "budget_wakes_used", "budget_wall_time_used", "budget_input_tokens_used",
		"budget_output_tokens_used", "budget_exhausted_reason", "budget_updated_at",
	}
	values := []string{
		payload.WorkspaceID,
		strconv.Itoa(payload.Total.WakeCount),
		strconv.Itoa(payload.Total.ReservedWakeCount),
		strconv.Itoa(payload.Total.ActualWakeCount),
		strconv.Itoa(payload.Total.UnavailableWakeCount),
		payload.Total.ChargedWallTime,
		strconv.FormatInt(payload.Total.InputTokens, 10),
		strconv.FormatInt(payload.Total.OutputTokens, 10),
		payload.NextCursor,
		"", "", "", "", "", "", "",
	}
	if payload.Budget != nil {
		values[9] = payload.Budget.OwnerKey
		values[10] = strconv.Itoa(payload.Budget.WakesUsed)
		values[11] = payload.Budget.WallTimeUsed
		values[12] = strconv.FormatInt(payload.Budget.InputTokensUsed, 10)
		values[13] = strconv.FormatInt(payload.Budget.OutputTokensUsed, 10)
		values[14] = payload.Budget.ExhaustedReason
		values[15] = formatTime(payload.Budget.UpdatedAt)
	}
	blocks := []string{renderToonObject("network_usage", fields, values)}
	if len(payload.Details) > 0 {
		blocks = append(blocks, renderToonArray(
			"network_usage_details",
			[]string{
				"wake_id",
				"task_run_id",
				"owner_key",
				taskWorkspaceIDKey,
				"channel",
				"root_id",
				"depth",
				networkStateKey,
				"usage_state",
				"charged_wall_time",
				"input_tokens",
				"output_tokens",
				"reserved_at",
				"settled_at", "reason",
			},
			networkUsageDetailToonRows(payload),
		))
	}
	return renderHumanBlocks(blocks...)
}

func networkUsageDetailToonRows(payload NetworkUsageRecord) [][]string {
	rows := make([][]string, 0, len(payload.Details))
	for _, detail := range payload.Details {
		rows = append(rows, []string{
			detail.WakeID,
			detail.TaskRunID,
			detail.OwnerKey,
			detail.WorkspaceID,
			detail.Channel,
			detail.RootID,
			strconv.Itoa(detail.Depth),
			detail.State,
			detail.UsageState,
			detail.ChargedWallTime,
			strconv.FormatInt(detail.InputTokens, 10),
			strconv.FormatInt(detail.OutputTokens, 10),
			formatTime(detail.ReservedAt),
			formatTimePtr(detail.SettledAt),
			detail.Reason,
		})
	}
	return rows
}
