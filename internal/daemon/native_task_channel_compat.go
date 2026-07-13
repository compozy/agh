package daemon

import (
	"fmt"
	"strings"

	taskpkg "github.com/compozy/agh/internal/task"
	toolspkg "github.com/compozy/agh/internal/tools"
)

func nativeRejectRemovedTaskChannel(id toolspkg.ToolID, channel string) error {
	if strings.TrimSpace(channel) == "" {
		return nil
	}
	return nativeTaskToolError(
		id,
		fmt.Errorf("%w: network_channel is no longer supported", taskpkg.ErrValidation),
	)
}
