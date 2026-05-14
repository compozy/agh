import { describe, expect, it } from "vitest";

import { Route as ThreadDetailRoute } from "../network.$workspaceId.$channel.threads.$threadId";
import { Route as DirectDetailRoute } from "../network.$workspaceId.$channel.directs.$directId";
import { Route as ActivityRoute } from "../network.$workspaceId.$channel.activity";

interface RouteShape {
  options?: { component?: unknown; validateSearch?: (search: Record<string, unknown>) => unknown };
}

describe("network task_14 route components", () => {
  it("Should bind thread detail route to a component", () => {
    const route = ThreadDetailRoute as RouteShape;
    expect(typeof route.options?.component).toBe("function");
  });

  it("Should bind direct detail route to a component", () => {
    const route = DirectDetailRoute as RouteShape;
    expect(typeof route.options?.component).toBe("function");
  });

  it("Should bind activity tab route to a component", () => {
    const route = ActivityRoute as RouteShape;
    expect(typeof route.options?.component).toBe("function");
  });

  it("Should validate `view=full` search param on the thread detail route", () => {
    const route = ThreadDetailRoute as RouteShape;
    const validate = route.options?.validateSearch;
    expect(typeof validate).toBe("function");
    if (typeof validate === "function") {
      expect(validate({ view: "full" })).toEqual({ view: "full" });
      expect(validate({ view: "anything-else" })).toEqual({ view: undefined });
      expect(validate({})).toEqual({ view: undefined });
    }
  });
});
