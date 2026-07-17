package model

import (
	"fmt"

	"github.com/compozy/agh/internal/network/participation"
)

// NormalizeDirectTaskParticipation returns canonical authored participation
// for an Automation job that materializes a Task directly.
func NormalizeDirectTaskParticipation(
	request *participation.Request,
) (*participation.Request, error) {
	if request == nil {
		return nil, nil
	}
	normalized, err := participation.NormalizeIntent(*request)
	if err != nil {
		return nil, err
	}
	if normalized.ChannelStrategy != nil &&
		*normalized.ChannelStrategy == participation.StrategyLoopRun {
		return nil, fmt.Errorf(
			"%w: direct Automation tasks cannot derive a Loop run channel",
			participation.ErrStrategyInvalid,
		)
	}
	if normalized == (participation.Request{}) {
		return nil, nil
	}
	return &normalized, nil
}
