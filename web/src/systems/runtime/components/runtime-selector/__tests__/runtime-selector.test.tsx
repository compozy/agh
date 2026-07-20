import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { FAVORITES_STORAGE_KEY, RECENTS_LIMIT, RECENTS_STORAGE_KEY } from "../favorites";
import { registerCommandK, triggerVisible } from "../command-k-registry";
import { IntensityMeter } from "../intensity-meter";
import { runtimeModelKey } from "../model-key";
import { ReasoningBar } from "../reasoning-bar";
import { RuntimeSelector, type RuntimeSelectorProps } from "../runtime-selector";
import { useRuntimeFavorites } from "../use-runtime-favorites";
import {
  reasoningEffortPosition,
  REASONING_EFFORT_ORDER,
  resolveReasoningState,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
} from "../types";

// ---------------------------------------------------------------------------
// Fixtures + harness
// ---------------------------------------------------------------------------

const codexProvider: RuntimeProviderOption = {
  id: "codex",
  name: "Codex",
  runtime_provider: "codex",
};
const claudeProvider: RuntimeProviderOption = {
  id: "claude",
  name: "Claude",
  runtime_provider: "claude",
};

// Browser-local favorites/recents must not leak across cases (each seeds its own).
beforeEach(() => window.localStorage.clear());

function model(id: string, overrides: Partial<RuntimeModelOption> = {}): RuntimeModelOption {
  return {
    id,
    provider: "codex",
    name: id,
    efforts: [],
    availability: "live",
    curated: true,
    ...overrides,
  };
}

interface HarnessOptions {
  value: RuntimeSelectorValue;
  providers?: RuntimeProviderOption[];
  models?: RuntimeModelOption[];
  props?: Partial<RuntimeSelectorProps>;
}

function renderSelector({
  value,
  providers = [codexProvider],
  models = [],
  props,
}: HarnessOptions) {
  const onChange = vi.fn<(next: RuntimeSelectorValue) => void>();
  function Harness() {
    const [current, setCurrent] = useState<RuntimeSelectorValue>(value);
    return (
      <UIProvider reducedMotion="always">
        <RuntimeSelector
          {...props}
          value={current}
          onChange={next => {
            onChange(next);
            setCurrent(next);
          }}
          providers={providers}
          models={models}
          triggerTestId="rt-trigger"
        />
      </UIProvider>
    );
  }
  const result = render(<Harness />);
  return { onChange, unmount: result.unmount };
}

async function openVia(
  user: ReturnType<typeof userEvent.setup>,
  focus: "provider" | "model" | "reasoning"
) {
  const segment = document.querySelector<HTMLElement>(`button[data-focus="${focus}"]`);
  if (!segment) throw new Error(`Missing trigger segment "${focus}"`);
  await user.click(segment);
  return screen.findByTestId("runtime-selector-popup");
}

function row(id: string): HTMLElement {
  const el = document.querySelector<HTMLElement>(`[data-model="${id}"]`);
  if (!el) throw new Error(`Missing model row "${id}"`);
  return el;
}

function optionOrder(): string[] {
  return screen
    .getAllByRole("option")
    .map(node => node.getAttribute("data-model") ?? "")
    .filter(Boolean);
}

describe("RuntimeSelector read-only contract", () => {
  it("Should keep controls focusable and aria-disabled while preventing the popup from opening", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "gpt-a", reasoning_effort: "" },
      models: [model("gpt-a")],
      props: { readOnly: true },
    });

    const trigger = screen.getByTestId("rt-trigger");
    const provider = within(trigger).getByRole("button", { name: /^Provider:/ });
    expect(trigger).toHaveAttribute("aria-disabled", "true");
    expect(provider).toHaveAttribute("aria-disabled", "true");
    expect(provider).not.toBeDisabled();
    provider.focus();
    expect(provider).toHaveFocus();
    await user.click(provider);
    expect(screen.queryByTestId("runtime-selector-popup")).toBeNull();
    expect(onChange).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// Reset-on-switch
// ---------------------------------------------------------------------------

describe("RuntimeSelector reasoning reset-on-switch", () => {
  it("Should reset reasoning_effort to '' when the picked model cannot honor the current effort", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "with-high", reasoning_effort: "high" },
      models: [
        model("with-high", { name: "With High", efforts: ["low", "medium", "high"] }),
        model("no-high", { name: "No High", efforts: ["low", "medium"] }),
      ],
    });

    await openVia(user, "model");
    await user.click(row("no-high"));

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "no-high",
      reasoning_effort: "",
    });
  });

  it("Should keep the current reasoning_effort when the picked model still supports it", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "with-high", reasoning_effort: "high" },
      models: [
        model("with-high", { name: "With High", efforts: ["low", "medium", "high"] }),
        model("also-high", { name: "Also High", efforts: ["medium", "high", "xhigh"] }),
      ],
    });

    await openVia(user, "model");
    await user.click(row("also-high"));

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "also-high",
      reasoning_effort: "high",
    });
  });
});

// ---------------------------------------------------------------------------
// Reasoning trigger segment + footer
// ---------------------------------------------------------------------------

describe("RuntimeSelector reasoning trigger + footer", () => {
  it("Should hide the reasoning trigger segment when the selected model exposes no efforts", () => {
    renderSelector({
      value: { provider: "codex", model: "plain", reasoning_effort: "" },
      models: [model("plain", { name: "Plain", efforts: [] })],
    });

    expect(document.querySelector('button[data-focus="provider"]')).toBeInTheDocument();
    expect(document.querySelector('button[data-focus="model"]')).toBeInTheDocument();
    expect(document.querySelector('button[data-focus="reasoning"]')).not.toBeInTheDocument();
  });

  it("Should render a hollow meter with the 'Default' label when reasoning is unset", () => {
    renderSelector({
      value: { provider: "codex", model: "leveled", reasoning_effort: "" },
      models: [model("leveled", { name: "Leveled", efforts: ["low", "medium", "high"] })],
    });

    const reasoningSegment = document.querySelector<HTMLElement>('button[data-focus="reasoning"]');
    expect(reasoningSegment).toBeInTheDocument();
    expect(reasoningSegment).toHaveTextContent("Default");
    const meter = reasoningSegment?.querySelector('[data-slot="runtime-intensity-meter"]');
    expect(meter).not.toBeNull();
    // Semantic state (hollow, zero bars filled), not the visual class.
    expect(meter).toHaveAttribute("data-hollow", "true");
    expect(meter).toHaveAttribute("data-position", "0");
  });

  it("Should render the Default button plus only the model's efforts in canonical order", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "leveled", reasoning_effort: "" },
      models: [model("leveled", { name: "Leveled", efforts: ["high", "low", "xhigh", "medium"] })],
    });

    await openVia(user, "reasoning");

    const strip = screen.getByTestId("runtime-selector-reasoning");
    const buttons = Array.from(strip.querySelectorAll<HTMLElement>("button[data-rz]"));
    expect(buttons.map(button => button.getAttribute("data-rz"))).toEqual([
      "",
      "low",
      "medium",
      "high",
      "xhigh",
    ]);
    expect(within(strip).getByText("Default")).toBeInTheDocument();
  });

  it("Should tag the reasoning footer source badge as ACP for acp-sourced reasoning", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "acp-model", reasoning_effort: "" },
      models: [
        model("acp-model", {
          name: "ACP Model",
          efforts: ["low", "high"],
          reasoning_source: "acp",
        }),
      ],
    });

    await openVia(user, "reasoning");
    expect(
      within(screen.getByTestId("runtime-selector-reasoning")).getByText("ACP")
    ).toBeInTheDocument();
  });

  it("Should tag the reasoning footer source badge as CATALOG for catalog-sourced reasoning", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "cat-model", reasoning_effort: "" },
      models: [
        model("cat-model", {
          name: "Catalog Model",
          efforts: ["low", "high"],
          reasoning_source: "catalog",
        }),
      ],
    });

    await openVia(user, "reasoning");
    expect(
      within(screen.getByTestId("runtime-selector-reasoning")).getByText("CATALOG")
    ).toBeInTheDocument();
  });

  it("Should emit the chosen effort when a reasoning footer level is clicked", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "leveled", reasoning_effort: "" },
      models: [model("leveled", { name: "Leveled", efforts: ["low", "medium", "high"] })],
    });

    await openVia(user, "reasoning");
    await user.click(
      screen.getByTestId("runtime-selector-reasoning").querySelector('button[data-rz="high"]')!
    );

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "leveled",
      reasoning_effort: "high",
    });
  });
});

