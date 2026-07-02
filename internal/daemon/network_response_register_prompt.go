package daemon

import (
	"context"
	"strings"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/session"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

var (
	_ session.PromptProvider       = networkResponseRegisterPromptSectionProvider{}
	_ startupPromptSectionProvider = networkResponseRegisterPromptSectionProvider{}
)

type networkResponseRegisterPromptSectionProvider struct{}

const (
	networkResponseRegisterDefaultLine = "Network response register: reply briefly only when addressed, " +
		"activated, or adding value; promote actionable work to tasks."
	networkResponseRegisterDirectLine = "Network response register: direct replies are brief and actionable; " +
		"promote executable work to tasks."
	networkResponseRegisterThreadLine = "Network response register: in threads, reply briefly only when addressed, " +
		"activated, or adding value; promote executable work to tasks."
	networkResponseRegisterChannelLine = "Network response register: channel replies stay brief; respond only " +
		"when addressed, activated, or adding value."
)

func (networkResponseRegisterPromptSectionProvider) PromptSection(
	context.Context,
	*workspacepkg.ResolvedWorkspace,
) (string, error) {
	return "", nil
}

func (networkResponseRegisterPromptSectionProvider) PromptStartupSection(
	_ context.Context,
	startup session.StartupPromptContext,
	_ aghconfig.AgentDef,
	_ *workspacepkg.ResolvedWorkspace,
) (string, error) {
	return renderNetworkResponseRegisterStartupSection(startup), nil
}

func renderNetworkResponseRegisterStartupSection(startup session.StartupPromptContext) string {
	var builder strings.Builder
	builder.WriteString("# AGH Network Response Register\n\n")
	builder.WriteString(
		"Threads decide and discuss; actionable work is promoted to tasks before execution. " +
			"When network prompts arrive, reply briefly only when addressed, mentioned, activated, or adding value.",
	)
	if channel := strings.TrimSpace(startup.Channel); channel != "" {
		builder.WriteString(" Current channel: `")
		builder.WriteString(channel)
		builder.WriteString("`.")
	}
	return builder.String()
}

func newNetworkResponseRegisterAugmenter() session.PromptInputAugmenter {
	return func(_ context.Context, sess *session.Session, message string) (string, error) {
		if sess == nil || sess.CurrentTurnSource() != session.TurnSourceNetwork {
			return message, nil
		}
		line := networkResponseRegisterTurnLine(sess.CurrentPromptMeta().Network)
		if strings.TrimSpace(line) == "" {
			return message, nil
		}
		return line + "\n\n" + message, nil
	}
}

func networkResponseRegisterTurnLine(meta *acp.PromptNetworkMeta) string {
	if meta == nil {
		return networkResponseRegisterDefaultLine
	}
	normalized := meta.Normalize()
	switch strings.TrimSpace(normalized.Surface) {
	case nativeToolsDirectKey:
		return networkResponseRegisterDirectLine
	case "thread":
		return networkResponseRegisterThreadLine
	default:
		if strings.TrimSpace(normalized.ThreadID) != "" {
			return networkResponseRegisterThreadLine
		}
		if strings.TrimSpace(normalized.DirectID) != "" {
			return networkResponseRegisterDirectLine
		}
		return networkResponseRegisterChannelLine
	}
}
