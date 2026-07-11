import { describe, expect, it } from "vitest";

import type { SessionPayload } from "@/systems/session";

import type { AgentPayload } from "../../types";
import { FIXTURE_AGENT_DEFINITION_DIGEST } from "../../mocks/fixtures";
import {
  agentFleetChipsToFilters,
  agentFleetFiltersToChips,
  buildAgentFleetFilterFields,
} from "../agent-fleet-filters";
import {
  formatAgentFleetAriaLabel,
  formatAgentFleetCardMeta,
  formatAgentFleetMeta,
  formatCategoryMetaSegment,
  projectAgentFleetRows,
  sortAgentsByNameStable,
} from "../agent-fleet-projection";
import { hasActiveAgentFleetFilters, validateAgentsFleetSearch } from "../agent-fleet-search";

function agent(overrides: Partial<AgentPayload> & Pick<AgentPayload, "name">): AgentPayload {
  return {
    provider: "claude",
    prompt: "test",
    origin: "global",
    definition_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
    ...overrides,
  };
}

function session(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess-1",
    agent_name: "coder",
    provider: "claude",
    workspace_id: "ws_alpha",
    workspace_path: "/ws",
    state: "stopped",
    badge: "idle",
    attachable: true,
    available_commands: [],
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T01:00:00Z",
    ...overrides,
  };
}

describe("validateAgentsFleetSearch", () => {
  it("Should parse q, category, status, and view and drop invalid values", () => {
    expect(
      validateAgentsFleetSearch({
        q: "  release  ",
        category: "Engineering / Release",
        status: "idle",
        view: "cards",
      })
    ).toEqual({
      q: "release",
      category: "Engineering / Release",
      status: "idle",
      view: "cards",
    });
    expect(validateAgentsFleetSearch({ status: "running", q: "   ", view: "grid" })).toEqual({
      q: undefined,
      category: undefined,
      status: undefined,
      view: undefined,
    });
  });

  it("Should report active filters when any search facet is set", () => {
    expect(hasActiveAgentFleetFilters({})).toBe(false);
    expect(hasActiveAgentFleetFilters({ q: "x" })).toBe(true);
    expect(hasActiveAgentFleetFilters({ category: "Ops" })).toBe(true);
    expect(hasActiveAgentFleetFilters({ status: "active" })).toBe(true);
    expect(hasActiveAgentFleetFilters({ view: "cards" })).toBe(false);
  });
});

