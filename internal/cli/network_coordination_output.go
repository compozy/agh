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
				{Label: "Workspace", Value: stringOrDash(payload.WorkspaceID)},
				{Label: networkEnabledValue, Value: formatBool(payload.Enabled)},
				{Label: "Revision", Value: strconv.FormatInt(payload.Revision, 10)},
				{Label: "Updated By", Value: stringOrDash(payload.UpdatedBy)},
			}
			if payload.Invitation != nil {
				rows = append(rows,
					keyValue{Label: "Invitation Scope", Value: stringOrDash(payload.Invitation.Scope)},
					keyValue{Label: "Invitation Dismissed", Value: formatBool(payload.Invitation.Dismissed)},
				)
			}
			return renderHumanSection("Coordination", rows), nil
		},
	}
}

func networkUsageOutputBundle(payload NetworkUsageRecord) outputBundle {
	return outputBundle{
		jsonValue: payload,
		human: func() (string, error) {
			rows := []keyValue{
				{Label: "Workspace", Value: stringOrDash(payload.WorkspaceID)},
				{Label: "Wake Count", Value: strconv.Itoa(payload.Total.WakeCount)},
				{Label: "Actual Wakes", Value: strconv.Itoa(payload.Total.ActualWakeCount)},
				{Label: "Charged Wall Time", Value: stringOrDash(payload.Total.ChargedWallTime)},
				{Label: "Input Tokens", Value: strconv.FormatInt(payload.Total.InputTokens, 10)},
				{Label: "Output Tokens", Value: strconv.FormatInt(payload.Total.OutputTokens, 10)},
			}
			return renderHumanSection("Network Usage", rows), nil
		},
	}
}
