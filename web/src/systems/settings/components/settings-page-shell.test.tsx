import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SettingsPageShell } from "./settings-page-shell";
import { SettingsSaveBar } from "./settings-save-bar";

describe("SettingsPageShell", () => {
  it("renders the SETTINGS eyebrow + H1 title + actions slot", () => {
    render(
      <SettingsPageShell
        slug="general"
        title="General"
        actions={<button data-testid="header-action">Restart</button>}
      >
        <p>body</p>
      </SettingsPageShell>
    );

    const eyebrow = screen.getByTestId("settings-page-general-eyebrow");
    expect(eyebrow).toHaveTextContent("Settings / General");
    expect(screen.getByRole("heading", { level: 1, name: "General" })).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-general-actions")).toContainElement(
      screen.getByTestId("header-action")
    );
  });

  it("renders the banner, scroll body, and footer as separate layout bands", () => {
    render(
      <SettingsPageShell
        slug="general"
        title="General"
        banner={<div data-testid="shell-banner">banner</div>}
        footer={
          <SettingsSaveBar
            slug="general"
            isDirty={true}
            isSaving={false}
            onSave={() => {}}
            onReset={() => {}}
          />
        }
      >
        <div data-testid="shell-body-content">content</div>
      </SettingsPageShell>
    );

    expect(screen.getByTestId("settings-page-general-header")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-general-banner-slot")).toContainElement(
      screen.getByTestId("shell-banner")
    );
    expect(screen.getByTestId("settings-page-general-body")).toContainElement(
      screen.getByTestId("shell-body-content")
    );
    expect(screen.getByTestId("settings-page-general-footer")).toContainElement(
      screen.getByTestId("settings-page-general-save-bar")
    );
  });

  it("keeps the save bar outside of the scroll body", () => {
    render(
      <SettingsPageShell
        slug="network"
        title="Network"
        footer={
          <SettingsSaveBar
            slug="network"
            isDirty={false}
            isSaving={false}
            onSave={() => {}}
            onReset={() => {}}
          />
        }
      >
        <div data-testid="shell-network-body-content">content</div>
      </SettingsPageShell>
    );

    const body = screen.getByTestId("settings-page-network-body");
    const footer = screen.getByTestId("settings-page-network-footer");

    expect(within(body).queryByTestId("settings-page-network-save-bar")).not.toBeInTheDocument();
    expect(footer).toContainElement(screen.getByTestId("settings-page-network-save-bar"));
  });

  it("omits the banner slot and footer when no content is provided", () => {
    render(
      <SettingsPageShell slug="memory" title="Memory">
        <span>body</span>
      </SettingsPageShell>
    );

    expect(screen.queryByTestId("settings-page-memory-banner-slot")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-page-memory-footer")).not.toBeInTheDocument();
  });

  it("renders a status line region when provided", () => {
    render(
      <SettingsPageShell
        slug="general"
        title="General"
        statusLine={<span data-testid="shell-status">daemon ok</span>}
      >
        <span>body</span>
      </SettingsPageShell>
    );

    expect(screen.getByTestId("settings-page-general-status")).toContainElement(
      screen.getByTestId("shell-status")
    );
  });

  it("allows overriding the eyebrow prefix", () => {
    render(
      <SettingsPageShell slug="general" title="General" eyebrow="Admin">
        <span>body</span>
      </SettingsPageShell>
    );

    expect(screen.getByTestId("settings-page-general-eyebrow")).toHaveTextContent(
      "Admin / General"
    );
  });
});
