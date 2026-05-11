import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PageShell, Section } from "@agh/ui";
import { SettingsSaveBar } from "../settings-save-bar";

describe("PageShell", () => {
  it("renders the banner, scroll body, and footer as separate layout bands", () => {
    render(
      <PageShell
        slug="general"
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

    expect(screen.getByTestId("settings-page-general")).toBeInTheDocument();
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
      <PageShell slug="memory">
        <span>body</span>
      </PageShell>
    );

    expect(screen.queryByTestId("settings-page-memory-banner-slot")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-page-memory-footer")).not.toBeInTheDocument();
  });

  it("does not own the route title (the shell topbar does)", () => {
    render(
      <PageShell slug="general">
        <span>body</span>
      </PageShell>
    );

    expect(screen.queryByRole("heading", { level: 1 })).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-page-general-header")).not.toBeInTheDocument();
  });

  it("renders the section title with the canonical --tracking-section-head token (ADR-003 §2)", () => {
    render(
      <Section divided label="Runtime">
        <span>content</span>
      </Section>
    );

    const heading = screen.getByText("Runtime");
    expect(heading.className).toContain("tracking-(--tracking-section-head)");
    expect(heading.className).toContain("text-(length:--text-section-head)");
    expect(heading.className).not.toContain("tracking-[-0.026em]");
  });
});
