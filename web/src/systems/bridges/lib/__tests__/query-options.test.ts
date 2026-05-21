import { describe, expect, it } from "vitest";

import {
  bridgeDetailOptions,
  bridgeProvidersOptions,
  bridgeRoutesOptions,
  bridgeSecretBindingsOptions,
  bridgeTargetsOptions,
  bridgesListOptions,
} from "@/systems/bridges/lib/query-options";

describe("bridgesListOptions", () => {
  it("uses the expected timings and list query key", () => {
    const options = bridgesListOptions({ scope: "all", workspace_id: "ws_alpha" });

    expect(options.queryKey).toEqual(["bridges", "list", "all", "ws_alpha", ""]);
    expect(options.staleTime).toBe(15_000);
    expect(options.refetchInterval).toBe(30_000);
  });
});

describe("bridgeProvidersOptions", () => {
  it("uses the providers key and slower refetch cadence", () => {
    const options = bridgeProvidersOptions();

    expect(options.queryKey).toEqual(["bridges", "providers"]);
    expect(options.refetchInterval).toBe(60_000);
  });
});

describe("bridgeDetailOptions", () => {
  it("is disabled when the bridge id is missing", () => {
    const options = bridgeDetailOptions("");

    expect(options.queryKey).toEqual(["bridges", "detail", ""]);
    expect(options.enabled).toBe(false);
  });

  it("is enabled for real bridge ids", () => {
    const options = bridgeDetailOptions("brg_support");

    expect(options.enabled).toBe(true);
  });
});

describe("bridgeRoutesOptions", () => {
  it("uses the expected routes key and is gated by id", () => {
    const options = bridgeRoutesOptions("brg_support");

    expect(options.queryKey).toEqual(["bridges", "routes", "brg_support"]);
    expect(options.enabled).toBe(true);
  });
});

describe("bridgeTargetsOptions", () => {
  it("uses the target directory key and is gated by id", () => {
    const options = bridgeTargetsOptions("brg_support", { limit: "50", q: "support" });

    expect(options.queryKey).toEqual(["bridges", "targets", "brg_support", "support", "50"]);
    expect(options.enabled).toBe(true);
    expect(options.refetchInterval).toBe(30_000);
  });
});

describe("bridgeSecretBindingsOptions", () => {
  it("uses the secret bindings key and is gated by id", () => {
    const options = bridgeSecretBindingsOptions("brg_support");

    expect(options.queryKey).toEqual(["bridges", "secret-bindings", "brg_support"]);
    expect(options.enabled).toBe(true);
    expect(options.refetchInterval).toBe(30_000);
  });
});