// ---------------------------------------------------------------------------
// Needs-auth provider
// ---------------------------------------------------------------------------

describe("RuntimeSelector needs-auth provider", () => {
  const authProvider: RuntimeProviderOption = {
    id: "codex",
    name: "Codex",
    runtime_provider: "codex",
    needs_auth: true,
  };
  const authModels = [
    model("gpt-a", {
      name: "GPT A",
      availability: "unavailable",
      disabled: true,
      disabled_reason: "Sign in",
    }),
    model("gpt-b", {
      name: "GPT B",
      availability: "unavailable",
      disabled: true,
      disabled_reason: "Sign in",
    }),
  ];

  it("Should surface the sign-in warning on the trigger for a needs-auth provider", () => {
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [authProvider],
      models: authModels,
    });

    expect(screen.getByRole("img", { name: "Provider needs sign in" })).toBeInTheDocument();
  });

  it("Should expose a model-unavailable warning to assistive tech on the trigger", () => {
    renderSelector({
      value: { provider: "codex", model: "gone", reasoning_effort: "" },
      models: [
        model("gone", {
          name: "Gone",
          availability: "unavailable",
          disabled: true,
          disabled_reason: "Unavailable",
        }),
      ],
    });

    expect(screen.getByRole("img", { name: "Model unavailable" })).toBeInTheDocument();
  });

  it("Should dim the rail item and disable every row with a reason for a needs-auth provider", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [authProvider],
      models: authModels,
    });

    await openVia(user, "model");

    expect(document.querySelector('[data-rail="codex"]')).toHaveAttribute("data-dim", "true");
    expect(row("gpt-a")).toHaveAttribute("data-disabled", "true");
    expect(row("gpt-a")).toHaveTextContent("Sign in");
  });

  it("Should not emit onChange when a disabled row is clicked", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [authProvider],
      models: authModels,
    });

    await openVia(user, "model");
    await user.click(row("gpt-a"));

    expect(onChange).not.toHaveBeenCalled();
  });
});

describe("RuntimeSelector group availability", () => {
  it("Should label a group Unavailable when every model is unavailable", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [
        model("gone-a", {
          name: "Gone A",
          availability: "unavailable",
          disabled: true,
          disabled_reason: "Offline",
        }),
        model("gone-b", {
          name: "Gone B",
          availability: "unavailable",
          disabled: true,
          disabled_reason: "Offline",
        }),
      ],
    });

    await openVia(user, "model");

    const groupHead = screen
      .getByTestId("runtime-selector-list")
      .querySelector('[role="presentation"]');
    expect(groupHead).toHaveTextContent("Unavailable");
    expect(groupHead).not.toHaveTextContent("Live");
  });

  it("Should keep Stale ahead of Unavailable when any model is stale", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [
        model("stale-a", { name: "Stale A", availability: "stale" }),
        model("gone-b", {
          name: "Gone B",
          availability: "unavailable",
          disabled: true,
          disabled_reason: "Offline",
        }),
      ],
    });

    await openVia(user, "model");

    const groupHead = screen
      .getByTestId("runtime-selector-list")
      .querySelector('[role="presentation"]');
    expect(groupHead).toHaveTextContent("Stale");
    expect(groupHead).not.toHaveTextContent("Unavailable");
  });
});

// ---------------------------------------------------------------------------
// Custom model ID
// ---------------------------------------------------------------------------

