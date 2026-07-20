import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Logo } from "../logo";

describe("Logo", () => {
  it("renders the full logo variant by default", () => {
    render(<Logo data-testid="logo" />);

    const logo = screen.getByRole("img", { name: "AGH" });
    expect(logo).toHaveAttribute("data-slot", "logo");
    expect(logo).toHaveAttribute("data-variant", "logo");
    expect(logo).toHaveAttribute("viewBox", "0 0 972 386");
    expect(logo.querySelector('g[transform="translate(0 30.64)"]')).not.toBeNull();
    expect(logo.querySelector('g[transform="translate(429 0)"]')).not.toBeNull();
  });

  it("renders the symbol variant with the square symbol viewBox", () => {
    const { container } = render(<Logo variant="symbol" label="AGH symbol" />);

    const logo = screen.getByRole("img", { name: "AGH symbol" });
    expect(logo).toHaveAttribute("data-variant", "symbol");
    expect(logo).toHaveAttribute("viewBox", "0 0 355 355");
    expect(container.querySelector('rect[fill="#E8572A"]')).not.toBeNull();
  });

  it("renders the lettering variant with the lettering viewBox", () => {
    render(<Logo variant="lettering" label="AGH lettering" />);

    const logo = screen.getByRole("img", { name: "AGH lettering" });
    expect(logo).toHaveAttribute("data-variant", "lettering");
    expect(logo).toHaveAttribute("viewBox", "0 0 543 362");
    expect(logo.querySelector('path[fill="white"]')).not.toBeNull();
    expect(logo.querySelector("rect")).toBeNull();
  });

  it("renders the glyph variant with the compact square viewBox", () => {
    const { container } = render(<Logo variant="glyph" label="AGH glyph" />);

    const logo = screen.getByRole("img", { name: "AGH glyph" });
    expect(logo).toHaveAttribute("data-variant", "glyph");
    expect(logo).toHaveAttribute("viewBox", "0 0 32 32");
    expect(container.querySelector('rect[fill="#E8572A"]')).not.toBeNull();
  });

  it("renders the menubar variant as the OpenDesign mb-logo shell mark", () => {
    const { container } = render(<Logo variant="menubar" label="AGH menubar" />);

    const logo = screen.getByRole("img", { name: "AGH menubar" });
    expect(logo).toHaveAttribute("data-variant", "menubar");
    expect(logo).toHaveAttribute("viewBox", "0 0 16 16");
    expect(logo.getAttribute("class")).toContain("size-menubar-logo");
    const tile = container.querySelector('rect[fill="currentColor"]');
    expect(tile).not.toBeNull();
    expect(tile).toHaveAttribute("x", "1.5");
    expect(tile).toHaveAttribute("y", "1.5");
    expect(tile).toHaveAttribute("width", "13");
    expect(tile).toHaveAttribute("height", "13");
    expect(tile).toHaveAttribute("rx", "4");
    const dot = container.querySelector('circle[fill="var(--color-accent-ink)"]');
    expect(dot).not.toBeNull();
    expect(dot).toHaveAttribute("fill", "var(--color-accent-ink)");
    expect(dot).toHaveAttribute("cx", "8");
    expect(dot).toHaveAttribute("cy", "8");
    expect(dot).toHaveAttribute("r", "2.6");
  });

  it("uses an explicit aria-label when provided", () => {
    render(<Logo aria-label="AGH home" />);

    expect(screen.getByRole("img", { name: "AGH home" })).toBeInTheDocument();
  });

  it("can render as decorative artwork", () => {
    render(<Logo decorative data-testid="logo" />);

    const logo = screen.getByTestId("logo");
    expect(screen.queryByRole("img")).toBeNull();
    expect(logo).toHaveAttribute("aria-hidden", "true");
    expect(logo).not.toHaveAttribute("aria-label");
  });

  it("passes className and style through to the svg", () => {
    render(<Logo className="size-7 opacity-80" style={{ opacity: 0.5 }} data-testid="logo" />);

    const logo = screen.getByTestId("logo");
    expect(logo.getAttribute("class")).toContain("size-7");
    expect(logo).toHaveStyle({ opacity: "0.5" });
  });
});
