import { describe, expect, it } from "vitest";

import { bridgeKeys } from "../query-keys";

describe("bridgeKeys", () => {
  it("creates stable list and providers keys", () => {
    expect(bridgeKeys.list()).toEqual(["bridges", "list", "all"]);
    expect(bridgeKeys.providers()).toEqual(["bridges", "providers"]);
  });

  it("normalizes omitted detail and routes ids to empty strings", () => {
    expect(bridgeKeys.detail("")).toEqual(["bridges", "detail", ""]);
    expect(bridgeKeys.routes("")).toEqual(["bridges", "routes", ""]);
    expect(bridgeKeys.secretBindings("")).toEqual(["bridges", "secret-bindings", ""]);
  });

  it("includes bridge ids in detail and route query keys", () => {
    expect(bridgeKeys.detail("brg_support")).toEqual(["bridges", "detail", "brg_support"]);
    expect(bridgeKeys.routes("brg_support")).toEqual(["bridges", "routes", "brg_support"]);
    expect(bridgeKeys.secretBindings("brg_support")).toEqual([
      "bridges",
      "secret-bindings",
      "brg_support",
    ]);
  });
});
