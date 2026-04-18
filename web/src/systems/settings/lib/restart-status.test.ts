import { describe, expect, it } from "vitest";

import {
  isFailedRestart,
  isSuccessfulRestart,
  isTerminalRestartStatus,
  RESTART_TERMINAL_STATUSES,
} from "./restart-status";

describe("restart status helpers", () => {
  it("treats only ready and failed as terminal", () => {
    expect(RESTART_TERMINAL_STATUSES).toEqual(["ready", "failed"]);
    expect(isTerminalRestartStatus("ready")).toBe(true);
    expect(isTerminalRestartStatus("failed")).toBe(true);
    expect(isTerminalRestartStatus("pending")).toBe(false);
    expect(isTerminalRestartStatus("stopping")).toBe(false);
    expect(isTerminalRestartStatus(null)).toBe(false);
    expect(isTerminalRestartStatus(undefined)).toBe(false);
  });

  it("distinguishes success from failure", () => {
    expect(isSuccessfulRestart("ready")).toBe(true);
    expect(isSuccessfulRestart("failed")).toBe(false);
    expect(isFailedRestart("failed")).toBe(true);
    expect(isFailedRestart("ready")).toBe(false);
  });
});
