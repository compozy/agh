// Suite: Appearance settings pane
// Invariant: the pane binds wallpaper/magnification/reduce-motion to the desktop store with APG
// radio-group semantics, and states the system reduced-motion precedence truthfully (US-015.EC-1).
// Boundary IN: AppearanceSettingsPane interaction against a real desktop store.
// Boundary OUT: desktop-doc persistence (binder suite) and visual parity (VC-02 capture).
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OsShellContext, type OsShellHandle } from "../../../contexts/os-shell-context";
import { RoutingCoordinator, type OsRouterPort } from "../../../lib/routing-coordinator";
import { createDesktopStore } from "../../../stores/desktop-store";
import { AppearanceSettingsPane } from "../appearance-settings-pane";

vi.mock("@/systems/settings", () => ({
  useSettingsTopbar: vi.fn(),
}));

function matchMediaStub(matches: boolean) {
  return vi.fn().mockImplementation((query: string) => ({
    matches,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
  }));
}

function renderPane({ systemReducedMotion = false } = {}) {
  vi.stubGlobal("matchMedia", matchMediaStub(systemReducedMotion));
  const store = createDesktopStore();
  const port: OsRouterPort = { navigate: () => {}, replace: () => {} };
  const shell: OsShellHandle = {
    store,
    coordinator: new RoutingCoordinator(store, port),
    flushPersistence: () => {},
  };
  store.getState().hydrate([]);
  render(
    <OsShellContext.Provider value={shell}>
      <AppearanceSettingsPane />
    </OsShellContext.Provider>
  );
  return { store };
}

describe("AppearanceSettingsPane", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("Should select wallpapers as a radio group with pointer and arrow keys", async () => {
    const user = userEvent.setup();
    const { store } = renderPane();

    const group = screen.getByRole("radiogroup", { name: "Wallpaper" });
    expect(group).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /Ember/ })).toHaveAttribute("aria-checked", "true");

    await user.click(screen.getByRole("radio", { name: /Carbon/ }));
    expect(store.getState().wallpaper).toBe("carbon");
    expect(screen.getByRole("radio", { name: /Carbon/ })).toHaveAttribute("aria-checked", "true");

    // Arrow keys move AND select (automatic activation), wrapping the group.
    screen.getByRole("radio", { name: /Carbon/ }).focus();
    await user.keyboard("{ArrowRight}");
    expect(store.getState().wallpaper).toBe("ember");
    await user.keyboard("{ArrowLeft}");
    expect(store.getState().wallpaper).toBe("carbon");
  });

  it("Should write the magnification and reduce-motion toggles to the desktop doc", async () => {
    const user = userEvent.setup();
    const { store } = renderPane();

    await user.click(screen.getByTestId("os-appearance-magnify"));
    expect(store.getState().dockMagnify).toBe(false);

    await user.click(screen.getByTestId("os-appearance-reduce-motion"));
    expect(store.getState().reduceMotion).toBe(true);
  });

  it("Should state that the system reduced-motion preference wins while it is active (US-015.EC-1)", () => {
    renderPane({ systemReducedMotion: true });
    expect(
      screen.getByText(/system already prefers reduced motion — that preference wins/i)
    ).toBeInTheDocument();
  });
});