describe("agent fleet projection", () => {
  it("Should sort agents A–Z stably and keep order when sessions flip active", () => {
    const agents = [agent({ name: "zeta" }), agent({ name: "alpha" }), agent({ name: "Beta" })];
    const sorted = sortAgentsByNameStable(agents).map(item => item.name);
    expect(sorted).toEqual(["alpha", "Beta", "zeta"]);

    const idleRows = projectAgentFleetRows({
      agents,
      sessions: [
        session({ id: "1", agent_name: "zeta", state: "stopped" }),
        session({ id: "2", agent_name: "alpha", state: "stopped" }),
      ],
      search: {},
    });
    expect(idleRows.map(row => row.agent.name)).toEqual(["alpha", "Beta", "zeta"]);

    const activeRows = projectAgentFleetRows({
      agents,
      sessions: [
        session({ id: "1", agent_name: "zeta", state: "active" }),
        session({ id: "2", agent_name: "alpha", state: "stopped" }),
      ],
      search: {},
    });
    expect(activeRows.map(row => row.agent.name)).toEqual(["alpha", "Beta", "zeta"]);
    expect(activeRows[2]?.signals?.status).toBe("active");
  });

  it("Should AND-compose q, category, and status filters across name and category segments", () => {
    const agents = [
      agent({
        name: "release-captain",
        category_path: ["Engineering", "Release"],
        provider: "anthropic",
        model: "claude-sonnet-4-5",
      }),
      agent({
        name: "code-reviewer",
        category_path: ["Engineering"],
        provider: "openai",
      }),
      agent({
        name: "triage-bot",
        provider: "openai",
        origin: "workspace",
      }),
    ];
    const sessions = [
      session({ id: "a", agent_name: "release-captain", state: "active" }),
      session({ id: "b", agent_name: "code-reviewer", state: "stopped" }),
      session({ id: "c", agent_name: "triage-bot", state: "stopped" }),
    ];

    const byQuery = projectAgentFleetRows({
      agents,
      sessions,
      search: { q: "release" },
    });
    expect(byQuery.map(row => row.agent.name)).toEqual(["release-captain"]);

    const byCategoryAndIdle = projectAgentFleetRows({
      agents,
      sessions,
      search: { category: "Engineering", status: "idle" },
    });
    expect(byCategoryAndIdle.map(row => row.agent.name)).toEqual(["code-reviewer"]);
  });

  it("Should omit invented status when sessions are unavailable", () => {
    const agents = [
      agent({
        name: "coder",
        diagnostics: [{ error_kind: "parse", message: "bad", path: "AGENT.md" }],
      }),
    ];
    const rows = projectAgentFleetRows({
      agents,
      sessions: null,
      search: { status: "active" },
    });
    expect(rows).toHaveLength(1);
    expect(rows[0]?.signals).toBeNull();
    expect(rows[0]?.sessionsAvailable).toBe(false);
    expect(rows[0]?.ariaLabel).toBe("coder, session status unavailable");
    expect(rows[0]?.hasDiagnostics).toBe(true);
  });

  it("Should render meta with origin and middle-truncate deep categories", () => {
    expect(
      formatAgentFleetMeta(
        agent({
          name: "release-captain",
          category_path: ["Engineering", "Release"],
          provider: "anthropic",
          model: "claude-sonnet-4-5",
          origin: "workspace",
        })
      )
    ).toBe("Engineering / Release · anthropic · claude-sonnet-4-5 · Workspace");

    expect(
      formatAgentFleetCardMeta(
        agent({
          name: "release-captain",
          category_path: ["Engineering", "Release"],
          provider: "anthropic",
          model: "claude-sonnet-4-5",
          origin: "workspace",
        })
      )
    ).toBe("Engineering / Release · claude-sonnet-4-5 · Workspace");

    expect(
      formatAgentFleetCardMeta(
        agent({
          name: "triage-bot",
          provider: "openai",
          origin: "global",
        })
      )
    ).toBe("openai · Global");

    expect(
      formatCategoryMetaSegment(["Engineering", "Platform", "Infrastructure", "Release", "Canary"])
    ).toBe("Engineering / … / Canary");

    expect(
      formatCategoryMetaSegment([
        "ExtremelyLongSingleSegmentCategoryNameThatExceedsTheEllipsisLimitByDesign",
      ])
    ).toMatch(/…/);

    expect(
      formatAgentFleetAriaLabel(
        agent({ name: "release-captain" }),
        {
          status: "active",
          active: 2,
          total: 6,
          failed: 0,
          runtimeSeconds: 0,
          lastActivityMs: null,
        },
        true
      )
    ).toBe("release-captain, Active, 2 of 6 sessions active");
  });

  it("Should attach shared aria and card meta on projected rows", () => {
    const rows = projectAgentFleetRows({
      agents: [
        agent({
          name: "triage-bot",
          provider: "openai",
          origin: "workspace",
        }),
      ],
      sessions: [],
      search: {},
    });
    expect(rows[0]?.cardMeta).toBe("openai · Workspace");
    expect(rows[0]?.ariaLabel).toBe("triage-bot, Idle, 0 of 0 sessions active");
  });
});

describe("agent fleet filters", () => {
  it("Should build category and status fields and bridge chips both ways", () => {
    const fields = buildAgentFleetFilterFields(["Ops", "Engineering / Release"]);
    expect(fields).toEqual([
      {
        key: "category",
        label: "Category",
        type: "select",
        options: [
          { value: "Ops", label: "Ops" },
          { value: "Engineering / Release", label: "Engineering / Release" },
        ],
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: [
          { value: "active", label: "Active" },
          { value: "idle", label: "Idle" },
        ],
      },
    ]);

    expect(
      agentFleetFiltersToChips({
        category: "Ops",
        status: "active",
      })
    ).toEqual([
      {
        id: "agent-fleet-filter-category",
        field: "category",
        operator: "is",
        values: ["Ops"],
      },
      {
        id: "agent-fleet-filter-status",
        field: "status",
        operator: "is",
        values: ["active"],
      },
    ]);

    expect(
      agentFleetChipsToFilters([
        {
          id: "agent-fleet-filter-category",
          field: "category",
          operator: "is",
          values: ["  Ops  "],
        },
        {
          id: "agent-fleet-filter-status",
          field: "status",
          operator: "is",
          values: ["idle"],
        },
      ])
    ).toEqual({ category: "Ops", status: "idle" });

    expect(
      agentFleetChipsToFilters([
        {
          id: "agent-fleet-filter-status",
          field: "status",
          operator: "is",
          values: ["running"],
        },
      ])
    ).toEqual({ category: undefined, status: undefined });
  });
});