describe("RuntimeSelector custom model id", () => {
  it("Should commit an unknown query as a custom model id when the custom button is clicked", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("known", { name: "Known" })],
    });

    await openVia(user, "model");
    fireEvent.change(screen.getByTestId("runtime-selector-search"), {
      target: { value: "custom-unknown-model" },
    });
    await user.click(await screen.findByTestId("runtime-selector-custom"));

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "custom-unknown-model",
      reasoning_effort: "",
    });
  });

  it("Should commit an unknown query as a custom model id on Enter", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("known", { name: "Known" })],
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    fireEvent.change(search, { target: { value: "typed-custom-id" } });
    fireEvent.keyDown(search, { key: "Enter" });

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "typed-custom-id",
      reasoning_effort: "",
    });
  });

  it("Should not emit a custom id when the query is empty", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("known", { name: "Known" })],
    });

    await openVia(user, "model");
    const custom = await screen.findByTestId("runtime-selector-custom");
    expect(custom).toHaveTextContent("Use an exact custom model ID…");
    await user.click(custom);

    expect(onChange).not.toHaveBeenCalled();
  });

  it("Should not offer a custom commit when no explicit provider is active", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      // All rail active with no selected provider → there is no explicit target.
      value: { provider: "", model: "", reasoning_effort: "" },
      providers: [codexProvider, claudeProvider],
      models: [model("known", { name: "Known" })],
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    fireEvent.change(search, { target: { value: "custom-unknown-model" } });

    // No provider target → the affordance is hidden and Enter emits nothing.
    await waitFor(() =>
      expect(screen.queryByTestId("runtime-selector-custom")).not.toBeInTheDocument()
    );
    fireEvent.keyDown(search, { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("Should offer a custom commit for the active provider even when the id is known under another provider", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [codexProvider, claudeProvider],
      // "shared" exists ONLY under Claude — it must not block "shared" as a custom Codex id.
      models: [model("shared", { provider: "claude", name: "Shared on Claude" })],
    });

    await openVia(user, "model");
    fireEvent.change(screen.getByTestId("runtime-selector-search"), {
      target: { value: "shared" },
    });
    const custom = await screen.findByTestId("runtime-selector-custom");
    expect(custom).toHaveTextContent('Use "shared"');
    await user.click(custom);

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "shared",
      reasoning_effort: "",
    });
  });

  it("Should not offer a custom commit when the id exactly matches a model under the active provider", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [codexProvider],
      models: [model("gpt-5.6", { name: "GPT 5.6" })],
    });

    await openVia(user, "model");
    fireEvent.change(screen.getByTestId("runtime-selector-search"), {
      target: { value: "gpt-5.6" },
    });

    // Exact (codex, gpt-5.6) match → the real row is offered, never a custom commit.
    await waitFor(() => expect(row("gpt-5.6")).toBeInTheDocument());
    const custom = screen.getByTestId("runtime-selector-custom");
    expect(custom).toHaveTextContent("Use an exact custom model ID…");
    expect(custom).not.toHaveTextContent('Use "gpt-5.6"');
  });

  it("Should emit the exact rail-selected provider for a custom id", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "", model: "", reasoning_effort: "" },
      providers: [codexProvider, claudeProvider],
      models: [model("codex-a", { provider: "codex", name: "Codex A" })],
    });

    await openVia(user, "model");
    // Pick the Claude provider rail — the custom target is now explicitly Claude.
    await user.click(document.querySelector('[data-rail="claude"]')!);
    fireEvent.change(screen.getByTestId("runtime-selector-search"), {
      target: { value: "custom-z" },
    });
    await user.click(await screen.findByTestId("runtime-selector-custom"));

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "claude",
      model: "custom-z",
      reasoning_effort: "",
    });
  });
});

// ---------------------------------------------------------------------------
// Browse vs search + ranking
// ---------------------------------------------------------------------------

describe("RuntimeSelector browse, search and ranking", () => {
  const rankingModels = [
    model("gpt-plain", { name: "GPT Plain" }),
    model("gpt-featured", { name: "GPT Featured", featured: true }),
    model("gpt-fav", { name: "GPT Fav" }),
    model("gpt-hidden", { name: "GPT Hidden", curated: false }),
  ];

  it("Should show only curated rows while browsing and reveal non-curated rows on search", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: rankingModels,
    });

    await openVia(user, "model");

    // Browse: curated visible, non-curated hidden.
    expect(document.querySelector('[data-model="gpt-plain"]')).toBeInTheDocument();
    expect(document.querySelector('[data-model="gpt-hidden"]')).not.toBeInTheDocument();

    // Search: non-curated match now appears.
    fireEvent.change(screen.getByTestId("runtime-selector-search"), { target: { value: "gpt" } });
    await waitFor(() =>
      expect(document.querySelector('[data-model="gpt-hidden"]')).toBeInTheDocument()
    );
  });

  it("Should order rows favorites first, then featured, then daemon order", async () => {
    window.localStorage.setItem(
      FAVORITES_STORAGE_KEY,
      JSON.stringify([runtimeModelKey("codex", "gpt-fav")])
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: rankingModels,
    });

    await openVia(user, "model");
    // Drop the pinned "Recent & favorites" block so the provider group is the only listbox content.
    await user.click(document.querySelector('[data-rail="codex"]')!);

    await waitFor(() => expect(optionOrder()).toEqual(["gpt-fav", "gpt-featured", "gpt-plain"]));
  });
});

// ---------------------------------------------------------------------------
// Favorites + recents persistence
// ---------------------------------------------------------------------------

