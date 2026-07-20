import { describe, expect, it } from "vitest";

import { parseTaskWindowLocation } from "../task-window-location";

describe("parseTaskWindowLocation", () => {
  it("Should fall back to the list mode when the tasks search mode is invalid", () => {
    expect(parseTaskWindowLocation({ pathname: "/tasks", search: { mode: "bogus" } })).toEqual({
      kind: "catalog",
      mode: "list",
    });
  });
});
