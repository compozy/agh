package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/network/participation"
	"github.com/spf13/cobra"
)

type networkParticipationFlags struct {
	mode            string
	channelStrategy string
	channel         string
	boundsJSON      string
}

func bindNetworkParticipationFlags(cmd *cobra.Command, flags *networkParticipationFlags) {
	cmd.Flags().StringVar(&flags.mode, "network", "", "Network participation mode: local or live")
	cmd.Flags().StringVar(
		&flags.channelStrategy,
		"network-channel-strategy",
		"",
		"Live channel strategy: named, run, or loop_run",
	)
	cmd.Flags().StringVar(&flags.channel, "network-channel", "", "Named channel id when strategy is named")
	cmd.Flags().StringVar(
		&flags.boundsJSON,
		"network-bounds",
		"",
		"JSON object for live participation bounds (one object, not flag explosion)",
	)
}

func (f networkParticipationFlags) request() (*participation.Request, error) {
	mode := strings.TrimSpace(f.mode)
	strategy := strings.TrimSpace(f.channelStrategy)
	channel := strings.TrimSpace(f.channel)
	boundsRaw := strings.TrimSpace(f.boundsJSON)
	if mode == "" && strategy == "" && channel == "" && boundsRaw == "" {
		return nil, nil
	}
	req := &participation.Request{}
	if mode != "" {
		value := participation.Mode(mode)
		req.Mode = &value
	}
	if strategy != "" {
		value := participation.ChannelStrategy(strategy)
		req.ChannelStrategy = &value
	}
	if channel != "" {
		req.ChannelID = &channel
	}
	if boundsRaw != "" {
		var bounds participation.BoundsRequest
		if err := json.Unmarshal([]byte(boundsRaw), &bounds); err != nil {
			return nil, fmt.Errorf("cli: parse --network-bounds: %w", err)
		}
		req.Bounds = &bounds
	}
	return req, nil
}

func bindConfirmNetworkRequirementFlag(cmd *cobra.Command, confirmed *bool) {
	cmd.Flags().BoolVar(
		confirmed,
		"confirm-network-requirement",
		false,
		"Confirm the extension Live network participation requirement digest",
	)
}
