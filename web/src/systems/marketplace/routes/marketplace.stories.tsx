import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";

import { aghApiMock } from "@/storybook/openapi-msw";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { marketplaceDetails, marketplaceSearchFixture } from "@/systems/marketplace/mocks";
import { skillFixtures } from "@/systems/skill/mocks";

const gitFlowSkill = {
  ...skillFixtures[0]!,
  name: "git-flow",
  dir: "/opt/agh/skills/git-flow",
  version: "1.4.2",
};

const gitFlowShadows = {
  name: "git-flow",
  winner: {
    detected_at: "2026-04-17T16:41:00Z",
    path: "/opt/agh/skills/git-flow/SKILL.md",
    resolved_to_winner: true,
    tier: "workspace",
  },
  shadows: [
    {
      detected_at: "2026-04-17T16:41:00Z",
      path: "/opt/agh/skills/git-flow/SKILL.md",
      resolved_to_winner: true,
      tier: "workspace",
    },
    {
      detected_at: "2026-04-17T16:42:00Z",
      path: "/opt/agh/marketplace/git-flow/SKILL.md",
      resolved_to_winner: false,
      tier: "marketplace",
    },
  ],
};

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/marketplace/routes/Marketplace",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Unified marketplace kind pages: RouteNav kinds, Installed|Marketplace scope, cards-only grids.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const SkillsMarketplace: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/skills"),
  render: () => <StorybookWorkspaceSetup />,
};

export const SkillsInstalled: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/skills?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
};

export const SkillsLoading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/skills"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", async () => {
          await delay("infinite");
          return HttpResponse.json(marketplaceSearchFixture.kinds[0]);
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const McpsMarketplace: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/mcps"),
  render: () => <StorybookWorkspaceSetup />,
};

export const McpsInstalled: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/mcps?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
};

export const ExtensionsMarketplace: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/extensions"),
  render: () => <StorybookWorkspaceSetup />,
};

export const ExtensionsInstalled: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/extensions?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
};

export const BundlesMarketplace: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/bundles"),
  render: () => <StorybookWorkspaceSetup />,
};

export const BundlesInstalled: Story = {
  args: {},
  parameters: appRouteParameters("/marketplace/bundles?tab=installed"),
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailSkillInstalled: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/skill/git-flow"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/skills/{name}", () => HttpResponse.json({ skill: gitFlowSkill })),
        aghApiMock.get("/api/skills/{name}/content", () =>
          HttpResponse.json({
            content:
              "# Git Flow\n\nBranch, review, and land changes using the repository's own checks as the gate.\n",
          })
        ),
        aghApiMock.get("/api/skills/{name}/shadows", () => HttpResponse.json(gitFlowShadows)),
        aghApiMock.post("/api/skills/{name}/enable", () => HttpResponse.json({ ok: true })),
        aghApiMock.post("/api/skills/{name}/disable", () => HttpResponse.json({ ok: true })),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailExtensionInstalled: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/extension/slack-notify"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}/{entry_id}", () =>
          HttpResponse.json({
            ...marketplaceDetails["extension:slack-notify"],
            entry: {
              ...marketplaceDetails["extension:slack-notify"]!.entry,
              installed: true,
              installed_name: "slack-notify",
              installed_version: "1.1.4",
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const DetailMcpInstalled: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/mcp/linear"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}/{entry_id}", () =>
          HttpResponse.json({
            ...marketplaceDetails["mcp:linear"],
            entry: {
              ...marketplaceDetails["mcp:linear"]!.entry,
              installed: true,
              installed_name: "linear",
              installed_version: "1.0.0",
            },
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const SkillsError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/skills"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () =>
          HttpResponse.json({ error: "unreachable" }, { status: 503 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

export const SkillsEmpty: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/marketplace/skills"),
    ...storybookMswParameters({
      marketplace: [
        aghApiMock.get("/api/marketplace/{kind}", () =>
          HttpResponse.json({ items: [], kind: "skill", stale: false, total: 0 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
