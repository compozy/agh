import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SettingsSaveBar } from "../settings-save-bar";

describe("SettingsSaveBar", () => {
  it("renders nothing while the page is clean", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={false}
        isSaving={false}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.queryByTestId("settings-page-general-save-bar")).not.toBeInTheDocument();
  });

  it("appears when dirty and fires onSave from the enabled Save button", () => {
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

    expect(screen.getByTestId("settings-page-general-save-bar")).toHaveAttribute(
      "data-dirty",
      "true"
    );
    expect(screen.getByTestId("settings-page-general-save-message")).toHaveTextContent(
      "Unsaved changes"
    );
    const save = screen.getByTestId("settings-page-general-save");
    expect(save).not.toBeDisabled();
    fireEvent.click(save);
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("disables Save and names the blocker when invalid even if dirty", () => {
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
    expect(screen.getByTestId("settings-page-general-save-message")).toHaveTextContent(
      "Resolve validation errors before saving"
    );
  });

  it("names the saving state and disables both actions while saving", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={true}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save-message")).toHaveTextContent("Saving…");
    expect(screen.getByTestId("settings-page-general-save")).toBeDisabled();
    expect(screen.getByTestId("settings-page-general-reset")).toBeDisabled();
  });

  it("renders the error message through an assertive live region", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        error="config write failed"
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    const bar = screen.getByTestId("settings-page-general-save-bar");
    expect(bar).toHaveAttribute("aria-live", "assertive");
    expect(screen.getByTestId("settings-page-general-save-message")).toHaveTextContent(
      "config write failed"
    );
  });

  it("flashes the applied label after a clean save, then dismisses", () => {
    vi.useFakeTimers();
    try {
      const { rerender } = render(
        <SettingsSaveBar
          slug="general"
          isDirty={true}
          isSaving={true}
          onSave={vi.fn()}
          onReset={vi.fn()}
        />
      );

      rerender(
        <SettingsSaveBar
          slug="general"
          isDirty={false}
          isSaving={false}
          lastAppliedLabel="Applied 2 fields"
          onSave={vi.fn()}
          onReset={vi.fn()}
        />
      );

      expect(screen.getByTestId("settings-page-general-save-message")).toHaveTextContent(
        "Applied 2 fields"
      );

      vi.advanceTimersByTime(2000);
      rerender(
        <SettingsSaveBar
          slug="general"
          isDirty={false}
          isSaving={false}
          lastAppliedLabel="Applied 2 fields"
          onSave={vi.fn()}
          onReset={vi.fn()}
        />
      );
      expect(screen.queryByTestId("settings-page-general-save-bar")).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("keeps warnings visible alongside the dirty state", () => {
    render(
      <SettingsSaveBar
        slug="general"
        isDirty={true}
        isSaving={false}
        warnings={["value clamped to 3600"]}
        onSave={vi.fn()}
        onReset={vi.fn()}
      />
    );

    expect(screen.getByTestId("settings-page-general-save-warnings")).toHaveTextContent(
      "value clamped to 3600"
    );
  });
});
