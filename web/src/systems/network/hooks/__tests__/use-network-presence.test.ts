// @vitest-environment jsdom

import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useNetworkPresence } from "../use-network-presence";

describe("useNetworkPresence (placeholder)", () => {
  it("Should return idle by default until protocol presence ships", () => {
    const { result } = renderHook(() => useNetworkPresence({ channel: "ops", peerId: "p1" }));
    expect(result.current.state).toBe("idle");
  });

  it("Should return idle when called with no args", () => {
    const { result } = renderHook(() => useNetworkPresence());
    expect(result.current.state).toBe("idle");
  });
});
