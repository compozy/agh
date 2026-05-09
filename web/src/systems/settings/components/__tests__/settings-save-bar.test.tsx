import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsSaveBar } from "../settings-save-bar";

describe("SettingsSaveBar", () => {
  it("disables the Save and Discard buttons when nothing is dirty", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save")).toBeDisabled();
    expect(screen.getByTestId("settings-page-general-reset")).toBeDisabled();
    expect(screen.getByTestId("settings-page-general-save-bar")).toHaveAttribute(
      "data-dirty",
      "false"
    );
  });

  it("enables the Save button when dirty and not saving or invalid", () => {
    const onSave = vi.fn();
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        isInvalid={false}
        onSave={onSave}
        onReset={vi.fn()}
      />
    );

    const save = screen.getByTestId("settings-page-general-save");
    expect(save).not.toBeDisabled();
    fireEvent.click(save);
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("disables Save when invalid even if dirty", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        isInvalid={true}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save")).toBeDisabled();
    expect(screen.getByTestId("settings-page-general-save-invalid")).toHaveTextContent(
      "Resolve validation errors before saving"
    );
  });

  it("shows the Saving label + spinner while saving and re-enables after isSaving flips to false", () => {
    const { rerender } = render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={true}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const save = screen.getByTestId("settings-page-general-save");
    expect(save).toBeDisabled();
    expect(save).toHaveTextContent("Saving...");

    rerender(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save")).not.toBeDisabled();
    expect(screen.getByTestId("settings-page-general-save")).toHaveTextContent("Save changes");
  });

  it("renders the error line in the danger tone when error is non-null", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        error="boom"
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const errorLine = screen.getByTestId("settings-page-general-save-error");
    expect(errorLine).toHaveTextContent("boom");
    expect(errorLine.className).toContain("text-(--color-danger)");
    expect(screen.queryByTestId("settings-page-general-save-warnings")).not.toBeInTheDocument();
  });

  it("renders the warnings list in the warning tone when error is null and warnings exist", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        warnings={["restart required", "env missing"]}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const warnings = screen.getByTestId("settings-page-general-save-warnings");
    expect(warnings).toHaveTextContent("restart required");
    expect(warnings).toHaveTextContent("env missing");
    expect(warnings.className).toContain("text-(--color-warning)");
  });

  it("renders the lastAppliedLabel with a success check when no error or warnings and not dirty", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        lastAppliedLabel="Applied 2m ago"
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const applied = screen.getByTestId("settings-page-general-save-applied");
    expect(applied).toHaveTextContent("Applied 2m ago");
  });

  it("prioritizes dirty messaging over a stale applied label", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        lastAppliedLabel="Applied 2m ago"
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save-dirty")).toHaveTextContent(
      "Unsaved changes"
    );
    expect(screen.queryByTestId("settings-page-general-save-applied")).not.toBeInTheDocument();
  });

  it("announces save state changes through a live region", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        lastAppliedLabel="Applied 2m ago"
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite");
  });

  it("uses the same responsive horizontal spacing as the settings shell", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const bar = screen.getByTestId("settings-page-general-save-bar");
    expect(bar.className).toContain("px-4");
    expect(bar.className).toContain("sm:px-6");
    expect(bar.className).toContain("md:px-8");
  });

  it("fires onReset when Discard is clicked", () => {
    const onReset = vi.fn();
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        onSave={vi.fn()}
        onReset={onReset}
      />
    );

    fireEvent.click(screen.getByTestId("settings-page-general-reset"));
    expect(onReset).toHaveBeenCalledTimes(1);
  });
});
