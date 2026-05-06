package config

import (
	"path/filepath"
	"testing"
)

func TestEditAgentDefFileCategoryPath(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve category path on skill toggle", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), agentDefName)
		writeFile(t, path, `---
name: coder
provider: claude
category_path:
  - Marketing
  - Sales
skills:
  disabled:
    - old-skill
---

Prompt.
`)

		agent, err := EditAgentDefFile(path, func(agent *AgentDef) error {
			agent.Skills.Disabled = []string{"old-skill", "new-skill"}
			return nil
		})
		if err != nil {
			t.Fatalf("EditAgentDefFile() error = %v", err)
		}
		if !equalStringSlicesForTest(agent.CategoryPath, []string{"Marketing", "Sales"}) {
			t.Fatalf("EditAgentDefFile() CategoryPath = %#v", agent.CategoryPath)
		}

		reloaded, err := LoadAgentDefFile(path)
		if err != nil {
			t.Fatalf("LoadAgentDefFile() error = %v", err)
		}
		if !equalStringSlicesForTest(reloaded.CategoryPath, []string{"Marketing", "Sales"}) {
			t.Fatalf("LoadAgentDefFile() CategoryPath = %#v", reloaded.CategoryPath)
		}
		if !equalStringSlicesForTest(reloaded.Skills.Disabled, []string{"old-skill", "new-skill"}) {
			t.Fatalf("LoadAgentDefFile() Skills.Disabled = %#v", reloaded.Skills.Disabled)
		}
	})
}