describe("RuntimeSelector favorites and recents persistence", () => {
  function readList(key: string): string[] {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as string[]) : [];
  }

  it("Should persist a favorite toggle via the external favorite action (pointer)", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    // Hover activates the row; the real external favorite button acts on it — no
    // interactive control is nested inside the listbox option.
    await user.hover(row("gpt-a"));
    await user.click(screen.getByTestId("runtime-selector-favorite-action"));

    expect(readList(FAVORITES_STORAGE_KEY)).toContain(runtimeModelKey("codex", "gpt-a"));
    expect(row("gpt-a")).toHaveAttribute("data-favorite", "true");
  });

  it("Should expose the favorite action as a real toggle button with no interactive control nested in options", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    // ARIA-valid list: a listbox option wraps no button / role=button descendant.
    expect(row("gpt-a").querySelector("button, [role='button']")).toBeNull();

    await user.hover(row("gpt-a"));
    const favButton = screen.getByTestId("runtime-selector-favorite-action");
    expect(favButton.tagName).toBe("BUTTON");
    // A real toggle button whose accessible name is STABLE (includes provider
    // identity) and does NOT flip on toggle — favorite truth rides on aria-pressed.
    expect(favButton).toHaveAttribute("aria-pressed", "false");
    expect(favButton).toHaveAccessibleName("Favorite GPT A from Codex");

    await user.click(favButton);
    // Toggling flips aria-pressed; the accessible name stays identical.
    expect(favButton).toHaveAttribute("aria-pressed", "true");
    expect(favButton).toHaveAccessibleName("Favorite GPT A from Codex");
  });

  it("Should keep the favorite action disabled for a disabled row", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [
        model("gpt-disabled", {
          name: "GPT Disabled",
          disabled: true,
          disabled_reason: "Unavailable",
        }),
      ],
    });

    await openVia(user, "model");
    await user.hover(row("gpt-disabled"));
    const favorite = screen.getByTestId("runtime-selector-favorite-action");
    expect(favorite).toBeDisabled();
    expect(favorite).not.toHaveAttribute("aria-keyshortcuts");

    fireEvent.keyDown(screen.getByTestId("runtime-selector-search"), {
      code: "KeyF",
      altKey: true,
    });
    expect(readList(FAVORITES_STORAGE_KEY)).toEqual([]);
  });

  it("Should favorite the highlighted option with the Alt+F keyboard action", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("gpt-a")).toHaveAttribute("data-highlighted", "true"));
    // Alt+F (layout-independent code) — NOT Cmd/Ctrl-D (browser bookmark conflict).
    fireEvent.keyDown(search, { code: "KeyF", altKey: true });

    expect(readList(FAVORITES_STORAGE_KEY)).toContain(runtimeModelKey("codex", "gpt-a"));
  });

  it("Should carry the Alt+F shortcut on the external favorite control, never on listbox options", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    // The option is pure — no aria-keyshortcuts. The real external favorite button
    // is the only carrier of the Alt+F accelerator once it has an active target.
    expect(row("gpt-a")).not.toHaveAttribute("aria-keyshortcuts");
    await user.hover(row("gpt-a"));
    expect(screen.getByTestId("runtime-selector-favorite-action")).toHaveAttribute(
      "aria-keyshortcuts",
      "Alt+F"
    );
  });

  it("Should keep the active (provider,model) target and live aria-activedescendant across favorite reorder", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-one", { name: "GPT One" }), model("gpt-two", { name: "GPT Two" })],
    });

    // Search (no pinned block) so each row renders once; highlight the SECOND row.
    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    await user.type(search, "gpt");
    fireEvent.keyDown(search, { key: "ArrowDown" });
    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("gpt-two")).toHaveAttribute("data-highlighted", "true"));
    expect(optionOrder()).toEqual(["gpt-one", "gpt-two"]);

    // Favorite the active row: favorites rank first, so gpt-two moves to the top.
    fireEvent.keyDown(search, { code: "KeyF", altKey: true });
    await waitFor(() => expect(optionOrder()).toEqual(["gpt-two", "gpt-one"]));

    // The active target followed its (provider,model) identity to the new position,
    // and aria-activedescendant still points at gpt-two — not whatever row now sits
    // at the old index.
    expect(row("gpt-two")).toHaveAttribute("data-highlighted", "true");
    expect(row("gpt-one")).toHaveAttribute("data-highlighted", "false");
    expect(search).toHaveAttribute("aria-activedescendant", row("gpt-two").id);
  });

  it("Should keep pinned and grouped occurrences independently highlightable", async () => {
    const key = runtimeModelKey("codex", "gpt-a");
    window.localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([key]));
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    const occurrences = document.querySelectorAll<HTMLElement>('[data-model="gpt-a"]');
    expect(occurrences).toHaveLength(2);

    await user.hover(occurrences[1]);
    await waitFor(() => expect(occurrences[1]).toHaveAttribute("data-highlighted", "true"));
    expect(occurrences[0]).toHaveAttribute("data-highlighted", "false");
    const search = screen.getByTestId("runtime-selector-search");
    expect(search).toHaveAttribute("aria-activedescendant", occurrences[1].id);

    await user.click(screen.getByTestId("runtime-selector-favorite-action"));
    await waitFor(() =>
      expect(document.querySelectorAll<HTMLElement>('[data-model="gpt-a"]')).toHaveLength(1)
    );
    const remaining = document.querySelector<HTMLElement>('[data-model="gpt-a"]');
    expect(remaining).toHaveAttribute("data-highlighted", "true");
    expect(search).toHaveAttribute("aria-activedescendant", remaining?.id);
  });

  it("Should announce Favorited/Unfavorited through a polite status while focus stays in search", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("gpt-a")).toHaveAttribute("data-highlighted", "true"));

    const announcer = screen.getByTestId("runtime-selector-announcer");
    expect(announcer).toHaveAttribute("aria-live", "polite");

    fireEvent.keyDown(search, { code: "KeyF", altKey: true });
    await waitFor(() => expect(announcer).toHaveTextContent("Favorited GPT A from Codex"));
    fireEvent.keyDown(search, { code: "KeyF", altKey: true });
    await waitFor(() => expect(announcer).toHaveTextContent("Unfavorited GPT A from Codex"));
  });

  it("Should purge invalid favorites once the catalog loads EMPTY, but not while it is still loading", async () => {
    const ghost = runtimeModelKey("codex", "ghost");
    // Loading (catalogLoaded=false) + empty catalog: nothing is purged — a valid
    // favorite must never be wiped before the catalog resolves.
    window.localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([ghost]));
    const loading = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [],
      props: { catalogLoaded: false },
    });
    expect(readList(FAVORITES_STORAGE_KEY)).toEqual([ghost]);
    loading.unmount();

    // Loaded-empty (catalogLoaded=true) + empty catalog: every persisted entry is
    // now invalid and is purged so it can never resurrect after reload.
    window.localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([ghost]));
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [],
      props: { catalogLoaded: true },
    });
    await waitFor(() => expect(readList(FAVORITES_STORAGE_KEY)).toEqual([]));
  });

  it("Should persist picked models to recents deduped and most-recent-first", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" }), model("gpt-b", { name: "GPT B" })],
    });

    await openVia(user, "model");
    await user.click(row("gpt-a"));
    await user.click(row("gpt-b"));
    await user.click(row("gpt-a"));

    expect(readList(RECENTS_STORAGE_KEY)).toEqual([
      runtimeModelKey("codex", "gpt-a"),
      runtimeModelKey("codex", "gpt-b"),
    ]);
  });

  it("Should cap recents at the configured limit when a new model is picked", async () => {
    // Seed six recents that ARE current tuples (their models exist), so the strict
    // reconcile keeps them and the cap is exercised on valid entries.
    const seededIds = ["r1", "r2", "r3", "r4", "r5", "r6"];
    window.localStorage.setItem(
      RECENTS_STORAGE_KEY,
      JSON.stringify(seededIds.map(id => runtimeModelKey("codex", id)))
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [...seededIds, "r7"].map(id => model(id, { name: id.toUpperCase() })),
    });

    await openVia(user, "model");
    await user.click(row("r7"));

    const recents = readList(RECENTS_STORAGE_KEY);
    expect(recents).toHaveLength(RECENTS_LIMIT);
    expect(recents[0]).toBe(runtimeModelKey("codex", "r7"));
    expect(recents).not.toContain(runtimeModelKey("codex", "r6"));
  });

  it("Should discard non-current-tuple favorites and re-persist only valid compound keys", async () => {
    const validKey = runtimeModelKey("codex", "gpt-a");
    window.localStorage.setItem(
      FAVORITES_STORAGE_KEY,
      JSON.stringify([
        validKey, // exact current tuple → kept
        "gpt-a", // bare model id (no provider) → dropped
        "garbage-foreign-string", // foreign string → dropped
        runtimeModelKey("codex", "ghost"), // structurally valid but absent from catalog → dropped
      ])
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
    });

    await openVia(user, "model");

    // Storage is reconciled to ONLY the valid current tuple; the ghosts are purged
    // so they never reappear, and the valid favorite still renders as favorited.
    await waitFor(() => expect(readList(FAVORITES_STORAGE_KEY)).toEqual([validKey]));
    expect(row("gpt-a")).toHaveAttribute("data-favorite", "true");
  });

  it("Should discard non-current-tuple recents and keep each provider's key distinct", async () => {
    const validKey = runtimeModelKey("codex", "gpt-a");
    window.localStorage.setItem(
      RECENTS_STORAGE_KEY,
      JSON.stringify([
        validKey, // codex/gpt-a exists → kept
        "gpt-a", // bare id → dropped
        runtimeModelKey("claude", "gpt-a"), // claude/gpt-a not in catalog → dropped (distinct key)
      ])
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: [codexProvider, claudeProvider],
      models: [model("gpt-a", { provider: "codex", name: "GPT A" })],
    });

    await openVia(user, "model");

    await waitFor(() => expect(readList(RECENTS_STORAGE_KEY)).toEqual([validKey]));
  });

  it("Should normalize duplicate storage and reject mutations outside the loaded catalog", async () => {
    const validKey = runtimeModelKey("codex", "gpt-a");
    const invalidKey = runtimeModelKey("codex", "ghost");
    window.localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([validKey, validKey]));
    window.localStorage.setItem(RECENTS_STORAGE_KEY, JSON.stringify([validKey, validKey]));
    const validKeys = new Set([validKey]);
    const { result } = renderHook(() => useRuntimeFavorites(validKeys, true));

    await waitFor(() => expect(readList(FAVORITES_STORAGE_KEY)).toEqual([validKey]));
    await waitFor(() => expect(readList(RECENTS_STORAGE_KEY)).toEqual([validKey]));

    act(() => {
      result.current.toggleFavorite(invalidKey);
      result.current.pushRecent(invalidKey);
    });
    expect(readList(FAVORITES_STORAGE_KEY)).toEqual([validKey]);
    expect(readList(RECENTS_STORAGE_KEY)).toEqual([validKey]);
  });

  it("Should not rewrite unchanged favorites when only the valid-key set identity changes", async () => {
    const validKey = runtimeModelKey("codex", "gpt-a");
    window.localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([validKey]));
    const writeSpy = vi.spyOn(Storage.prototype, "setItem");
    let validKeys = new Set([validKey]);
    const { rerender } = renderHook(() => useRuntimeFavorites(validKeys, true));

    await waitFor(() => expect(readList(FAVORITES_STORAGE_KEY)).toEqual([validKey]));
    writeSpy.mockClear();
    validKeys = new Set([validKey]);
    rerender();

    expect(writeSpy).not.toHaveBeenCalled();
  });

  it("Should render the pinned 'Recent & favorites' block under the all rail and a 'Favorites' block under the fav rail", async () => {
    window.localStorage.setItem(
      FAVORITES_STORAGE_KEY,
      JSON.stringify([runtimeModelKey("codex", "gpt-a")])
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" }), model("gpt-b", { name: "GPT B" })],
    });

    await openVia(user, "model");
    const list = screen.getByTestId("runtime-selector-list");
    expect(within(list).getByText("Recent & favorites")).toBeInTheDocument();

    await user.click(document.querySelector('[data-rail="fav"]')!);
    expect(within(list).getByText("Favorites")).toBeInTheDocument();
    expect(within(list).queryByText("Recent & favorites")).not.toBeInTheDocument();
    expect(document.querySelector('[data-model="gpt-a"]')).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Keyboard navigation + segment focus
// ---------------------------------------------------------------------------

describe("RuntimeSelector keyboard navigation", () => {
  const navModels = [
    model("nav-a", { name: "Nav A" }),
    model("nav-b", {
      name: "Nav B",
      disabled: true,
      disabled_reason: "Unavailable",
      availability: "unavailable",
    }),
    model("nav-c", { name: "Nav C" }),
  ];

  it("Should move the highlight with ArrowDown/ArrowUp, skipping disabled rows and wrapping", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: navModels,
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");

    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("nav-a")).toHaveAttribute("data-highlighted", "true"));

    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("nav-c")).toHaveAttribute("data-highlighted", "true"));
    expect(row("nav-b")).toHaveAttribute("data-highlighted", "false");

    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("nav-a")).toHaveAttribute("data-highlighted", "true"));

    fireEvent.keyDown(search, { key: "ArrowUp" });
    await waitFor(() => expect(row("nav-c")).toHaveAttribute("data-highlighted", "true"));
  });

  it("Should select the highlighted row on Enter", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: navModels,
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");
    fireEvent.keyDown(search, { key: "ArrowDown" });
    await waitFor(() => expect(row("nav-a")).toHaveAttribute("data-highlighted", "true"));
    fireEvent.keyDown(search, { key: "Enter" });

    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "nav-a",
      reasoning_effort: "",
    });
  });

  it("Should close on Escape and restore focus to the exact segment that opened the popup", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: navModels,
    });

    // Open via the MODEL segment — focus must return there, not always to provider.
    await openVia(user, "model");
    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(screen.queryByTestId("runtime-selector-popup")).not.toBeInTheDocument()
    );
    const trigger = screen.getByTestId("rt-trigger");
    await waitFor(() => expect(trigger.querySelector('button[data-focus="model"]')).toHaveFocus());
  });

  it("Should focus the current provider rail item when the provider segment opens the popup", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: navModels,
    });

    await openVia(user, "provider");
    await waitFor(() => expect(document.querySelector('[data-rail="codex"]')).toHaveFocus());
  });

  it("Should focus the active reasoning button when the reasoning segment opens the popup", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "leveled", reasoning_effort: "high" },
      models: [model("leveled", { name: "Leveled", efforts: ["low", "medium", "high"] })],
    });

    await openVia(user, "reasoning");
    await waitFor(() => expect(document.querySelector('button[data-rz="high"]')).toHaveFocus());
  });
});

