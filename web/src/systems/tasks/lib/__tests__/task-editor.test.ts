import { describe, expect, it } from "vitest";

import {
  buildCreateChildTaskRequest,
  buildCreateTaskRequest,
  EMPTY_TASK_EDITOR_DRAFT,
} from "../task-editor";

describe("buildCreateTaskRequest", () => {
  it("builds the root-task payload without a parent task id", () => {
    const payload = buildCreateTaskRequest(
      {
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Standalone task",
        workspaceId: "ws_signalforge",
      },
      {
        asDraft: false,
        templateId: "one_shot",
      }
    );

    expect(payload.workspace).toBe("ws_signalforge");
    expect(payload.network_channel).toBeUndefined();
    expect(payload.identifier).toBeUndefined();
    expect("parent_task_id" in payload).toBe(false);
  });
});

describe("buildCreateChildTaskRequest", () => {
  it("builds the child-task payload when the editor draft provides a parent task id", () => {
    const payload = buildCreateChildTaskRequest(
      {
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Web child creation probe 0425",
        description: "Probe task",
        scope: "workspace",
        workspaceId: "ws_signalforge",
        parentTaskId: " task-44a84096bb3e51ea ",
        networkChannel: " launch-sprint-0425 ",
        identifier: " WEB-CHILD-0425 ",
      },
      {
        asDraft: false,
        templateId: "one_shot",
      }
    );

    expect(payload.workspace).toBe("ws_signalforge");
    expect(payload.network_channel).toBe("launch-sprint-0425");
    expect(payload.identifier).toBe("WEB-CHILD-0425");
    expect("parent_task_id" in payload).toBe(false);
  });
});
