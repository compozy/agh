import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// Mock next/link to render as a plain anchor
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    className,
  }: {
    href: string;
    children: React.ReactNode;
    className?: string;
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}));

// Mock next/navigation
vi.mock("next/navigation", () => ({
  usePathname: () => "/",
}));

import { Hero } from "../hero";
import { TwoPillars } from "../two-pillars";
import { HowItWorks } from "../how-it-works";
import { RuntimeFeatures } from "../runtime-features";
import { ProtocolSection } from "../protocol-section";
import { Architecture } from "../architecture";
import { Comparison } from "../comparison";
import { FinalCta } from "../final-cta";

describe("Hero", () => {
  it("renders headline and both CTA buttons", () => {
    render(<Hero />);
    expect(screen.getByText("Durable runtime for real agent work.")).toBeDefined();
    const runtimeLink = screen.getByText("Enter Runtime Docs");
    expect(runtimeLink.closest("a")?.getAttribute("href")).toBe("/runtime");
    const networkLink = screen.getByText("Explore AGH Network");
    expect(networkLink.closest("a")?.getAttribute("href")).toBe("/protocol");
  });

  it("matches snapshot", () => {
    const { container } = render(<Hero />);
    expect(container).toMatchSnapshot();
  });
});

describe("TwoPillars", () => {
  it("renders Runtime and Protocol pillars with links", () => {
    render(<TwoPillars />);
    expect(screen.getByText("Local-first control plane")).toBeDefined();
    expect(screen.getByText("Open coordination layer")).toBeDefined();
    expect(screen.getByText("Get Started with Runtime")).toBeDefined();
    expect(screen.getByText("Explore AGH Network")).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<TwoPillars />);
    expect(container).toMatchSnapshot();
  });
});

describe("HowItWorks", () => {
  it("renders 3 steps with code snippets", () => {
    render(<HowItWorks />);
    expect(screen.getByText("Install the runtime")).toBeDefined();
    expect(screen.getByText("Start the control plane")).toBeDefined();
    expect(screen.getByText("Launch durable work")).toBeDefined();
    expect(screen.getByText(/curl -fsSL/)).toBeDefined();
    expect(screen.getByText("agh daemon start")).toBeDefined();
    expect(screen.getByText("agh session new")).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<HowItWorks />);
    expect(container).toMatchSnapshot();
  });
});

describe("RuntimeFeatures", () => {
  it("renders 8 feature cards", () => {
    render(<RuntimeFeatures />);
    const expectedTitles = [
      "Durable Sessions",
      "Replayable History",
      "Memory That Sticks",
      "Skills Without Glue Code",
      "Workspace-Aware Defaults",
      "Automation and Triggers",
      "Bridges to Real Work",
      "One Operator Surface",
    ];
    for (const title of expectedTitles) {
      expect(screen.getByText(title)).toBeDefined();
    }
  });

  it("matches snapshot", () => {
    const { container } = render(<RuntimeFeatures />);
    expect(container).toMatchSnapshot();
  });
});

describe("ProtocolSection", () => {
  it("renders protocol outcomes and adoption path", () => {
    render(<ProtocolSection />);
    expect(screen.getByText("Coordinated agent work without runtime lock-in.")).toBeDefined();
    expect(screen.getByText("Discover the right specialist")).toBeDefined();
    expect(screen.getByText("Delegate work cleanly")).toBeDefined();
    expect(screen.getByText("Move updates across runtimes")).toBeDefined();
    expect(screen.getAllByText("Keep your runtime")).toHaveLength(2);
    expect(screen.getByText("Map your agents")).toBeDefined();
    expect(screen.getByText("Add deeper profiles later")).toBeDefined();
    expect(screen.getByText("Read the AGH Network docs")).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<ProtocolSection />);
    expect(container).toMatchSnapshot();
  });
});

describe("Architecture", () => {
  it("renders architecture diagram", () => {
    render(<Architecture />);
    expect(
      screen.getByText("One runtime for operator control and open coordination.")
    ).toBeDefined();
    expect(screen.getByLabelText(/AGH runtime diagram/)).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<Architecture />);
    expect(container).toMatchSnapshot();
  });
});

describe("Comparison", () => {
  it("renders named market comparison cards", () => {
    render(<Comparison />);
    expect(screen.getByText("Where AGH fits in the landscape.")).toBeDefined();
    const expectedCards = ["OpenClaw", "OpenFang", "GoClaw", "AGH"];
    for (const name of expectedCards) {
      expect(screen.getByText(name)).toBeDefined();
    }
  });

  it("matches snapshot", () => {
    const { container } = render(<Comparison />);
    expect(container).toMatchSnapshot();
  });
});

describe("FinalCta", () => {
  it("renders dual CTA buttons", () => {
    render(<FinalCta />);
    expect(screen.getByText("Start with the runtime. Grow into AGH Network.")).toBeDefined();
    const runtimeLink = screen.getByText("Enter Runtime Docs");
    expect(runtimeLink.closest("a")?.getAttribute("href")).toBe("/runtime");
    const networkLink = screen.getByText("Explore AGH Network");
    expect(networkLink.closest("a")?.getAttribute("href")).toBe("/protocol");
  });

  it("matches snapshot", () => {
    const { container } = render(<FinalCta />);
    expect(container).toMatchSnapshot();
  });
});