describe("RuntimeSelector listbox semantics", () => {
  it("Should expose loading and empty content as status rather than listbox options", async () => {
    const user = userEvent.setup();
    const loading = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [],
      props: { loading: true },
    });

    await openVia(user, "model");
    expect(screen.getByTestId("runtime-selector-list")).toHaveAttribute("role", "status");
    expect(screen.queryByRole("listbox", { name: "Models" })).not.toBeInTheDocument();
    expect(screen.getByTestId("runtime-selector-loading")).toBeInTheDocument();
    loading.unmount();

    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [],
      props: { catalogLoaded: true },
    });
    await openVia(user, "model");
    expect(screen.getByTestId("runtime-selector-list")).toHaveAttribute("role", "status");
    expect(screen.queryByRole("listbox", { name: "Models" })).not.toBeInTheDocument();
    expect(screen.getByTestId("runtime-selector-empty")).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Compound provider·model identity (two providers sharing a model id)
// ---------------------------------------------------------------------------

function rowFor(provider: string, id: string): HTMLElement {
  const el = document.querySelector<HTMLElement>(
    `[data-provider="${provider}"][data-model="${id}"]`
  );
  if (!el) throw new Error(`Missing row ${provider}/${id}`);
  return el;
}

function readStoredList(key: string): string[] {
  const raw = window.localStorage.getItem(key);
  return raw ? (JSON.parse(raw) as string[]) : [];
}

