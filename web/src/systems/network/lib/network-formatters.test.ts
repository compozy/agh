import { describe, expect, it } from "vitest";

import type { NetworkCapabilityBrief, NetworkCapabilityCatalog } from "../types";
import {
  buildPeerCapabilityViews,
  formatNetworkKindLabel,
  hasCapabilityDetail,
} from "./network-formatters";

describe("buildPeerCapabilityViews", () => {
  it("Should merge brief entries with their catalog counterparts by id", () => {
    const brief: NetworkCapabilityBrief[] = [
      { id: "chat", summary: "Brief summary" },
      { id: "tools", summary: "Runs tools" },
    ];
    const catalog: NetworkCapabilityCatalog = {
      capabilities: [
        {
          id: "chat",
          summary: "Rich summary",
          outcome: "Peers align.",
          requirements: ["introspection"],
        },
      ],
    };

    const views = buildPeerCapabilityViews(brief, catalog);

    expect(views).toHaveLength(2);
    expect(views[0]?.id).toBe("chat");
    expect(views[0]?.summary).toBe("Brief summary");
    expect(views[0]?.detail?.outcome).toBe("Peers align.");
    expect(views[1]?.id).toBe("tools");
    expect(views[1]?.detail).toBeNull();
  });

  it("Should surface catalog-only capabilities that lack a brief entry", () => {
    const catalog: NetworkCapabilityCatalog = {
      capabilities: [{ id: "orphan", summary: "Only rich.", outcome: "No brief entry." }],
    };

    const views = buildPeerCapabilityViews(undefined, catalog);

    expect(views).toHaveLength(1);
    expect(views[0]?.id).toBe("orphan");
    expect(views[0]?.summary).toBe("Only rich.");
    expect(views[0]?.detail?.outcome).toBe("No brief entry.");
  });

  it("Should return an empty list when neither brief nor catalog has entries", () => {
    expect(buildPeerCapabilityViews(undefined, undefined)).toEqual([]);
    expect(buildPeerCapabilityViews([], { capabilities: [] })).toEqual([]);
  });
});

describe("hasCapabilityDetail", () => {
  it("Should report false when no catalog detail is attached", () => {
    expect(hasCapabilityDetail({ id: "chat", summary: "Brief only", detail: null })).toBe(false);
  });

  it("Should report true when any rich field carries content", () => {
    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: {
          id: "chat",
          summary: "Brief",
          outcome: "Peers align.",
        },
      })
    ).toBe(true);

    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: {
          id: "chat",
          summary: "Brief",
          outcome: "",
          requirements: ["introspection"],
        },
      })
    ).toBe(true);
  });

  it("Should report false for an empty catalog detail stub", () => {
    expect(
      hasCapabilityDetail({
        id: "chat",
        summary: "Brief",
        detail: { id: "chat", summary: "Brief", outcome: "" },
      })
    ).toBe(false);
  });
});

describe("formatNetworkKindLabel", () => {
  it.each(["capability", "say", "direct", "trace", "receipt", "greet", "whois"] as const)(
    "Should keep the %s timeline event aligned with the API kind name",
    kind => {
      expect(formatNetworkKindLabel(kind)).toBe(kind);
    }
  );

  it("Should preserve unknown kind strings as-is", () => {
    expect(formatNetworkKindLabel("custom-signal")).toBe("custom-signal");
  });
});
