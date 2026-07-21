import { describe, expect, it } from "vitest";

import { parseTaskWindowLocation } from "../task-window-location";

describe("parseTaskWindowLocation", () => {
  it("Should fall back to the list mode when the tasks search mode is invalid", () => {
    expect(parseTaskWindowLocation({ pathname: "/tasks", search: { mode: "bogus" } })).toEqual({
      kind: "catalog",
      mode: "list",
    });
  });

  it("Should resolve task detail tab and inspect defaults at the window boundary", () => {
    expect(
      parseTaskWindowLocation({
        pathname: "/tasks/task_1",
        search: { inspect: "stream", tab: "activity" },
      })
    ).toEqual({
      kind: "detail",
      taskId: "task_1",
      search: { inspect: "stream", tab: "activity" },
    });
  });
});
