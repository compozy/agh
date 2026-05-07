import { describe, expect, it } from "vitest";

describe("tasks Storybook route stories", () => {
  it("loads the tasks route story modules and the tasks mock barrel", async () => {
    const [
      tasksStories,
      taskDetailStories,
      taskNewStories,
      taskEditStories,
      taskRunStories,
      tasksMocks,
    ] = await Promise.all([
      import("@/routes/_app/stories/-tasks.stories"),
      import("@/routes/_app/stories/-tasks.$id.stories"),
      import("@/routes/_app/stories/-tasks.new.stories"),
      import("@/routes/_app/stories/-tasks.$id.edit.stories"),
      import("@/routes/_app/stories/-tasks.$id.runs.$runId.stories"),
      import("@/systems/tasks/mocks"),
    ]);

    expect(tasksStories.default).toBeDefined();
    expect(taskDetailStories.default).toBeDefined();
    expect(taskNewStories.default).toBeDefined();
    expect(taskEditStories.default).toBeDefined();
    expect(taskRunStories.default).toBeDefined();

    expect(tasksMocks).toMatchObject({
      TASK_FIXTURES: expect.any(Array),
      handlers: expect.any(Array),
      taskDetailFixture: expect.any(Object),
    });
  });
});
