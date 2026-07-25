import { fireEvent, render, screen } from "@testing-library/react";
import { KeyRound } from "lucide-react";
import { describe, expect, it, vi } from "vitest";

import { dialogShellClass } from "@agh/ui";

import { SettingsEditorDialog } from "../settings-editor-dialog";

function baseProps() {
  return {
    open: true,
    mode: "create" as const,
    icon: KeyRound,
    eyebrow: "System · Widgets",
    size: "sm" as const,
    title: "New widget",
    slug: "widgets",
    canSave: true,
    isSaving: false,
    onSave: vi.fn(),
    onOpenChange: vi.fn(),
  };
}

describe("SettingsEditorDialog", () => {
  it("renders title, body, and create save label in create mode", () => {
    const props = baseProps();
    render(
      <SettingsEditorDialog {...props}>
        <div data-testid="body">hello</div>
      </SettingsEditorDialog>
    );
    expect(screen.getByTestId("settings-widgets-editor-title")).toHaveTextContent("New widget");
    expect(screen.getByTestId("body")).toBeInTheDocument();
    expect(screen.getByTestId("settings-widgets-editor-save")).toHaveTextContent("Create");
  });

  it("switches the save label to the replace action in edit mode", () => {
    const props = { ...baseProps(), mode: "edit" as const, title: "Edit widget" };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    expect(screen.getByTestId("settings-widgets-editor-save")).toHaveTextContent("Save changes");
  });

  it("disables save when canSave is false", () => {
    const props = { ...baseProps(), canSave: false };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    expect(screen.getByTestId("settings-widgets-editor-save")).toBeDisabled();
  });

  it("surfaces validation errors inline", () => {
    const props = { ...baseProps(), error: "Field missing" };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    expect(screen.getByTestId("settings-widgets-editor-error")).toHaveTextContent("Field missing");
  });

  it("renders warnings when no error is present", () => {
    const props = { ...baseProps(), warnings: ["restart required", "applied to new sessions"] };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    const warnings = screen.getByTestId("settings-widgets-editor-warnings");
    expect(warnings).toHaveTextContent("restart required");
    expect(warnings).toHaveTextContent("applied to new sessions");
  });

  it("invokes onSave and onOpenChange from the footer controls", () => {
    const props = baseProps();
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    fireEvent.click(screen.getByTestId("settings-widgets-editor-save"));
    expect(props.onSave).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("settings-widgets-editor-cancel"));
    expect(props.onOpenChange).toHaveBeenCalledWith(false);
  });

  it("Should pin the shared ruled header with an accent eyebrow and icon well", () => {
    render(
      <SettingsEditorDialog {...baseProps()}>
        <div />
      </SettingsEditorDialog>
    );
    const dialog = screen.getByTestId("settings-widgets-editor");
    expect(dialog).toHaveAttribute("data-frame", "unframed");
    expect(dialog.querySelector('[data-slot="dialog-header"]')).toHaveAttribute(
      "data-variant",
      "ruled"
    );
    expect(dialog.querySelector('[data-slot="entity-dialog-header-icon"]')).not.toBeNull();
    expect(screen.getByText("System · Widgets")).toBeInTheDocument();
  });

  it("Should render the footer consequence hint on the ruled footer", () => {
    const props = { ...baseProps(), hint: "Stored write-only; AGH returns presence only." };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    const dialog = screen.getByTestId("settings-widgets-editor");
    expect(dialog.querySelector('[data-slot="dialog-footer"]')).toHaveAttribute(
      "data-variant",
      "ruled"
    );
    expect(screen.getByTestId("settings-widgets-editor-hint")).toHaveTextContent(
      "Stored write-only; AGH returns presence only."
    );
  });

  it("Should give the body sole ownership of overflow", () => {
    const props = { ...baseProps(), error: "Field missing." };
    render(
      <SettingsEditorDialog {...props}>
        <div />
      </SettingsEditorDialog>
    );
    const dialog = screen.getByTestId("settings-widgets-editor");
    expect(dialog.className).toContain("grid-rows-[auto_minmax(0,1fr)_auto_auto]");

    const scrollers = Array.from(dialog.querySelectorAll<HTMLElement>("*")).filter(el =>
      el.className.toString().includes("overflow-y-auto")
    );
    expect(scrollers).toHaveLength(1);
    expect(scrollers[0]).toHaveAttribute("data-testid", "settings-widgets-editor-body");
  });

  it("Should omit the feedback region entirely when there is nothing to report", () => {
    render(
      <SettingsEditorDialog {...baseProps()}>
        <div />
      </SettingsEditorDialog>
    );
    // An always-present wrapper would open a dead grid row between body and footer.
    expect(screen.queryByTestId("settings-widgets-editor-feedback")).not.toBeInTheDocument();
  });

  it("Should host the shell on the size token instead of an ad-hoc width", () => {
    render(
      <SettingsEditorDialog {...baseProps()} size="md">
        <div />
      </SettingsEditorDialog>
    );
    const host = screen.getByTestId("settings-widgets-editor");
    for (const token of dialogShellClass("md").split(" ")) {
      expect(host).toHaveClass(token);
    }
  });
});
