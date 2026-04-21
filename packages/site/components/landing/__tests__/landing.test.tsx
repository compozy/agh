import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { baseOptions } from "@/lib/layout.shared";

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
import { FeaturesSection } from "../features-section";
import { SupportedAgents } from "../supported-agents";
import { RuntimeMicroDiagram } from "../runtime-micro-diagram";
import { RuntimeSection } from "../runtime-section";
import { BridgesSection } from "../bridges-section";
import { ExtensibilitySection } from "../extensibility-section";
import { NetworkSection } from "../network-section";
import { InstallSection } from "../install-section";
import { Comparison } from "../comparison";
import { FinalCta } from "../final-cta";
import { KindChip, KIND_MEANING, type NetworkKind } from "../primitives/kind-chip";

describe("Hero", () => {
  it("leads with the runtime + network headline and drops ACP from the fold", () => {
    render(<Hero />);
    expect(screen.getByText("An agent runtime with a network built in.")).toBeDefined();
    const install = screen.getByText("Install the runtime");
    expect(install.closest("a")?.getAttribute("href")).toBe(
      "/runtime/core/getting-started/installation"
    );
    const network = screen.getByText("See the network");
    expect(network.closest("a")?.getAttribute("href")).toBe("/protocol");
  });

  it("renders four outcome-framed signal tiles", () => {
    render(<Hero />);
    expect(screen.getByText("Complete agent runtime")).toBeDefined();
    expect(screen.getByText("Built-in agent network")).toBeDefined();
    expect(screen.getByText("Local-first, self-hosted")).toBeDefined();
    expect(screen.getByText("Open protocol, open source")).toBeDefined();
  });
});

describe("FeaturesSection", () => {
  it("renders the eight runtime capabilities", () => {
    render(<FeaturesSection />);
    const eyebrows = [
      "Sessions",
      "Memory",
      "Skills",
      "Workspaces",
      "Automation",
      "Observability",
      "Hooks",
      "Bridges",
    ];
    for (const label of eyebrows) {
      expect(screen.getByText(label)).toBeDefined();
    }
    expect(screen.getByText("Everything a modern agent runtime should have.")).toBeDefined();
  });
});

describe("SupportedAgents", () => {
  it("renders as a compact support strip, not a hero section", () => {
    render(<SupportedAgents />);
    const expected = ["claude", "codex", "gemini", "opencode", "copilot", "cursor", "kiro", "pi"];
    for (const id of expected) {
      expect(screen.getByText(id)).toBeDefined();
    }
    expect(screen.getByText("Works with your agent CLIs")).toBeDefined();
  });
});

describe("RuntimeSection", () => {
  it("renders the four runtime feature cards", () => {
    render(<RuntimeSection />);
    const expected = [
      "Durable sessions in SQLite",
      "Replayable event stream",
      "Three operator surfaces, one daemon",
      "Permission modes with an audit trail",
    ];
    for (const title of expected) {
      expect(screen.getByText(title)).toBeDefined();
    }
  });

  it("gives the sticky runtime rail a large-screen top inset", () => {
    const { container } = render(<RuntimeSection />);
    const stickyRail = container.querySelector('div[class*="lg:sticky"]');

    expect(stickyRail).toBeTruthy();
    expect(stickyRail?.getAttribute("class")).toContain("lg:top-24");
  });
});

describe("RuntimeMicroDiagram", () => {
  it("injects a prefers-reduced-motion CSS guard for pre-hydration renders", () => {
    const { container } = render(<RuntimeMicroDiagram />);
    const styleText = container.querySelector("style")?.textContent ?? "";

    expect(styleText).toContain("@media (prefers-reduced-motion: reduce)");
    expect(styleText).toContain("animation: none !important;");
  });
});

describe("BridgesSection", () => {
  it("renders the live bridges with brand logos and the catalogued set", () => {
    render(<BridgesSection />);
    const expected = [
      "Slack",
      "Discord",
      "Telegram",
      "WhatsApp",
      "Microsoft Teams",
      "Google Chat",
      "GitHub",
      "Linear",
    ];
    for (const name of expected) {
      expect(screen.getByText(name)).toBeDefined();
    }
    expect(screen.getByText("Your users live on these. Now so do your agents.")).toBeDefined();
  });

  it("marks the three live bridges separately from the next batch", () => {
    render(<BridgesSection />);
    expect(screen.getAllByText("live").length).toBe(3);
    expect(screen.getAllByText("next").length).toBe(5);
  });
});

