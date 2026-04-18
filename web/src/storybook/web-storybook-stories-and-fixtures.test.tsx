import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import { uiMessageFixtures } from "@/systems/session/mocks";
import { workspaceDetailFixture, workspaceFixtures } from "@/systems/workspace/mocks";

describe("storybook story and fixture regressions", () => {
  const fromWeb = (path: string) => resolve(process.cwd(), path);

  it("loads the edited story modules", async () => {
    const modules = await Promise.all([
      import("@/systems/knowledge/components/stories/knowledge-detail-panel.stories"),
      import("@/systems/knowledge/components/stories/knowledge-list-panel.stories"),
      import("@/systems/network/components/stories/network-channels-list-panel.stories"),
      import("@/systems/session/components/stories/copy-button.stories"),
      import("@/systems/session/components/tool-renderers/stories/read-content.stories"),
      import("@/systems/session/components/tool-renderers/stories/search-content.stories"),
    ]);

    expect(modules).toHaveLength(6);

    for (const module of modules) {
      expect(module.default).toBeDefined();
    }
  });

  it("keeps the scoped story and fixture source aligned with the review fixes", async () => {
    const sources = await Promise.all([
      readFile(
        fromWeb("src/systems/knowledge/components/stories/knowledge-detail-panel.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/knowledge/components/stories/knowledge-list-panel.stories.tsx"),
        "utf8"
      ),
      readFile(
        fromWeb("src/systems/network/components/stories/network-channels-list-panel.stories.tsx"),
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
      knowledgeDetailStory,
      knowledgeListStory,
      networkChannelsStory,
      copyButtonStory,
      readContentStory,
      searchContentStory,
      sessionFixturesSource,
      workspaceFixturesSource,
    ] = sources;

    expect(knowledgeDetailStory).toContain(
      'import { KnowledgeDetailPanel } from "@/systems/knowledge/components/knowledge-detail-panel";'
    );
    expect(knowledgeListStory).toContain(
      'import { KnowledgeListPanel } from "@/systems/knowledge/components/knowledge-list-panel";'
    );
    expect(networkChannelsStory).toContain(
      'import { NetworkChannelsListPanel } from "@/systems/network/components/network-channels-list-panel";'
    );
    expect(networkChannelsStory).toContain('import type { ComponentProps } from "react";');
    expect(networkChannelsStory).not.toContain("React.ComponentProps");
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

    for (const path of workspacePaths) {
      expect(path).not.toMatch(/^\/Users\//);
      expect(path).not.toContain("/pedro/");
    }
  });
});
