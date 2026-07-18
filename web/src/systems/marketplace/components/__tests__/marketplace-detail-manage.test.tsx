// Suite: Marketplace installed-detail management
// Invariant: Installed detail pages retain the owning kind's management controls after legacy shells are removed.
// Boundary IN: Kind-specific query and mutation view-models.
// Boundary OUT: Transport/cache behavior, owned by each system's hook suites.
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MarketplaceDetailExtensionManage } from "../marketplace-detail-extension-manage";
import { MarketplaceDetailSkillManage } from "../marketplace-detail-skill-manage";

const mocks = vi.hoisted(() => ({
  disableSkill: vi.fn(),
  extensionNavigate: vi.fn(),
  extensionToggle: vi.fn(),
  skillContent: "# Bundled skill\n\nFollow the incident checklist.",
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to }: { children?: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({ activeWorkspaceId: "workspace-a" }),
}));

vi.mock("@/systems/skill", () => ({
  skillSourceTone: () => "neutral",
  useDisableSkill: () => ({ isPending: false, mutate: mocks.disableSkill }),
  useEnableSkill: () => ({ isPending: false, mutate: vi.fn() }),
  useSkill: () => ({ data: { enabled: true } }),
  useSkillContent: (_name: string, _workspace: string, enabled: boolean) => ({
    data: enabled ? mocks.skillContent : undefined,
    error: null,
    isLoading: false,
    refetch: vi.fn(),
  }),
  useSkillShadows: () => ({
    data: {
      shadows: [
        {
          detected_at: "2026-07-18T12:00:00Z",
          path: "/workspace/.agents/skills/bundled-skill/SKILL.md",
          resolved_to_winner: true,
          tier: "workspace",
        },
      ],
    },
    error: null,
    isLoading: false,
  }),
}));

vi.mock("@/systems/extensions", () => ({
  ExtensionProvenanceDialog: () => null,
  RemoveExtensionDialog: () => null,
  VerifiedMark: () => <span>checksum verified</span>,
  useExtensionDetailState: () => ({
    bundles: {
      data: [
        {
          bundle_name: "ops-starter",
          extension_name: "ops-extension",
          id: "activation-ops-starter",
        },
      ],
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    },
    detail: {
      data: {
        extension: {
          bundles: [{ description: "On-call defaults", name: "ops-starter" }],
          diagnostics: [
            {
              id: "healthy",
              message: "Runtime handshake passed.",
              severity: "info",
              title: "Healthy",
            },
          ],
          enabled: true,
          missing_env: ["PAGER_TOKEN"],
          name: "ops-extension",
          provenance: {
            checksum_sha256: "a".repeat(64),
            checksum_verified: true,
            installed_from: "marketplace",
            registry_tier: "verified",
            slug: "agh/ops-extension",
          },
          requires_env: ["PAGER_TOKEN", "REGION"],
          source: "marketplace",
        },
      },
    },
    navigate: mocks.extensionNavigate,
    provenanceOpen: false,
    removeOpen: false,
    setProvenanceOpen: vi.fn(),
    setRemoveOpen: vi.fn(),
    toggle: { isPending: false, mutate: mocks.extensionToggle },
  }),
}));

describe("Marketplace installed-detail management", () => {
  beforeEach(() => vi.clearAllMocks());

  it("Should preserve skill enablement, content, and shadow resolution", async () => {
    const user = userEvent.setup();
    render(<MarketplaceDetailSkillManage name="bundled-skill" />);

    expect(screen.getByText("Enabled")).toBeInTheDocument();
    expect(screen.getByTestId("skill-shadow-table")).toHaveTextContent("workspace");
    await user.click(screen.getByTestId("view-full-content-btn"));
    expect(screen.getByTestId("content-body")).toHaveTextContent("Follow the incident checklist");
    await user.click(screen.getByTestId("skill-enabled-switch"));
    expect(mocks.disableSkill).toHaveBeenCalledWith({
      name: "bundled-skill",
      workspace: "workspace-a",
    });
  });

  it("Should preserve extension enablement, environment, diagnostics, provenance, and activation link", async () => {
    const user = userEvent.setup();
    render(<MarketplaceDetailExtensionManage name="ops-extension" />);

    expect(screen.getByText("PAGER_TOKEN")).toBeInTheDocument();
    expect(screen.getByText("missing")).toBeInTheDocument();
    expect(screen.getByText("Runtime handshake passed.")).toBeInTheDocument();
    expect(screen.getAllByText("verified")).toHaveLength(2);
    expect(screen.getByRole("link", { name: "Open active bundle →" })).toHaveAttribute(
      "href",
      "/marketplace/bundles/activations/$id"
    );
    await user.click(screen.getByTestId("extension-enabled-switch"));
    expect(mocks.extensionToggle).toHaveBeenCalledWith({ enabled: false, name: "ops-extension" });
  });
});
