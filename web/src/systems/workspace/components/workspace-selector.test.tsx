import { fireEvent, render, screen } from "@testing-library/react";
import { UIProvider } from "@agh/ui";
import { describe, expect, it, vi } from "vitest";

import { WorkspaceSelector } from "./workspace-selector";

const workspaces = [
  {
    id: "ws_alpha",
    root_dir: "/workspace/alpha",
    add_dirs: [],
    name: "alpha",
    created_at: "2026-04-06T10:00:00Z",
    updated_at: "2026-04-06T10:00:00Z",
  },
  {
    id: "ws_beta",
    root_dir: "/workspace/beta",
    add_dirs: [],
    name: "beta",
    created_at: "2026-04-06T10:00:00Z",
    updated_at: "2026-04-06T10:00:00Z",
  },
];

function renderSelector(props: Partial<React.ComponentProps<typeof WorkspaceSelector>> = {}) {
  const onSelectWorkspace = props.onSelectWorkspace ?? vi.fn();
  const merged = {
    workspaces,
    activeWorkspaceId: "ws_alpha" as string | null,
    onSelectWorkspace,
    ...props,
  };

  const utils = render(
    <UIProvider reducedMotion="always">
      <WorkspaceSelector {...merged} />
    </UIProvider>
  );

  return { ...utils, onSelectWorkspace };
}

describe("WorkspaceSelector", () => {
  it("renders one row per workspace with name, root dir, and status dot", () => {
    renderSelector();

    expect(screen.getByTestId("workspace-selector")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-selector-item-ws_alpha")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-selector-item-ws_beta")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-selector-name-ws_alpha")).toHaveTextContent("alpha");
    expect(screen.getByTestId("workspace-selector-root-dir-ws_alpha")).toHaveTextContent(
      "/workspace/alpha"
    );
    expect(screen.getByTestId("workspace-selector-dot-ws_alpha")).toBeInTheDocument();
  });

  it("highlights the active workspace via aria-current + data-active", () => {
    renderSelector({ activeWorkspaceId: "ws_beta" });

    const activeRow = screen.getByTestId("workspace-selector-item-ws_beta");
    expect(activeRow).toHaveAttribute("aria-current", "true");
    expect(activeRow).toHaveAttribute("data-active", "true");

    const inactiveRow = screen.getByTestId("workspace-selector-item-ws_alpha");
    expect(inactiveRow).not.toHaveAttribute("aria-current");
    expect(inactiveRow).toHaveAttribute("data-active", "false");
  });

  it("uses native list + button semantics instead of a broken listbox pattern", () => {
    renderSelector();

    expect(screen.getByTestId("workspace-selector")).not.toHaveAttribute("role");
    expect(screen.getByTestId("workspace-selector-item-ws_alpha")).not.toHaveAttribute("role");
    expect(screen.getByTestId("workspace-selector-item-ws_beta")).not.toHaveAttribute("role");
  });

  it("calls onSelectWorkspace with the id when a row is clicked", () => {
    const { onSelectWorkspace } = renderSelector();

    fireEvent.click(screen.getByTestId("workspace-selector-item-ws_beta"));

    expect(onSelectWorkspace).toHaveBeenCalledWith("ws_beta");
  });

  it("marks the global workspace with a HOME pill and others with a PATH pill", () => {
    renderSelector({ globalWorkspaceId: "ws_alpha" });

    expect(screen.getByTestId("workspace-selector-home-ws_alpha")).toHaveTextContent("HOME");
    expect(screen.getByTestId("workspace-selector-path-ws_beta")).toHaveTextContent("PATH");
  });

  it("renders the Empty state when no workspaces are passed", () => {
    renderSelector({ workspaces: [] });

    expect(screen.getByTestId("workspace-selector-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-selector")).not.toBeInTheDocument();
  });

  it("does not fire onSelectWorkspace when disabled", () => {
    const { onSelectWorkspace } = renderSelector({ disabled: true });

    fireEvent.click(screen.getByTestId("workspace-selector-item-ws_beta"));

    expect(onSelectWorkspace).not.toHaveBeenCalled();
  });
});
