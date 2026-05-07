import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsDeleteDialog } from "../settings-delete-dialog";

function baseProps() {
  return {
    open: true,
    slug: "widgets",
    title: "Delete widget?",
    isDeleting: false,
    onConfirm: vi.fn(),
    onOpenChange: vi.fn(),
  };
}

describe("SettingsDeleteDialog", () => {
  it("renders the title and description", () => {
    render(
      <SettingsDeleteDialog
        {...baseProps()}
        description="Removing the widget clears the overlay entry."
      />
    );
    expect(screen.getByTestId("settings-widgets-delete-title")).toHaveTextContent("Delete widget?");
    expect(screen.getByTestId("settings-widgets-delete-description")).toHaveTextContent(
      "Removing the widget"
    );
  });

  it("shows a fallback note explaining builtin fallback behavior", () => {
    render(
      <SettingsDeleteDialog {...baseProps()} fallbackNote="Builtin provider will be revealed." />
    );
    expect(screen.getByTestId("settings-widgets-delete-fallback")).toHaveTextContent(
      "Builtin provider will be revealed."
    );
  });

  it("surfaces server-side delete errors", () => {
    render(<SettingsDeleteDialog {...baseProps()} error="cannot delete builtin" />);
    expect(screen.getByTestId("settings-widgets-delete-error")).toHaveTextContent(
      "cannot delete builtin"
    );
  });

  it("wires confirm and cancel handlers", () => {
    const props = baseProps();
    render(<SettingsDeleteDialog {...props} />);
    fireEvent.click(screen.getByTestId("settings-widgets-delete-confirm"));
    expect(props.onConfirm).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("settings-widgets-delete-cancel"));
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
  });

  it("disables both buttons while the delete is pending", () => {
    render(<SettingsDeleteDialog {...baseProps()} isDeleting />);
    expect(screen.getByTestId("settings-widgets-delete-confirm")).toBeDisabled();
    expect(screen.getByTestId("settings-widgets-delete-cancel")).toBeDisabled();
  });
});
