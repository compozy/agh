import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import {
  multiHunkEditToolMessageFixture,
  sessionFixtures,
  uiMessageFixtures,
} from "@/systems/session/mocks";
import { workspaceDetailFixture, workspaceFixtures } from "@/systems/workspace/mocks";

describe("storybook story and fixture regressions", () => {
  const testDir = dirname(fileURLToPath(import.meta.url));
  const webRoot = resolve(testDir, "../..");
  const fromWeb = (path: string) => resolve(webRoot, path);

  it("loads the edited story modules", { timeout: 15_000 }, async () => {
    const modules = await Promise.all([
      import("@/systems/knowledge/components/stories/knowledge-detail-panel.stories"),
      import("@/systems/knowledge/components/stories/knowledge-list-panel.stories"),
      import("@/systems/network/components/stories/network-create-channel-dialog.stories"),
      import("@/systems/network/storybook").then(module => module.networkWorkspaceShellStories),
      import("@/routes/_app/stories/-network.stories"),
      import("@/systems/automation/components/stories/automation-editor-dialog.stories"),
      import("@/systems/session/components/stories/copy-button.stories"),
      import("@/systems/session/components/tool-renderers/stories/read-content.stories"),
      import("@/systems/session/components/tool-renderers/stories/search-content.stories"),
      import("@/routes/_app/stories/-agents.$name.stories"),
      import("@/systems/agent/components/stories/agent-info-panel.stories"),
      import("@/systems/agent/components/stories/agent-sessions-list.stories"),
      import("@/systems/agent/components/stories/agent-stats-grid.stories"),
    ]);

    expect(modules).toHaveLength(13);

    for (const module of modules) {
      expect(module.default).toBeDefined();
    }
  });

  it("keeps the scoped story and fixture source aligned with the review fixes", async () => {
    const sources = await Promise.all([
      readFile(
        resolve(webRoot, "../packages/ui/src/components/stories/collapsible.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/knowledge/components/stories/knowledge-detail-panel.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/knowledge/components/stories/knowledge-list-panel.stories.tsx"),
        "utf8"
      ),
      readFile(fromWeb("src/styles.css"), "utf8"),
      readFile(
        fromWeb("src/systems/network/components/stories/network-create-channel-dialog.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/network/components/stories/network-workspace-shell.stories.tsx"),
        "utf8"
      ),
      readFile(fromWeb("src/routes/_app/stories/-network.stories.tsx"), "utf8"),
      readFile(
        fromWeb("src/systems/automation/components/stories/automation-editor-dialog.stories.tsx"),
        "utf8"
      ),
      readFile(fromWeb("src/systems/session/components/stories/copy-button.stories.tsx"), "utf8"),
      readFile(
        fromWeb("src/systems/session/components/tool-renderers/stories/read-content.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/session/components/tool-renderers/stories/search-content.stories.tsx"),
        "utf8"
      ),
      readFile(fromWeb("src/systems/session/mocks/fixtures.ts"), "utf8"),
      readFile(fromWeb("src/systems/workspace/mocks/fixtures.ts"), "utf8"),
    ]);

    const [
      collapsibleStory,
      knowledgeDetailStory,
      knowledgeListStory,
      stylesSource,
      networkCreateDialogStory,
      networkWorkspaceShellStory,
      networkRouteStory,
      automationEditorDialogStory,
      copyButtonStory,
      readContentStory,
      searchContentStory,
      sessionFixturesSource,
      workspaceFixturesSource,
    ] = sources;

    expect(collapsibleStory).toContain("group-data-[panel-open]/collapsible-trigger:rotate-180");
    expect(collapsibleStory).not.toContain("data-[panel-open]:rotate-180");
    expect(knowledgeDetailStory).toContain(
      'import { KnowledgeDetailPanel } from "@/systems/knowledge/components/knowledge-detail-panel";'
    );
    expect(knowledgeListStory).toContain(
      'import { KnowledgeListPanel } from "@/systems/knowledge/components/knowledge-list-panel";'
    );
    expect(knowledgeListStory).toContain(
      'import { knowledgeMemoryKey } from "@/systems/knowledge";'
    );
    expect(knowledgeListStory).toContain("memory-item-${knowledgeMemoryKey(defaultMemories[2])}");
    expect(stylesSource).toContain("animation-duration: var(--duration-base);");
    expect(stylesSource).toContain("animation-timing-function: var(--ease-out);");
    expect(networkCreateDialogStory).toContain(
      'import { NetworkCreateChannelDialog } from "../network-create-channel-dialog";'
    );
    expect(networkCreateDialogStory).toContain(
      'purpose: "Coordinate release handoffs and deploy verification.",'
    );
    expect(networkWorkspaceShellStory).toContain("NetworkWorkspaceShell");
    expect(networkWorkspaceShellStory).toContain("networkChannelMessagesFixture");
    expect(networkRouteStory).toContain('"routes/app/network"');
    expect(networkRouteStory).toContain("storybookMswParameters");
    expect(automationEditorDialogStory).toContain(
      'import { AutomationEditorDialog } from "@/systems/automation/components/automation-editor-dialog";'
    );
    expect(automationEditorDialogStory).not.toContain("useAutomationPage");
    expect(automationEditorDialogStory).not.toContain("page.editorDialogProps");
    expect(copyButtonStory).toContain(
      'import { CopyButton } from "@/systems/session/components/copy-button";'
    );
    expect(copyButtonStory).toContain("const hadClipboard = Boolean(navigator.clipboard);");
    expect(copyButtonStory).toContain("const originalWriteText = navigator.clipboard.writeText;");
    expect(copyButtonStory).toContain("navigator.clipboard.writeText = async () => undefined;");
    expect(readContentStory).toContain(
      'import { ReadContent } from "@/systems/session/components/tool-renderers/read-content";'
    );
    expect(readContentStory).toContain('import type { ReactNode } from "react";');
    expect(readContentStory).not.toContain("React.ReactNode");
    expect(searchContentStory).toContain(
      'import { SearchContent } from "@/systems/session/components/tool-renderers/search-content";'
    );
    expect(sessionFixturesSource).toContain('} from "@/systems/session/types";');
    expect(sessionFixturesSource).toContain('id: "tool_bash_result"');
    expect(workspaceFixturesSource).toContain(
      'import type { WorkspaceDetailPayload, WorkspacePayload } from "@/systems/workspace/types";'
    );
    expect(workspaceFixturesSource).toContain('root_dir: "/workspaces/home"');
  });

  it("keeps UI message fixture ids unique and workspace paths neutral", () => {
    const ids = uiMessageFixtures.map(message => message.id);
    const sessionPaths = sessionFixtures
      .map(session => session.workspace_path)
      .filter((value): value is string => typeof value === "string");
    const skillDirs =
      workspaceDetailFixture.skills?.flatMap(skill =>
        typeof skill === "object" &&
        skill !== null &&
        "dir" in skill &&
        typeof skill.dir === "string"
          ? [skill.dir]
          : []
      ) ?? [];
    const workspacePaths = [
      ...workspaceFixtures.flatMap(workspace => [workspace.root_dir, ...workspace.add_dirs]),
      workspaceDetailFixture.workspace.root_dir,
      ...workspaceDetailFixture.workspace.add_dirs,
      ...skillDirs,
    ];

    expect(new Set(ids).size).toBe(ids.length);

    for (const path of [...workspacePaths, ...sessionPaths]) {
      expect(path).not.toMatch(/^\/Users\//);
      expect(path).not.toContain("/pedro/");
    }
  });

  it("keeps the multi-hunk edit fixture truthful", () => {
    const oldString = String(multiHunkEditToolMessageFixture.toolInput?.old_string ?? "");
    const newString = String(multiHunkEditToolMessageFixture.toolInput?.new_string ?? "");

    expect(oldString).not.toEqual(newString);
    expect(oldString).toContain("export const Default = {};");
    expect(newString).toContain("export const Default = { args: { state: 'default' } };");
    expect(oldString).toContain("export const Streaming = {};");
    expect(newString).toContain("export const Streaming = { args: { state: 'streaming' } };");
  });
});
