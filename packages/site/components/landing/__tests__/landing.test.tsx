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
import { BentoSection } from "../bento-section";
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
  it("renders six illustrated runtime capabilities", () => {
    render(<FeaturesSection />);
    const eyebrows = ["Sessions", "Memory", "Skills", "Workspaces", "Automation", "Observability"];
    for (const label of eyebrows) {
      expect(screen.getByText(label)).toBeDefined();
    }

    expect(screen.getAllByTestId("feature-card")).toHaveLength(6);
    expect(screen.queryByText("Hooks")).toBeNull();
    expect(screen.queryByText("Bridges")).toBeNull();
    expect(screen.getByText("Everything a modern agent runtime should have.")).toBeDefined();
  });

  it("uses the six everything illustration assets", () => {
    render(<FeaturesSection />);

    const expectedSources = [
      "/images/everything/illustration_01.png",
      "/images/everything/illustration_02.png",
      "/images/everything/illustration_04.png",
      "/images/everything/illustration_05.png",
      "/images/everything/illustration_06.png",
      "/images/everything/illustration_03.png",
    ];

    const sources = screen.getAllByRole("img").map(image => image.getAttribute("src"));

    for (const source of expectedSources) {
      expect(sources).toContain(source);
    }
  });
});

describe("BentoSection", () => {
  it("renders the five-tile runtime bento without an extra section header", () => {
    render(<BentoSection />);

    expect(screen.getByTestId("bento-grid")).toBeDefined();
    expect(screen.queryByText("The runtime surface in five parts.")).toBeNull();

    for (const label of ["Runtime", "Network", "Bridges", "Memory", "Trace"]) {
      expect(screen.getByText(label)).toBeDefined();
    }

    for (const title of [
      "Your agents. Under control.",
      "Built-in network. Delegate. Deliver. Done.",
      "From anywhere. Into a session.",
      "Context that remembers.",
      "Every step. Always replayable.",
    ]) {
      expect(screen.getByRole("heading", { name: title })).toBeDefined();
    }

    expect(screen.getByText("One local daemon. Every session. Every event.")).toBeDefined();
  });

  it("uses the five exported bento illustration assets", () => {
    render(<BentoSection />);

    const expectedSources = [
      "/images/bento-illustrations/runtime-v2.png",
      "/images/bento-illustrations/network-v2.png",
      "/images/bento-illustrations/bridges-v2.png",
      "/images/bento-illustrations/memory-v2.png",
      "/images/bento-illustrations/trace-v2.png",
    ];

    const sources = screen.getAllByRole("img").map(image => image.getAttribute("src"));

    for (const source of expectedSources) {
      expect(sources).toContain(source);
    }
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

  it("uses the runtime daemon illustration in the sticky rail", () => {
    render(<RuntimeSection />);

    expect(
      screen
        .getByAltText(
          "AGH daemon connecting CLI, API, and web UI surfaces to sessions, memory, skills, workspaces, and observability."
        )
        .getAttribute("src")
    ).toBe("/images/runtime/illustration_1.png");
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

  it("uses the dedicated skill contract illustration for the lower section", () => {
    render(<ExtensibilitySection />);

    expect(
      screen
        .getByAltText(
          "deploy-staging.skill.md shown as a Markdown skill contract with frontmatter, deployment capabilities, and a staged execution trace."
        )
        .getAttribute("src")
    ).toBe("/images/extensibility-skill-contract-v1.png");
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
