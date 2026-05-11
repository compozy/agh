import { render, screen } from "@testing-library/react";
import { Wrench } from "lucide-react";
import { describe, expect, it } from "vitest";

import { Button } from "../../button";
import { CatalogCard } from "../catalog-card";

describe("CatalogCard", () => {
  it("Should compose logo, title, description, meta, and actions slots in order", () => {
    render(
      <CatalogCard data-testid="catalog-card" actionable selected>
        <div className="flex items-start gap-3">
          <CatalogCard.Logo data-testid="catalog-logo">
            <Wrench className="size-4" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title data-testid="catalog-title">skill-pack</CatalogCard.Title>
            <CatalogCard.Meta data-testid="catalog-meta">
              <span>@operator</span>
              <span>v1.2.3</span>
            </CatalogCard.Meta>
          </div>
        </div>
        <CatalogCard.Description data-testid="catalog-description">
          Shared marketplace card.
        </CatalogCard.Description>
        <CatalogCard.Actions data-testid="catalog-actions">
          <Button size="sm" type="button">
            Install
          </Button>
        </CatalogCard.Actions>
      </CatalogCard>
    );

    const card = screen.getByTestId("catalog-card");
    const logo = screen.getByTestId("catalog-logo");
    const title = screen.getByTestId("catalog-title");
    const meta = screen.getByTestId("catalog-meta");
    const description = screen.getByTestId("catalog-description");
    const actions = screen.getByTestId("catalog-actions");

    expect(card).toHaveAttribute("data-selected", "true");
    expect(card).toHaveAttribute("data-actionable", "true");
    expect(logo).toHaveAttribute("data-slot", "catalog-card-logo");
    expect(title).toHaveTextContent("skill-pack");
    expect(meta).toHaveTextContent("@operator");
    expect(description).toHaveTextContent("Shared marketplace card.");
    expect(actions).toHaveTextContent("Install");
    expect(title.compareDocumentPosition(meta)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(description.compareDocumentPosition(actions)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it.each(["accent", "neutral", "success", "warning", "danger", "info"] as const)(
    "Should expose %s logo tone through data attributes",
    tone => {
      render(
        <CatalogCard>
          <CatalogCard.Logo data-testid="logo" tone={tone} />
        </CatalogCard>
      );

      expect(screen.getByTestId("logo")).toHaveAttribute("data-tone", tone);
    }
  );

  it("Should render flat — 16 px padding, no border, no accent on resting state", () => {
    render(<CatalogCard data-testid="catalog-card" />);
    const card = screen.getByTestId("catalog-card");
    expect(card.className).toContain("p-4");
    expect(card.className).not.toContain("border-(--line)");
    expect(card.className).not.toContain("border-(--accent)");
    expect(card.className).not.toContain("bg-(--accent-tint)");
    expect(card.className).toContain("bg-(--canvas-soft)");
  });

  it("Should hover into --elevated when `actionable`", () => {
    render(<CatalogCard data-testid="catalog-card" actionable />);
    const card = screen.getByTestId("catalog-card");
    expect(card.className).toContain("hover:bg-(--elevated)");
  });

  it("Should paint --surface-glaze + 1 px inset --line-strong ring on selected state (no accent)", () => {
    render(<CatalogCard data-testid="catalog-card" selected />);
    const card = screen.getByTestId("catalog-card");
    expect(card.className).toContain("bg-(--surface-glaze)");
    expect(card.className).toContain("shadow-[inset_0_0_0_1px_var(--line-strong)]");
    expect(card.className).not.toContain("border-(--accent)");
    expect(card.className).not.toContain("bg-(--accent-tint)");
  });

  it("Should size the icon-well at --size-catalog-logo (24 px) by default", () => {
    render(
      <CatalogCard>
        <CatalogCard.Logo data-testid="logo" />
      </CatalogCard>
    );
    const logo = screen.getByTestId("logo");
    expect(logo.className).toContain("size-(--size-catalog-logo)");
    expect(logo.className).toContain("bg-(--surface-glaze)");
    expect(logo.className).toContain("rounded-(--radius)");
    expect(logo).toHaveAttribute("data-size", "default");
  });

  it("Should size the icon-well at --size-provider-logo-well (40 px) when logoSize='lg'", () => {
    render(
      <CatalogCard>
        <CatalogCard.Logo data-testid="logo" size="lg" />
      </CatalogCard>
    );
    const logo = screen.getByTestId("logo");
    expect(logo.className).toContain("size-(--size-provider-logo-well)");
    expect(logo).toHaveAttribute("data-size", "lg");
  });

  it("Should render the title at 13 px / 510 / -0.012em", () => {
    render(
      <CatalogCard>
        <CatalogCard.Title data-testid="title">release-auditor</CatalogCard.Title>
      </CatalogCard>
    );
    const title = screen.getByTestId("title");
    expect(title.className).toContain("text-[13px]");
    expect(title.className).toContain("tracking-[-0.012em]");
  });
});
