import { describe, expect, it } from "vitest";

import {
  FORMAT_TIME_FALLBACK,
  formatAbsoluteTime,
  formatDuration,
  formatRelativeTime,
} from "../format-time";

const NOW = new Date("2026-05-11T12:00:00Z").getTime();

describe("format-time re-export", () => {
  it("Should re-export the @agh/ui sentinel string", () => {
    expect(FORMAT_TIME_FALLBACK).toBe("—");
  });

  it("Should render relative seconds/minutes for past deltas", () => {
    expect(formatRelativeTime(new Date(NOW - 45_000).toISOString(), NOW)).toBe("45s ago");
    expect(formatRelativeTime(new Date(NOW - 5 * 60_000).toISOString(), NOW)).toBe("5m ago");
  });

  it("Should render 'just now' for sub-30s deltas", () => {
    expect(formatRelativeTime(new Date(NOW - 1_000).toISOString(), NOW)).toBe("just now");
  });

  it("Should render 'in N' for future deltas", () => {
    expect(formatRelativeTime(new Date(NOW + 90_000).toISOString(), NOW)).toBe("in 1m");
  });

  it("Should return the sentinel for invalid ISO inputs", () => {
    expect(formatRelativeTime("not-a-date", NOW)).toBe(FORMAT_TIME_FALLBACK);
    expect(formatAbsoluteTime("invalid")).toBe(FORMAT_TIME_FALLBACK);
  });

  it("Should render an absolute timestamp containing the year", () => {
    expect(formatAbsoluteTime(new Date(NOW).toISOString())).toMatch(/2026/);
  });

  it("Should compose hours + minutes + seconds for durations", () => {
    expect(formatDuration(125_000)).toBe("2m 5s");
    expect(formatDuration(3_725_000)).toBe("1h 2m 5s");
    expect(formatDuration(0)).toBe("0s");
    expect(formatDuration(-1)).toBe("0s");
  });
});
