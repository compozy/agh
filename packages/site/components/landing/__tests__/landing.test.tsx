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
    expect(screen.getByText("Your agents can finally talk to each other.")).toBeDefined();
    const protocolLink = screen.getByText("Read the Protocol Spec");
    expect(protocolLink.closest("a")).toHaveProperty("href");
    expect(protocolLink.closest("a")?.getAttribute("href")).toBe("/protocol");
    const runtimeLink = screen.getByText("Get Started");
    expect(runtimeLink.closest("a")?.getAttribute("href")).toBe("/runtime");
  });

  it("matches snapshot", () => {
    const { container } = render(<Hero />);
    expect(container).toMatchSnapshot();
  });
});

describe("TwoPillars", () => {
  it("renders Runtime and Protocol pillars with links", () => {
    render(<TwoPillars />);
    expect(screen.getByText("Agent Operating System")).toBeDefined();
    expect(screen.getByText("Agent Network Protocol")).toBeDefined();
    expect(screen.getByText("Explore the Runtime")).toBeDefined();
    expect(screen.getByText("Read the Protocol Spec")).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<TwoPillars />);
    expect(container).toMatchSnapshot();
  });
});

describe("HowItWorks", () => {
  it("renders 3 steps with code snippets", () => {
    render(<HowItWorks />);
    expect(screen.getByText("Install AGH")).toBeDefined();
    expect(screen.getByText("Start the daemon")).toBeDefined();
    expect(screen.getByText("Create your first session")).toBeDefined();
    expect(screen.getByText(/curl -fsSL/)).toBeDefined();
    expect(screen.getByText("agh daemon start")).toBeDefined();
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
      "Session Lifecycle",
      "Persistent Memory",
      "Skills System",
      "Workspace Isolation",
      "Jobs & Triggers",
      "Platform Bridges",
      "Event Hooks",
      "Extension System",
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
  it("renders 7 message kinds", () => {
    render(<ProtocolSection />);
    const kinds = ["request", "response", "notify", "discover", "subscribe", "cancel", "error"];
    for (const kind of kinds) {
      expect(screen.getByText(kind)).toBeDefined();
    }
  });

  it("matches snapshot", () => {
    const { container } = render(<ProtocolSection />);
    expect(container).toMatchSnapshot();
  });
});

describe("Architecture", () => {
  it("renders architecture diagram", () => {
    render(<Architecture />);
    expect(screen.getByText("How it all fits together")).toBeDefined();
    expect(screen.getByLabelText(/AGH architecture diagram/)).toBeDefined();
  });

  it("matches snapshot", () => {
    const { container } = render(<Architecture />);
    expect(container).toMatchSnapshot();
  });
});

describe("Comparison", () => {
  it("renders comparison table with at least 5 rows", () => {
    render(<Comparison />);
    expect(screen.getByText("AGH vs typical agent harness")).toBeDefined();
    const expectedFeatures = [
      "Agent execution",
      "Agent-to-agent communication",
      "Session persistence",
      "Memory system",
      "Deployment",
      "Configuration",
      "Extensibility",
      "Observability",
    ];
    for (const feature of expectedFeatures) {
      // Each feature appears twice (desktop table + mobile cards)
      expect(screen.getAllByText(feature).length).toBeGreaterThanOrEqual(1);
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
    expect(screen.getByText("Ready to connect your agents?")).toBeDefined();
    const protocolLink = screen.getByText("Read the Protocol Spec");
    expect(protocolLink.closest("a")?.getAttribute("href")).toBe("/protocol");
    const runtimeLink = screen.getByText("Get Started");
    expect(runtimeLink.closest("a")?.getAttribute("href")).toBe("/runtime");
  });

  it("matches snapshot", () => {
    const { container } = render(<FinalCta />);
    expect(container).toMatchSnapshot();
  });
});
