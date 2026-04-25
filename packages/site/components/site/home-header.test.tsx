import { render, screen } from "@testing-library/react";
import type { AnchorHTMLAttributes, ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { HomeHeader } from "./home-header";

vi.mock("next/navigation", () => ({
  usePathname: () => "/",
}));

vi.mock("fumadocs-ui/layouts/home", () => ({
  useHomeLayout: () => ({
    slots: {},
  }),
}));

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: {
    href: string;
    children: ReactNode;
  } & Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href">) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("HomeHeader", () => {
  it("renders the shared full AGH logo in the home link", () => {
    render(<HomeHeader />);

    const link = screen.getByRole("link", { name: "AGH home" });
    const logo = link.querySelector('[data-slot="logo"]');

    expect(link.getAttribute("href")).toBe("/");
    expect(logo).not.toBeNull();
    expect(logo?.getAttribute("data-variant")).toBe("logo");
    expect(logo?.getAttribute("viewBox")).toBe("0 0 972 386");
    expect(logo?.getAttribute("aria-hidden")).toBe("true");
  });
});
