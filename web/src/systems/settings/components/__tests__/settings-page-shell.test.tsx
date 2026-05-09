import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PageShell, Section } from "@agh/ui";
import { SettingsSaveBar } from "../settings-save-bar";

describe("PageShell", () => {
  it("renders the SETTINGS eyebrow + H1 title + actions slot", () => {
    render(
      <PageShell
        slug="general"
        title="General"
        actions={<button data-testid="header-action">Restart</button>}
      >
        <p>body</p>
      </PageShell>
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
      <PageShell
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
      </PageShell>
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
      <PageShell
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
      </PageShell>
    );

    const body = screen.getByTestId("settings-page-network-body");
    const footer = screen.getByTestId("settings-page-network-footer");

    expect(within(body).queryByTestId("settings-page-network-save-bar")).not.toBeInTheDocument();
    expect(footer).toContainElement(screen.getByTestId("settings-page-network-save-bar"));
  });

  it("omits the banner slot and footer when no content is provided", () => {
    render(
      <PageShell slug="memory" title="Memory">
        <span>body</span>
      </PageShell>
    );

    expect(screen.queryByTestId("settings-page-memory-banner-slot")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-page-memory-footer")).not.toBeInTheDocument();
  });

  it("renders a status line region when provided", () => {
    render(
      <PageShell
        slug="general"
        title="General"
        statusLine={<span data-testid="shell-status">daemon ok</span>}
      >
        <span>body</span>
      </PageShell>
    );

    expect(screen.getByTestId("settings-page-general-status")).toContainElement(
      screen.getByTestId("shell-status")
    );
  });

  it("allows overriding the eyebrow prefix", () => {
    render(
      <PageShell slug="general" title="General" eyebrow="Admin">
        <span>body</span>
      </PageShell>
    );

    expect(screen.getByTestId("settings-page-general-eyebrow")).toHaveTextContent(
      "Admin / General"
    );
  });

  it("uses the mono tracking token and responsive shell spacing", () => {
    render(
      <PageShell slug="general" title="General">
        <span>body</span>
      </PageShell>
    );

    const eyebrow = screen.getByTestId("settings-page-general-eyebrow");
    const header = screen.getByTestId("settings-page-general-header");
    const body = screen.getByTestId("settings-page-general-body");

    expect(eyebrow.className).toContain("tracking-mono");
    expect(header.className).toContain("px-4");
    expect(header.className).toContain("sm:px-6");
    expect(body.className).toContain("md:px-8");
  });

  it("uses the mono tracking token for section-card eyebrows", () => {
    render(
      <Section divided label="Runtime">
        <span>content</span>
      </Section>
    );

    expect(screen.getByText("Runtime").className).toContain("tracking-mono");
  });
});
