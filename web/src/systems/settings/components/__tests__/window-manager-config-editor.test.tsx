// Suite: Window Manager config editor
// Invariant: Save remains unavailable for numeric drafts the daemon would reject.
// Owning layer: the Settings config-editor component and its editor model.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import type { WindowManagerConfig } from "@/systems/os";

import { useWindowManagerConfigEditor } from "../../hooks/use-window-manager-config-editor";
import { WindowManagerConfigEditor } from "../window-manager-config-editor";

const CONFIG: WindowManagerConfig = {
  newWindowPolicy: "floating",
  smallViewportPolicy: "stack",
  focusPolicy: "click_directional",
  focusWrap: true,
  focusFollowsPointer: false,
  raiseOnFocus: true,
  dragAwayPolicy: "window",
  groupMoveModifier: "alt",
  swapModifier: "shift",
  historyLimit: 100,
  desktopTransition: "slide",
  gaps: { inner: 8, top: 8, right: 8, bottom: 8, left: 8 },
  snap: {
    edgeBand: 24,
    cornerReach: 96,
    exitSlack: 16,
    repeatRatios: [0.5, 0.33, 0.67],
  },
  bindings: { topCenter: "zoom", bottomCenter: "none" },
  shortcuts: {},
};

function EditorHarness() {
  const editor = useWindowManagerConfigEditor(CONFIG);
  return <WindowManagerConfigEditor editor={editor} />;
}

function renderEditor() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  render(<EditorHarness />, { wrapper });
}

describe("WindowManagerConfigEditor", () => {
  it("Should disable Save for empty and out-of-range numeric drafts", () => {
    renderEditor();
    const history = screen.getByRole("spinbutton", { name: "History limit" });
    const save = screen.getByRole("button", { name: "Save settings" });

    fireEvent.change(history, { target: { value: "101" } });
    expect(save).toBeEnabled();

    fireEvent.change(history, { target: { value: "" } });
    expect(history).toHaveAttribute("aria-invalid", "true");
    expect(save).toBeDisabled();

    fireEvent.change(history, { target: { value: "501" } });
    expect(history).toHaveAttribute("aria-invalid", "true");
    expect(save).toBeDisabled();
  });

  it("Should reject duplicate ratios using the daemon precision", () => {
    renderEditor();
    const ratios = screen.getByRole("textbox", { name: "Repeat ratios" });
    const save = screen.getByRole("button", { name: "Save settings" });

    fireEvent.change(ratios, { target: { value: "0.5, 0.5000001" } });

    expect(ratios).toHaveAttribute("aria-invalid", "true");
    expect(save).toBeDisabled();
  });
});