describe("RuntimeSelector compound provider·model identity", () => {
  const twoProviders = [codexProvider, claudeProvider];
  const sharedModels = [
    model("shared-model", { provider: "codex", name: "Shared on Codex", efforts: ["low", "high"] }),
    model("shared-model", {
      provider: "claude",
      name: "Shared on Claude",
      efforts: ["medium", "high"],
    }),
  ];

  it("Should render both providers' rows when a model id is shared across providers", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: sharedModels,
    });

    await openVia(user, "model");
    expect(rowFor("codex", "shared-model")).toBeInTheDocument();
    expect(rowFor("claude", "shared-model")).toBeInTheDocument();
    expect(document.querySelectorAll('[data-model="shared-model"]')).toHaveLength(2);
  });

  it("Should emit the exact provider of the clicked row for a shared model id", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: sharedModels,
    });

    await openVia(user, "model");
    await user.click(rowFor("claude", "shared-model"));
    expect(onChange).toHaveBeenLastCalledWith({
      provider: "claude",
      model: "shared-model",
      reasoning_effort: "",
    });

    await openVia(user, "model");
    await user.click(rowFor("codex", "shared-model"));
    expect(onChange).toHaveBeenLastCalledWith({
      provider: "codex",
      model: "shared-model",
      reasoning_effort: "",
    });
  });

  it("Should favorite only the active provider's row for a shared model id", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: sharedModels,
    });

    await openVia(user, "model");
    // Hover the Claude row so the external favorite action targets that exact tuple.
    await user.hover(rowFor("claude", "shared-model"));
    await user.click(screen.getByTestId("runtime-selector-favorite-action"));

    const favorites = readStoredList(FAVORITES_STORAGE_KEY);
    expect(favorites).toContain(runtimeModelKey("claude", "shared-model"));
    expect(favorites).not.toContain(runtimeModelKey("codex", "shared-model"));
    // Compound identity: only the Claude row flips; the Codex row stays un-favorited.
    expect(rowFor("claude", "shared-model")).toHaveAttribute("data-favorite", "true");
    expect(rowFor("codex", "shared-model")).toHaveAttribute("data-favorite", "false");
  });
});

// ---------------------------------------------------------------------------
// Provider rail — local filter only, cross-provider favorites
// ---------------------------------------------------------------------------

describe("RuntimeSelector provider rail filtering", () => {
  const twoProviders = [codexProvider, claudeProvider];
  const railModels = [
    model("codex-a", { provider: "codex", name: "Codex A" }),
    model("claude-a", { provider: "claude", name: "Claude A" }),
  ];

  it("Should filter the list to a provider without emitting a value change", async () => {
    const user = userEvent.setup();
    const { onChange } = renderSelector({
      value: { provider: "codex", model: "codex-a", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
    });

    await openVia(user, "model");
    await user.click(document.querySelector('[data-rail="claude"]')!);

    await waitFor(() => expect(rowFor("claude", "claude-a")).toBeInTheDocument());
    expect(
      document.querySelector('[data-provider="codex"][data-model="codex-a"]')
    ).not.toBeInTheDocument();
    // Rail is a list filter: the controlled value never changes.
    expect(onChange).not.toHaveBeenCalled();
  });

  it("Should surface favorites from every provider under the Favorites rail", async () => {
    window.localStorage.setItem(
      FAVORITES_STORAGE_KEY,
      JSON.stringify([runtimeModelKey("codex", "codex-a"), runtimeModelKey("claude", "claude-a")])
    );
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
    });

    await openVia(user, "model");
    await user.click(document.querySelector('[data-rail="fav"]')!);

    await waitFor(() => {
      expect(rowFor("codex", "codex-a")).toBeInTheDocument();
      expect(rowFor("claude", "claude-a")).toBeInTheDocument();
    });
  });

  it("Should be a radiogroup (not tabs) and move the filter with roving arrow-key navigation", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
    });

    await openVia(user, "model");
    // Truthful semantics: a local mutually-exclusive filter is a radiogroup, never
    // a tablist (the rail does not swap a tabpanel — it filters the list in place).
    const radiogroup = document.querySelector<HTMLElement>('[role="radiogroup"]')!;
    expect(radiogroup).toBeInTheDocument();
    expect(document.querySelector('[role="tablist"]')).toBeNull();
    expect(document.querySelector('[data-rail="all"]')).toHaveAttribute("role", "radio");
    expect(document.querySelector('[data-rail="all"]')).toHaveAttribute("aria-checked", "true");
    // The checked radio (All) is the only one in the Tab sequence (roving tabindex).
    expect(document.querySelector('[data-rail="all"]')).toHaveAttribute("tabindex", "0");
    expect(document.querySelector('[data-rail="claude"]')).toHaveAttribute("tabindex", "-1");

    fireEvent.keyDown(radiogroup, { key: "ArrowDown" });
    await waitFor(() =>
      expect(document.querySelector('[data-rail="fav"]')).toHaveAttribute("data-active", "true")
    );
  });

  it("Should keep Provider Settings structurally outside the filter radiogroup", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
      props: { onOpenProviderSettings: vi.fn() },
    });

    await openVia(user, "model");
    const settings = screen.getByTestId("runtime-selector-settings");
    const radiogroup = document.querySelector('[role="radiogroup"]')!;
    // Settings is an escape hatch, not a fourth filter — it must not live in the group.
    expect(radiogroup.contains(settings)).toBe(false);
    expect(settings).not.toHaveAttribute("role", "radio");
  });

  it("Should jump the rail filter to the last provider on End and back to All on Home", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
    });

    await openVia(user, "model");
    const radiogroup = document.querySelector<HTMLElement>('[role="radiogroup"]')!;

    fireEvent.keyDown(radiogroup, { key: "End" });
    await waitFor(() =>
      expect(document.querySelector('[data-rail="claude"]')).toHaveAttribute("data-active", "true")
    );

    fireEvent.keyDown(radiogroup, { key: "Home" });
    await waitFor(() =>
      expect(document.querySelector('[data-rail="all"]')).toHaveAttribute("data-active", "true")
    );
  });

  it("Should suppress rail radios while searching (not keyboard-activatable)", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      providers: twoProviders,
      models: railModels,
    });

    await openVia(user, "model");
    fireEvent.change(screen.getByTestId("runtime-selector-search"), { target: { value: "codex" } });

    await waitFor(() => expect(document.querySelector('[data-rail="codex"]')).toBeDisabled());
    expect(document.querySelector('[data-rail="all"]')).toBeDisabled();
  });
});

// ---------------------------------------------------------------------------
// Row chips + reasoning footer modes
// ---------------------------------------------------------------------------

