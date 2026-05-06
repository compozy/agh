import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksEmptyState } from "../tasks-empty-state";

describe("TasksEmptyState", () => {
  it("renders the headline with the workspace name and lists every template", () => {
    render(<TasksEmptyState onSelectTemplate={vi.fn()} workspaceName="Polybot" />);

    expect(screen.getByRole("heading", { name: "No tasks yet in Polybot" })).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-templates")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-one_shot")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-recurring")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-epic")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-remote_peer")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-human_in_loop")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-empty-template-blank")).toBeInTheDocument();
  });

  it("falls back to a generic headline when no workspace is provided", () => {
    render(<TasksEmptyState onSelectTemplate={vi.fn()} />);
    expect(screen.getByRole("heading", { name: "No tasks yet" })).toBeInTheDocument();
  });

  it("invokes onSelectTemplate from the primary CTA and from any template card", () => {
    const onSelectTemplate = vi.fn();
    render(<TasksEmptyState onSelectTemplate={onSelectTemplate} />);

    fireEvent.click(screen.getByTestId("tasks-empty-cta-new"));
    expect(onSelectTemplate).toHaveBeenLastCalledWith("one_shot");

    fireEvent.click(screen.getByTestId("tasks-empty-template-recurring"));
    expect(onSelectTemplate).toHaveBeenLastCalledWith("recurring");

    fireEvent.click(screen.getByTestId("tasks-empty-template-blank"));
    expect(onSelectTemplate).toHaveBeenLastCalledWith("blank");
  });

  it("only renders the copy CLI command when the handler is provided", () => {
    const onCopyCli = vi.fn();
    const { rerender } = render(<TasksEmptyState onSelectTemplate={vi.fn()} />);
    expect(screen.queryByTestId("tasks-empty-cta-cli")).not.toBeInTheDocument();

    rerender(<TasksEmptyState onCopyCli={onCopyCli} onSelectTemplate={vi.fn()} />);
    fireEvent.click(screen.getByTestId("tasks-empty-cta-cli"));
    expect(onCopyCli).toHaveBeenCalledTimes(1);
  });
});