describe("ExtensibilitySection", () => {
  it("renders five extensibility cards", () => {
    render(<ExtensibilitySection />);
    const eyebrows = ["Hooks", "Skills", "Memory", "Automation", "Extensions"];
    for (const label of eyebrows) {
      expect(screen.getByText(label)).toBeDefined();
    }
  });
});

describe("NetworkSection", () => {
  it("renders the protocol walkthrough and supporting cards", () => {
    render(<NetworkSection />);
    expect(screen.getByText("Real commands, not docs-ware")).toBeDefined();
    expect(screen.getByText("NATS under the hood, JSON over the wire")).toBeDefined();
    expect(screen.getByText("Receipts are first-class")).toBeDefined();
    expect(screen.getByLabelText(/Pause walkthrough|Play walkthrough/)).toBeDefined();
  });
});

describe("InstallSection", () => {
  it("renders three install tabs and the three CLI steps", () => {
    render(<InstallSection />);
    expect(screen.getByRole("tab", { name: "go install" })).toBeDefined();
    expect(screen.getByRole("tab", { name: "Build from source" })).toBeDefined();
    expect(screen.getByText("Bootstrap your AGH home")).toBeDefined();
    expect(screen.getByText("Start the daemon")).toBeDefined();
    expect(screen.getByText("Launch a real session")).toBeDefined();
  });

  it("wires tab roles, panels, and keyboard navigation", () => {
    render(<InstallSection />);

    const goInstall = screen.getByRole("tab", { name: "go install" });
    const source = screen.getByRole("tab", { name: "Build from source" });

    expect(goInstall.getAttribute("id")).toBe("install-tab-go");
    expect(goInstall.getAttribute("aria-controls")).toBe("install-panel-go");
    expect(goInstall.getAttribute("tabindex")).toBe("0");
    expect(source.getAttribute("tabindex")).toBe("-1");

    fireEvent.keyDown(goInstall, { key: "ArrowRight" });

    expect(source.getAttribute("aria-selected")).toBe("true");
    let panel = screen.getByRole("tabpanel");
    expect(panel.getAttribute("id")).toBe("install-panel-source");
    expect(panel.getAttribute("aria-labelledby")).toBe("install-tab-source");

    fireEvent.keyDown(source, { key: "Home" });

    expect(goInstall.getAttribute("aria-selected")).toBe("true");
    panel = screen.getByRole("tabpanel");
    expect(panel.getAttribute("id")).toBe("install-panel-go");
  });
});

describe("Comparison", () => {
  it("renders the four approaches and the Agents today column", () => {
    render(<Comparison />);
    expect(screen.getByText("Other tools stop at the runtime boundary.")).toBeDefined();
    for (const name of [
      "Assistant gateway",
      "All-in-one agent OS",
      "Multi-tenant gateway",
      "AGH",
    ]) {
      expect(screen.getByText(name)).toBeDefined();
    }
    expect(screen.getByText("8 ACP CLIs")).toBeDefined();
  });
});

describe("FinalCta", () => {
  it("renders the final CTAs and drops the old hedge copy", () => {
    render(<FinalCta />);
    expect(screen.getByText("Install AGH. Run a session. Join the network.")).toBeDefined();
    const install = screen.getByText("Install AGH");
    expect(install.closest("a")?.getAttribute("href")).toBe(
      "/runtime/core/getting-started/installation"
    );
    const spec = screen.getByText("Read agh-network/v0 spec");
    expect(spec.closest("a")?.getAttribute("href")).toBe("/protocol");
    const star = screen.getByText("Star on GitHub");
    expect(star.closest("a")?.getAttribute("href")).toBe(baseOptions.githubUrl);
  });
});

describe("KindChip", () => {
  it("has a meaning string for every NetworkKind", () => {
    const kinds: NetworkKind[] = [
      "greet",
      "whois",
      "say",
      "direct",
      "capability",
      "receipt",
      "trace",
    ];
    for (const kind of kinds) {
      expect(KIND_MEANING[kind]).toBeDefined();
      render(<KindChip kind={kind} />);
      expect(screen.getAllByText(kind)).toBeDefined();
    }
  });
});