describe("RuntimeSelector row chips", () => {
  it("Should render context, all five cost rates, tools and levels from truthful metadata", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [
        model("rich", {
          name: "Rich",
          context_window: 1_050_000,
          cost_input: 5,
          cost_output: 30,
          cost_cache_read: 0.5,
          cost_cache_write: 8,
          cost_reasoning: 40,
          supports_tools: true,
          efforts: ["low", "medium", "high"],
        }),
      ],
    });

    await openVia(user, "model");
    const richRow = row("rich");
    expect(richRow).toHaveTextContent("1.05M");
    expect(richRow).toHaveTextContent("input $5/M");
    expect(richRow).toHaveTextContent("output $30/M");
    expect(richRow).toHaveTextContent("cache read $0.5/M");
    expect(richRow).toHaveTextContent("cache write $8/M");
    expect(richRow).toHaveTextContent("reasoning $40/M");
    expect(richRow).toHaveTextContent("tools");
    expect(richRow).toHaveTextContent("3 levels");
  });

  it("Should render only explicitly present cost rates without borrowing another bucket", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [
        model("partial-cost", {
          name: "Partial cost",
          cost_input: 5,
          cost_cache_read: 0,
        }),
      ],
    });

    await openVia(user, "model");
    const partialRow = row("partial-cost");
    expect(partialRow).toHaveTextContent("input $5/M");
    expect(partialRow).toHaveTextContent("cache read $0/M");
    expect(partialRow).not.toHaveTextContent("output $");
    expect(partialRow).not.toHaveTextContent("cache write $");
    expect(partialRow).not.toHaveTextContent("reasoning $");
  });

  it("Should render a 'reasoning' chip when a model supports reasoning without selectable levels", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("supp", { name: "Supported", supports_reasoning: true, efforts: [] })],
    });

    await openVia(user, "model");
    expect(row("supp")).toHaveTextContent("reasoning");
  });
});

describe("RuntimeSelector reasoning footer modes", () => {
  async function footerFor(models: RuntimeModelOption[], value: RuntimeSelectorValue) {
    const user = userEvent.setup();
    renderSelector({ value, models });
    await openVia(user, "model");
    return screen.getByTestId("runtime-selector-reasoning");
  }

  it("Should prompt to pick a model when nothing is selected (no 'this model' claim)", async () => {
    const footer = await footerFor([model("a")], {
      provider: "codex",
      model: "",
      reasoning_effort: "",
    });
    expect(footer).toHaveAttribute("data-reasoning-mode", "no-model");
    expect(footer).toHaveTextContent("Select a model");
  });

  it("Should show the 'provider decides' note for supported-without-levels models", async () => {
    const footer = await footerFor([model("supp", { supports_reasoning: true, efforts: [] })], {
      provider: "codex",
      model: "supp",
      reasoning_effort: "",
    });
    expect(footer).toHaveAttribute("data-reasoning-mode", "supported-nolevels");
    expect(footer).toHaveTextContent("Reasoning is on");
  });

  it("Should show the 'no reasoning effort' note for models without reasoning", async () => {
    const footer = await footerFor([model("plain", { supports_reasoning: false, efforts: [] })], {
      provider: "codex",
      model: "plain",
      reasoning_effort: "",
    });
    expect(footer).toHaveAttribute("data-reasoning-mode", "none");
    expect(footer).toHaveTextContent("doesn’t use reasoning effort");
  });
});

// ---------------------------------------------------------------------------
// Reasoning strip label identity — per-instance ARIA ids
// ---------------------------------------------------------------------------

describe("RuntimeSelector reasoning strip label identity", () => {
  it("Should give each mounted reasoning strip a unique, locally-wired label id", () => {
    const reasoning = resolveReasoningState(
      model("leveled", { efforts: ["low", "high"], reasoning_source: "catalog" })
    );
    render(
      <UIProvider reducedMotion="always">
        <ReasoningBar
          reasoning={reasoning}
          value=""
          providerName="Codex"
          modelName="Leveled"
          onSelect={() => {}}
        />
        <ReasoningBar
          reasoning={reasoning}
          value=""
          providerName="Codex"
          modelName="Leveled"
          onSelect={() => {}}
        />
      </UIProvider>
    );

    const labelledby = screen
      .getAllByRole("group")
      .map(group => group.getAttribute("aria-labelledby"));
    expect(labelledby).toHaveLength(2);
    // Distinct per-instance ids (no shared fixed id colliding across mounts).
    expect(labelledby[0]).toBeTruthy();
    expect(labelledby[0]).not.toBe(labelledby[1]);
    // Each group points at a label that exists locally in the document.
    for (const id of labelledby) {
      expect(document.getElementById(id ?? "")).toBeInTheDocument();
    }
  });
});

// ---------------------------------------------------------------------------
// ⌘K ownership — one global listener, a single deterministic owner
// ---------------------------------------------------------------------------

