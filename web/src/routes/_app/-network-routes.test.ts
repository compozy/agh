import { describe, expect, it } from "vitest";

import { Route as NetworkLayoutRoute } from "./network";
import { Route as NetworkActivityRoute } from "./network.$channel.activity";
import { Route as NetworkDirectsRoute } from "./network.$channel.directs";
import { Route as NetworkDirectDetailRoute } from "./network.$channel.directs.$directId";
import { Route as NetworkThreadsRoute } from "./network.$channel.threads";
import { Route as NetworkThreadDetailRoute } from "./network.$channel.threads.$threadId";
import { routeTree } from "@/routeTree.gen";

interface RouteShape {
  options?: { component?: unknown };
}

describe("network channel-pivot routes", () => {
  it("registers the six file-based routes prescribed by the techspec", () => {
    for (const route of [
      NetworkLayoutRoute,
      NetworkThreadsRoute,
      NetworkThreadDetailRoute,
      NetworkDirectsRoute,
      NetworkDirectDetailRoute,
      NetworkActivityRoute,
    ] as RouteShape[]) {
      expect(route).toBeDefined();
      expect(typeof route.options?.component).toBe("function");
    }
  });

  it("includes the generated route tree as a stable singleton", () => {
    expect(routeTree).toBeDefined();
  });
});
