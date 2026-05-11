import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Button, buttonVariants } from "../button";

type ButtonVariant = NonNullable<Parameters<typeof buttonVariants>[0]>["variant"];
type ButtonSize = NonNullable<Parameters<typeof buttonVariants>[0]>["size"];

const VARIANTS: ButtonVariant[] = [
  "default",
  "primary",
  "outline",
  "secondary",
  "ghost",
  "destructive",
  "success",
  "link",
  "neutral",
];

const SIZES: ButtonSize[] = [
  "default",
  "xs",
  "sm",
  "lg",
  "cta",
  "cta-lg",
  "icon",
  "icon-xs",
  "icon-sm",
  "icon-lg",
];

describe("Button", () => {
  it('Should render a `data-slot="button"` root with default + default classes', () => {
    render(<Button>Action</Button>);
    const button = screen.getByRole("button", { name: /action/i });
    expect(button).toHaveAttribute("data-slot", "button");
    expect(button.className).toContain("bg-(--accent)");
    expect(button.className).toContain("h-[26px]");
  });

  it("Should emit identical class strings for variant='primary' and variant='default' (ADR-004 §1 initial parity)", () => {
    render(
      <>
        <Button variant="default" data-testid="default">
          A
        </Button>
        <Button variant="primary" data-testid="primary">
          B
        </Button>
      </>
    );
    const defaultBtn = screen.getByTestId("default");
    const primaryBtn = screen.getByTestId("primary");
    expect(primaryBtn.className).toBe(defaultBtn.className);
  });

  it("Should emit the neutral fill/hover token tuple for variant='neutral'", () => {
    render(<Button variant="neutral">N</Button>);
    const button = screen.getByRole("button", { name: /n/i });
    expect(button.className).toContain("bg-(--btn-default-fill)");
    expect(button.className).toContain("hover:bg-(--btn-default-hover)");
    expect(button.className).toContain("text-(--fg-strong)");
    expect(button.className).not.toContain("border-(--line)");
  });

  it("Should still compile and render variant='outline'", () => {
    render(<Button variant="outline">O</Button>);
    const button = screen.getByRole("button", { name: /o/i });
    expect(button.className).toContain("border-(--line)");
    expect(button.className).toContain("bg-transparent");
  });

  it("Should still compile and render variant='secondary' and variant='link'", () => {
    render(
      <>
        <Button variant="secondary" data-testid="secondary">
          S
        </Button>
        <Button variant="link" data-testid="link">
          L
        </Button>
      </>
    );
    expect(screen.getByTestId("secondary").className).toContain("bg-(--canvas-tint)");
    expect(screen.getByTestId("link").className).toContain("underline-offset-4");
  });

  it("Should still compile and render size='xs' and size='icon-xs' (regression — sizes retained)", () => {
    render(
      <>
        <Button size="xs" data-testid="xs">
          XS
        </Button>
        <Button size="icon-xs" data-testid="icon-xs">
          *
        </Button>
      </>
    );
    expect(screen.getByTestId("xs").className).toContain("h-[22px]");
    expect(screen.getByTestId("icon-xs").className).toContain("size-[22px]");
  });

  it("Should mark itself as disabled when the disabled prop is set", () => {
    render(<Button disabled>D</Button>);
    const button = screen.getByRole("button", { name: /d/i });
    expect(button).toBeDisabled();
    expect(button.className).toContain("disabled:opacity-50");
  });

  it("Should forward className alongside variant defaults", () => {
    render(<Button className="custom-tail">F</Button>);
    expect(screen.getByRole("button", { name: /f/i }).className).toContain("custom-tail");
  });
});

describe("Button class snapshot matrix", () => {
  for (const variant of VARIANTS) {
    for (const size of SIZES) {
      it(`Should lock the ${variant} × ${size} class block`, () => {
        const testId = `btn-${variant}-${size}`;
        render(
          <Button variant={variant} size={size} data-testid={testId}>
            x
          </Button>
        );
        const button = screen.getByTestId(testId);
        expect(button.className).toMatchSnapshot();
      });
    }
  }
});
