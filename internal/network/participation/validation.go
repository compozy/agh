package participation

import (
	"fmt"
	"regexp"
	"strings"
)

var channelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func Validate(request Request) (Request, error) {
	return validateRequest(request, Bounds{}, Limits{}, true)
}

func ValidateWithBounds(request Request, defaults Bounds, limits Limits) (Request, error) {
	return validateRequest(request, defaults, limits, false)
}

func validateRequest(request Request, defaults Bounds, limits Limits, requireCompleteBounds bool) (Request, error) {
	normalized := normalizeRequest(request)
	mode := ModeLocal
	if normalized.Mode != nil {
		mode = *normalized.Mode
	}
	if err := validateMode(mode); err != nil {
		return Request{}, err
	}
	if normalized.Mode == nil {
		normalized.Mode = new(ModeLocal)
	}
	if mode == ModeLocal {
		if normalized.ChannelStrategy != nil || normalized.ChannelID != nil || normalized.Bounds != nil {
			return Request{}, newContractError(
				ErrStrategyChannelConflict,
				"mode",
				"local mode cannot carry channel strategy, channel id, or bounds",
			)
		}
		return normalized, nil
	}
	if normalized.ChannelStrategy == nil {
		return Request{}, newContractError(ErrStrategyInvalid, "channel_strategy", "live mode requires a strategy")
	}
	if err := validateStrategy(*normalized.ChannelStrategy); err != nil {
		return Request{}, err
	}
	if *normalized.ChannelStrategy == StrategyNamed {
		if normalized.ChannelID == nil || *normalized.ChannelID == "" {
			return Request{}, newContractError(ErrStrategyInvalid, "channel_id", "named strategy requires a channel id")
		}
		if !channelPattern.MatchString(*normalized.ChannelID) {
			return Request{}, newContractError(
				ErrStrategyInvalid,
				"channel_id",
				"must match %q",
				channelPattern.String(),
			)
		}
	} else if normalized.ChannelID != nil {
		return Request{}, newContractError(
			ErrStrategyChannelConflict,
			"channel_id",
			"strategy %q derives its channel and cannot carry channel_id",
			*normalized.ChannelStrategy,
		)
	}
	if requireCompleteBounds && normalized.Bounds == nil {
		return Request{}, newContractError(
			ErrBoundsExceedCeiling,
			"bounds",
			"bounds_required: live mode requires finite bounds",
		)
	}
	resolved, err := ResolveBounds(normalized.Bounds, defaults, limits)
	if err != nil {
		return Request{}, err
	}
	normalized.Bounds = boundsRequestFromBounds(resolved)
	return normalized, nil
}

func normalizeRequest(request Request) Request {
	if request.Mode != nil {
		mode := Mode(strings.TrimSpace(string(*request.Mode)))
		request.Mode = &mode
	}
	if request.ChannelStrategy != nil {
		strategy := ChannelStrategy(strings.TrimSpace(string(*request.ChannelStrategy)))
		request.ChannelStrategy = &strategy
	}
	if request.ChannelID != nil {
		channelID := strings.TrimSpace(*request.ChannelID)
		request.ChannelID = &channelID
	}
	return request
}

func validateMode(mode Mode) error {
	switch mode {
	case ModeLocal, ModeLive:
		return nil
	default:
		return newContractError(ErrStrategyInvalid, "mode", "unsupported value %q; allowed values: local, live", mode)
	}
}

func validateStrategy(strategy ChannelStrategy) error {
	switch strategy {
	case StrategyNamed, StrategyRun, StrategyLoopRun:
		return nil
	default:
		return newContractError(
			ErrStrategyInvalid,
			"channel_strategy",
			"unsupported value %q; allowed values: named, run, loop_run",
			strategy,
		)
	}
}

func boundsRequestFromBounds(bounds Bounds) *BoundsRequest {
	return &BoundsRequest{
		MaxWakes:         new(bounds.MaxWakes),
		MaxWakeWallTime:  new(bounds.MaxWakeWallTime),
		MaxTotalWallTime: new(bounds.MaxTotalWallTime),
		MaxInputTokens:   new(bounds.MaxInputTokens),
		MaxOutputTokens:  new(bounds.MaxOutputTokens),
		MaxWakeDepth:     new(bounds.MaxWakeDepth),
		CoalesceWindow:   new(bounds.CoalesceWindow),
	}
}

func requestBounds(request Request) (Bounds, error) {
	if request.Bounds == nil {
		return Bounds{}, fmt.Errorf("validated live request has no bounds")
	}
	return ResolveBounds(request.Bounds, Bounds{}, Limits{})
}
