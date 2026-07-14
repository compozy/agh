package cli

import "github.com/compozy/agh/internal/network/participation"

func resolvedParticipationChannelID(spec *participation.Spec) string {
	if spec == nil {
		return ""
	}
	return spec.ChannelID
}

func participationRequestChannelID(req *participation.Request) string {
	if req == nil || req.ChannelID == nil {
		return ""
	}
	return *req.ChannelID
}

func testParticipationChannelID(channel string) *string {
	value := channel
	return &value
}

func testLiveNamedParticipationRequest(channel string) *participation.Request {
	mode := participation.ModeLive
	strategy := participation.StrategyNamed
	return &participation.Request{
		Mode:            &mode,
		ChannelStrategy: &strategy,
		ChannelID:       testParticipationChannelID(channel),
	}
}

func testLiveResolvedParticipation(channel string) *participation.Spec {
	return &participation.Spec{
		Version:   participation.SpecVersion,
		Mode:      participation.ModeLive,
		ChannelID: channel,
		Source:    participation.SourceExplicitRequest,
	}
}
