import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { render, screen, within } from "@testing-library/react";

import { UIProvider } from "@agh/ui";

import { DesignSystemShowcase, SECTIONS, TOKEN_GROUPS } from "@/components/design-system-showcase";

const SHOWCASE_PATH = resolve(__dirname, "../design-system-showcase.tsx");
const TOKENS_PATH = resolve(__dirname, "../../../../packages/ui/src/tokens.css");
const ROUTE_PATH = resolve(__dirname, "../../routes/design-system.tsx");
const COMPONENTS_DIR = resolve(__dirname, "..");

const SHOWCASE_SOURCE = readFileSync(SHOWCASE_PATH, "utf8");
const TOKENS_SOURCE = readFileSync(TOKENS_PATH, "utf8");
const ROUTE_SOURCE = readFileSync(ROUTE_PATH, "utf8");

function renderShowcase() {
  return render(
    <UIProvider reducedMotion="always">
      <DesignSystemShowcase />
    </UIProvider>
  );
}

/**
 * Tokens that the showcase intentionally surfaces as discrete swatches. Shadcn
 * theme aliases (`--background`, `--primary`, ...) re-map to these AGH tokens
 * and are covered by the primitives themselves rather than the swatch wall.
 */
const SHADCN_ALIASES: ReadonlySet<string> = new Set([
  "--background",
  "--foreground",
  "--card",
  "--card-foreground",
  "--popover",
  "--popover-foreground",
  "--primary",
  "--primary-foreground",
  "--secondary",
  "--secondary-foreground",
  "--muted-foreground",
  "--accent-foreground",
  "--destructive",
  "--destructive-foreground",
  "--border",
  "--input",
  "--ring",
  "--chart-1",
  "--chart-2",
  "--chart-3",
  "--chart-4",
  "--chart-5",
  "--radius",
  "--sidebar-foreground",
  "--sidebar-primary",
  "--sidebar-primary-foreground",
  "--sidebar-accent",
  "--sidebar-accent-foreground",
  "--sidebar-border",
  "--sidebar-ring",
]);

const COMPONENT_GEOMETRY_TOKENS: ReadonlySet<string> = new Set([
  "--height-mono-badge",
  "--height-pill-group-segment-sm",
  "--size-pill-group-badge",
  "--space-pill-group-track-gap",
  "--space-pill-group-track-padding",
  "--space-pill-group-segment-sm-x",
  "--space-pill-group-segment-md-x",
  "--space-pill-group-badge-x",
  "--text-pill-group-badge",
  "--shadow-overlay",
  "--highlight",
  "--radius-chip",
  "--radius-mono-badge",
  "--radius-icon-well",
  "--duration-fast",
  "--duration-base",
  "--duration-slow",
  "--ease-out",
  "--ease-in-out",
]);

function extractAghTokens(source: string): string[] {
  const tokens = new Set<string>();
  const rootMatch = source.match(/:root\s*{([\s\S]*?)}/);
  if (!rootMatch) return [];
  const body = rootMatch[1];
  for (const line of body.split("\n")) {
    const match = line.match(/^\s*(--[a-z0-9-]+)\s*:\s*(.+?);\s*(?:\/\*[^*]*\*\/)?\s*$/i);
    if (!match) continue;
    const [, name, rawValue] = match;
    const value = rawValue.trim();
    if (value.startsWith("var(")) continue;
    if (SHADCN_ALIASES.has(name)) continue;
    if (COMPONENT_GEOMETRY_TOKENS.has(name)) continue;
    tokens.add(name);
  }
  return [...tokens];
}

function extractTokenValueMap(source: string): Map<string, string> {
  const values = new Map<string, string>();
  const rootMatch = source.match(/:root\s*{([\s\S]*?)}/);
  if (!rootMatch) return values;
  const body = rootMatch[1];
  for (const line of body.split("\n")) {
    const match = line.match(/^\s*(--[a-z0-9-]+)\s*:\s*(.+?);\s*(?:\/\*[^*]*\*\/)?\s*$/i);
    if (!match) continue;
    const [, name, rawValue] = match;
    values.set(name, rawValue.trim());
  }
  return values;
}

function normalizeTokenValue(value: string): string {
  return value.trim().toLowerCase();
}

