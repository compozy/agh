import { describe, expect, it } from "vitest";

import {
  bridgeDetailOptions,
  bridgeProvidersOptions,
  bridgeRoutesOptions,
  bridgesListOptions,
} from "@/systems/bridges/lib/query-options";

describe("bridgesListOptions", () => {
  it("uses the expected timings and list query key", () => {
    const options = bridgesListOptions();

    expect(options.queryKey).toEqual(["bridges", "list", "all"]);
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