describe("RuntimeSelector ⌘K ownership", () => {
  function Instance({ testId }: { testId: string }) {
    const [value, setValue] = useState<RuntimeSelectorValue>({
      provider: "codex",
      model: "",
      reasoning_effort: "",
    });
    return (
      <RuntimeSelector
        value={value}
        onChange={setValue}
        providers={[codexProvider]}
        models={[]}
        triggerTestId={testId}
      />
    );
  }

  function providerButton(testId: string): HTMLElement {
    return within(screen.getByTestId(testId)).getByRole("button", { name: /^Provider:/ });
  }

  it("Should resolve one deterministic owner across two instances and toggle it with ⌘K", async () => {
    render(
      <UIProvider reducedMotion="always">
        <Instance testId="rt-a" />
        <Instance testId="rt-b" />
      </UIProvider>
    );

    // Two eligible instances, none focused → ownership stays DETERMINISTIC (never
    // "no owner"): the most-recently-registered eligible selector (rt-b) owns ⌘K,
    // and exactly ONE popup opens — no competing per-instance handlers fight.
    fireEvent.keyDown(document.body, { key: "k", metaKey: true });
    await waitFor(() => expect(screen.getAllByTestId("runtime-selector-popup")).toHaveLength(1));
    expect(screen.getByTestId("rt-b")).toHaveAttribute("data-open", "true");
    expect(screen.getByTestId("rt-a")).toHaveAttribute("data-open", "false");

    // Close it and refocus the FIRST selector → it becomes the last-focused owner;
    // ⌘K now opens rt-a instead (focus wins over the registration fallback).
    fireEvent.keyDown(screen.getByTestId("runtime-selector-search"), { key: "Escape" });
    await waitFor(() => expect(screen.queryAllByTestId("runtime-selector-popup")).toHaveLength(0));
    const firstProvider = providerButton("rt-a");
    firstProvider.focus();
    fireEvent.keyDown(firstProvider, { key: "k", metaKey: true });
    await waitFor(() => expect(screen.getAllByTestId("runtime-selector-popup")).toHaveLength(1));
    expect(screen.getByTestId("rt-a")).toHaveAttribute("data-open", "true");

    // ⌘K again while owned+open toggles it closed (rule 1: the open selector owns).
    fireEvent.keyDown(screen.getByTestId("runtime-selector-search"), { key: "k", metaKey: true });
    await waitFor(() => expect(screen.queryAllByTestId("runtime-selector-popup")).toHaveLength(0));
  });

  it("Should hand ⌘K ownership to the sole remaining selector after the focused one unmounts", async () => {
    function Harness() {
      const [showB, setShowB] = useState(true);
      return (
        <UIProvider reducedMotion="always">
          <Instance testId="rt-a" />
          {showB ? <Instance testId="rt-b" /> : null}
          <button type="button" onClick={() => setShowB(false)}>
            hide b
          </button>
        </UIProvider>
      );
    }
    const user = userEvent.setup();
    render(<Harness />);

    // Focus B (making it the last-focused owner), then unmount it.
    providerButton("rt-b").focus();
    await user.click(screen.getByRole("button", { name: "hide b" }));
    expect(screen.queryByTestId("rt-b")).not.toBeInTheDocument();

    // The unmounted B can no longer own ⌘K; A is now the sole eligible owner and
    // opens deterministically even without focus. B is gone, so the single popup
    // that appears is unambiguously A's.
    fireEvent.keyDown(document.body, { key: "k", metaKey: true });
    await waitFor(() => expect(screen.getAllByTestId("runtime-selector-popup")).toHaveLength(1));
    expect(screen.getByTestId("runtime-selector-search")).toBeInTheDocument();
  });

  it("Should ignore open owners that are disabled or no longer visible", () => {
    const disabledClose = vi.fn();
    const hiddenClose = vi.fn();
    const eligibleOpen = vi.fn();
    const unregisterDisabled = registerCommandK({
      id: "disabled-open",
      isEligible: () => false,
      isVisible: () => true,
      isOpen: () => true,
      open: vi.fn(),
      close: disabledClose,
    });
    const unregisterHidden = registerCommandK({
      id: "hidden-open",
      isEligible: () => true,
      isVisible: () => false,
      isOpen: () => true,
      open: vi.fn(),
      close: hiddenClose,
    });
    const unregisterEligible = registerCommandK({
      id: "eligible",
      isEligible: () => true,
      isVisible: () => true,
      isOpen: () => false,
      open: eligibleOpen,
      close: vi.fn(),
    });

    try {
      fireEvent.keyDown(document.body, { key: "k", metaKey: true });
      expect(eligibleOpen).toHaveBeenCalledTimes(1);
      expect(disabledClose).not.toHaveBeenCalled();
      expect(hiddenClose).not.toHaveBeenCalled();
    } finally {
      unregisterEligible();
      unregisterHidden();
      unregisterDisabled();
    }
  });

  it("Should treat triggers under aria-hidden ancestors as not visible", () => {
    const host = document.createElement("div");
    host.setAttribute("aria-hidden", "true");
    const trigger = document.createElement("button");
    host.appendChild(trigger);
    document.body.appendChild(host);
    try {
      expect(triggerVisible(trigger)).toBe(false);
    } finally {
      host.remove();
    }
  });
});

// ---------------------------------------------------------------------------
// Home/End highlight + focus routing while open + Provider Settings order
// ---------------------------------------------------------------------------

describe("RuntimeSelector list keyboard edges", () => {
  it("Should jump the highlight to the last row on End and the first row on Home", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("edge-a"), model("edge-b"), model("edge-c")],
    });

    await openVia(user, "model");
    const search = screen.getByTestId("runtime-selector-search");

    fireEvent.keyDown(search, { key: "End" });
    await waitFor(() => expect(row("edge-c")).toHaveAttribute("data-highlighted", "true"));

    fireEvent.keyDown(search, { key: "Home" });
    await waitFor(() => expect(row("edge-a")).toHaveAttribute("data-highlighted", "true"));
  });
});

describe("RuntimeSelector focus routing while open", () => {
  it("Should route focus to the reasoning strip when the reasoning segment is clicked while already open", async () => {
    const user = userEvent.setup();
    renderSelector({
      value: { provider: "codex", model: "leveled", reasoning_effort: "high" },
      models: [model("leveled", { name: "Leveled", efforts: ["low", "medium", "high"] })],
    });

    // Opened via the model segment → search is focused.
    await openVia(user, "model");
    await waitFor(() => expect(screen.getByTestId("runtime-selector-search")).toHaveFocus());

    // Clicking the reasoning segment while open must move focus to the active
    // effort button — not merely record an intent — without closing the popup.
    fireEvent.click(document.querySelector<HTMLElement>('button[data-focus="reasoning"]')!);
    expect(screen.getByTestId("runtime-selector-popup")).toBeInTheDocument();
    await waitFor(() => expect(document.querySelector('button[data-rz="high"]')).toHaveFocus());
  });
});

describe("RuntimeSelector provider settings action order", () => {
  it("Should close the popup then invoke onOpenProviderSettings when the gear is clicked", async () => {
    const user = userEvent.setup();
    const onOpenProviderSettings = vi.fn();
    renderSelector({
      value: { provider: "codex", model: "", reasoning_effort: "" },
      models: [model("gpt-a", { name: "GPT A" })],
      props: { onOpenProviderSettings },
    });

    await openVia(user, "model");
    await user.click(screen.getByTestId("runtime-selector-settings"));

    expect(onOpenProviderSettings).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(screen.queryByTestId("runtime-selector-popup")).not.toBeInTheDocument()
    );
  });
});

// ---------------------------------------------------------------------------
// Intensity meter
// ---------------------------------------------------------------------------

describe("IntensityMeter + reasoningEffortPosition", () => {
  it("Should map each canonical effort to its 1-based position", () => {
    expect(REASONING_EFFORT_ORDER.map(reasoningEffortPosition)).toEqual([1, 2, 3, 4, 5, 6, 7]);
    expect(reasoningEffortPosition("")).toBe(0);
    expect(reasoningEffortPosition("nonsense")).toBe(0);
  });

  it("Should mark exactly the requested number of bars as filled for each canonical position", () => {
    for (let position = 0; position <= 7; position += 1) {
      const { container, unmount } = render(<IntensityMeter position={position} />);
      const meter = container.querySelector('[data-slot="runtime-intensity-meter"]');
      const bars = Array.from(meter?.children ?? []);
      expect(bars).toHaveLength(7);
      expect(meter).toHaveAttribute("data-position", String(position));
      expect(bars.filter(bar => bar.getAttribute("data-fill") === "on")).toHaveLength(position);
      unmount();
    }
  });

  it("Should mark every bar hollow with zero filled when hollow", () => {
    const { container } = render(<IntensityMeter position={5} hollow />);
    const meter = container.querySelector('[data-slot="runtime-intensity-meter"]');
    const bars = Array.from(meter?.children ?? []);
    expect(meter).toHaveAttribute("data-hollow", "true");
    expect(meter).toHaveAttribute("data-position", "0");
    expect(bars.filter(bar => bar.getAttribute("data-fill") === "on")).toHaveLength(0);
    expect(bars.every(bar => bar.getAttribute("data-fill") === "hollow")).toBe(true);
  });
});