describe("DesignSystemShowcase", () => {
  describe("rendering", () => {
    it("renders the page header, filter toolbar, and search input", () => {
      renderShowcase();
      expect(screen.getByTestId("design-system-showcase")).toBeInTheDocument();
      expect(screen.getByText("AGH design system")).toBeInTheDocument();
      expect(screen.getByRole("toolbar", { name: /showcase filters/i })).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/search primitives/i)).toBeInTheDocument();
    });

    it("links the top-level DESIGN.md shortcut to the spec", () => {
      renderShowcase();
      const link = screen.getByTestId("showcase-open-design-md");
      expect(link.getAttribute("href")).toBe("https://github.com/compozy/agh/blob/main/DESIGN.md");
    });

    it("renders a dedicated section for every primitive grouping", () => {
      renderShowcase();
      for (const section of SECTIONS) {
        expect(screen.getByTestId(`section-${section.id}`)).toBeInTheDocument();
      }
    });

    it("section headers link to the DESIGN.md anchor for that group", () => {
      renderShowcase();
      for (const section of SECTIONS) {
        const link = screen.getByTestId(`section-link-${section.id}`);
        expect(link.getAttribute("href")).toBe(
          `https://github.com/compozy/agh/blob/main/DESIGN.md${section.anchor}`
        );
        expect(link.getAttribute("data-section-id")).toBe(section.id);
        expect(link.getAttribute("data-section-anchor")).toBe(section.anchor);
      }
    });

    it("renders button and pill primitives", () => {
      renderShowcase();
      const buttons = screen.getByTestId("section-buttons");
      expect(within(buttons).getByRole("button", { name: "Primary" })).toBeInTheDocument();
      expect(within(buttons).getByRole("button", { name: "Secondary" })).toBeInTheDocument();
      expect(within(buttons).getByRole("button", { name: "Destructive" })).toBeInTheDocument();
      expect(within(buttons).getByRole("button", { name: "Outline" })).toBeInTheDocument();
      expect(within(buttons).getByText("Action")).toBeInTheDocument();
      expect(within(buttons).getByText("Stable")).toBeInTheDocument();
    });

    it("renders input, select, toggle, switch, and search primitives", () => {
      renderShowcase();
      const inputs = screen.getByTestId("section-inputs");
      expect(within(inputs).getByLabelText("Display name")).toBeInTheDocument();
      expect(within(inputs).getByLabelText("Notes")).toBeInTheDocument();
      expect(within(inputs).getByLabelText("Environment")).toBeInTheDocument();
      expect(within(inputs).getByPlaceholderText(/filter sessions/i)).toBeInTheDocument();
      expect(within(inputs).getByRole("switch")).toBeInTheDocument();
      expect(within(inputs).getByRole("button", { name: "Tasks" })).toBeInTheDocument();
      expect(within(inputs).getByRole("button", { name: "Sessions" })).toBeInTheDocument();
    });

    it("renders status primitives, metric cards, mono badges, and kind chips", () => {
      renderShowcase();
      const status = screen.getByTestId("section-status");
      expect(within(status).getByText("Active sessions")).toBeInTheDocument();
      expect(within(status).getByText("RUNNING")).toBeInTheDocument();
      const kindChips = status.querySelectorAll('[data-slot="kind-chip"]');
      expect(kindChips.length).toBeGreaterThanOrEqual(7);
      expect(status.querySelectorAll('[data-slot="connection-indicator"]').length).toBe(3);
    });

    it("renders Alert + Empty feedback primitives", () => {
      renderShowcase();
      const feedback = screen.getByTestId("section-feedback");
      expect(within(feedback).getAllByRole("alert").length).toBe(2);
      expect(feedback.querySelector('[data-slot="empty"]')).toBeInTheDocument();
    });

    it("renders Dialog, Sheet, Popover, Tooltip, Dropdown, Tabs, Accordion, Collapsible triggers", () => {
      renderShowcase();
      expect(screen.getByTestId("showcase-dialog-trigger")).toBeInTheDocument();
      expect(screen.getByTestId("showcase-sheet-trigger")).toBeInTheDocument();
      expect(screen.getByTestId("showcase-popover-trigger")).toBeInTheDocument();
      expect(screen.getByTestId("showcase-tooltip-trigger")).toBeInTheDocument();
      expect(screen.getByTestId("showcase-menu-trigger")).toBeInTheDocument();
      expect(screen.getByTestId("showcase-collapsible-trigger")).toBeInTheDocument();
      expect(screen.getByRole("tab", { name: /overview/i })).toBeInTheDocument();
    });

    it("renders CodeBlock + ChatMessageBubble + ToolCallCard in the session shells block", () => {
      renderShowcase();
      const block = screen.getByTestId("section-code-chat");
      expect(block.querySelector('[data-slot="code-block"]')).toBeInTheDocument();
      expect(block.querySelectorAll('[data-slot="chat-message"]').length).toBeGreaterThanOrEqual(4);
      expect(block.querySelector('[data-slot="tool-call-card"]')).toBeInTheDocument();
    });

    it("renders Sidebar + SplitPane layout primitives", () => {
      renderShowcase();
      const layout = screen.getByTestId("section-layout");
      expect(layout.querySelector('[data-slot="sidebar"]')).toBeInTheDocument();
      expect(layout.querySelector('[data-slot="split-pane"]')).toBeInTheDocument();
    });
  });

  describe("token swatch wall", () => {
    it("renders one group per token category defined in TOKEN_GROUPS", () => {
      renderShowcase();
      for (const group of TOKEN_GROUPS) {
        expect(screen.getByTestId(`token-group-${group.id}`)).toBeInTheDocument();
      }
    });

    it("renders a swatch card for every AGH token in packages/ui/src/tokens.css", () => {
      renderShowcase();
      const aghTokens = extractAghTokens(TOKENS_SOURCE);
      expect(aghTokens.length).toBeGreaterThan(20);
      const rendered = new Set(
        TOKEN_GROUPS.flatMap(group => group.swatches.map(swatch => swatch.token))
      );
      const missing = aghTokens.filter(token => !rendered.has(token));
      expect(missing).toEqual([]);
      for (const token of aghTokens) {
        expect(
          document.querySelector(`[data-token="${token}"]`),
          `expected swatch for ${token}`
        ).not.toBeNull();
      }
    });

    it("renders the expected range of color, radius, and motion swatches", () => {
      renderShowcase();
      const kinds = new Set(
        TOKEN_GROUPS.flatMap(group => group.swatches.map(swatch => swatch.kind))
      );
      expect(kinds.has("color")).toBe(true);
      expect(kinds.has("radius")).toBe(true);
      expect(kinds.has("duration")).toBe(true);
      expect(kinds.has("easing")).toBe(true);
      expect(kinds.has("tracking")).toBe(true);
    });

    it("keeps rendered swatch metadata synchronized with tokens.css values", () => {
      const tokenValues = extractTokenValueMap(TOKENS_SOURCE);
      const mismatches = TOKEN_GROUPS.flatMap(group =>
        group.swatches.flatMap(swatch => {
          const expected = tokenValues.get(swatch.token);
          if (!expected) {
            return [];
          }
          return normalizeTokenValue(expected) === normalizeTokenValue(swatch.value)
            ? []
            : [`${swatch.token}: showcase=${swatch.value} tokens=${expected}`];
        })
      );

      expect(mismatches).toEqual([]);
    });
  });

  describe("file content contract", () => {
    it("imports only from @agh/ui + lucide-react + react + the local helpers that compose @agh/ui Pill primitives", () => {
      const specifierRegex = /from\s+["']([^"']+)["']/g;
      const sources = new Set<string>();
      for (const match of SHOWCASE_SOURCE.matchAll(specifierRegex)) {
        sources.add(match[1]);
      }
      expect(sources.has("@agh/ui")).toBe(true);
      expect(sources.has("lucide-react")).toBe(true);
      expect(sources.has("react")).toBe(true);
      const allowed = new Set(["@agh/ui", "lucide-react", "react", "@/systems/network"]);
      const forbidden = [...sources].filter(specifier => {
        if (allowed.has(specifier)) return false;
        return true;
      });
      expect(forbidden).toEqual([]);
    });

    it("never imports from the deleted design-system folder", () => {
      expect(SHOWCASE_SOURCE).not.toMatch(/@\/components\/design-system(\/|["'])/);
      expect(SHOWCASE_SOURCE).not.toMatch(/@\/components\/ui\//);
    });

    it("the design-system folder is deleted", () => {
      expect(() => readFileSync(resolve(COMPONENTS_DIR, "design-system/index.ts"))).toThrow();
      expect(() =>
        readFileSync(resolve(COMPONENTS_DIR, "design-system/design-system-showcase.tsx"))
      ).toThrow();
    });

    it("the /design-system route imports the showcase from the flat location", () => {
      expect(ROUTE_SOURCE).toContain('from "@/components/design-system-showcase"');
      expect(ROUTE_SOURCE).not.toMatch(/@\/components\/design-system(\/|["'])/);
    });
  });
});
