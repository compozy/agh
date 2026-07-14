import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { OnboardingNetworkMention } from "../onboarding-network-mention";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    ...rest
  }: {
    children: ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...(rest as Record<string, unknown>)}>
      {children}
    </a>
  ),
}));

describe("OnboardingNetworkMention", () => {
  it("Should link to Network without implying settings mutation", () => {
    render(<OnboardingNetworkMention />);
    const link = screen.getByTestId("onboarding-network-link");
    expect(link).toHaveAttribute("href", "/network");
    expect(screen.getByTestId("onboarding-network-mention")).toHaveTextContent(
      /does not enable coordination or change settings/i
    );
  });
});
