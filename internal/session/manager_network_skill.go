package session

import (
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/skills/bundled"
)

const networkSkillName = "agh-network"

func appendBundledNetworkSkill(prompt string, channel string) (string, error) {
	if strings.TrimSpace(channel) == "" {
		return strings.TrimSpace(prompt), nil
	}

	content, err := bundled.LoadContent(networkSkillName)
	if err != nil {
		return "", fmt.Errorf("session: load bundled network skill: %w", err)
	}

	return joinPromptSections(prompt, content), nil
}

func joinPromptSections(sections ...string) string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n\n")
}
