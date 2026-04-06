import { fireEvent, render, screen } from "@testing-library/react";
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

describe("WorkspaceSelector", () => {
  it("renders the current workspace id and root dir", () => {
    render(<WorkspaceSelector workspaces={workspaces} value="ws_alpha" onValueChange={vi.fn()} />);

    expect(screen.getByDisplayValue("alpha")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-selector-id")).toHaveTextContent("ws_alpha");
    expect(screen.getByTestId("workspace-selector-root-dir")).toHaveTextContent("/workspace/alpha");
  });

  it("emits changes when a different workspace is chosen", () => {
    const onValueChange = vi.fn();
    render(
      <WorkspaceSelector workspaces={workspaces} value="ws_alpha" onValueChange={onValueChange} />
    );

    fireEvent.change(screen.getByLabelText("Workspace"), {
      target: { value: "ws_beta" },
    });

    expect(onValueChange).toHaveBeenCalledWith("ws_beta");
  });
});
