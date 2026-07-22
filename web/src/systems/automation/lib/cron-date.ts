import { DAY_MS, DOW_LONG, DOW_SHORT, MONTH_SHORT, formatClock, pad2 } from "./cron-engine";

/**
 * Convert a timezone-naive `datetime-local` value (whose wall-clock we treat as
 * UTC, matching the scheduler) into a `Date`, or `null` when unparseable.
 */
export function localInputToDate(value: string): Date | null {
  const match = String(value).match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/);
  if (!match) {
    return null;
  }
  return new Date(
    Date.UTC(
      Number(match[1]),
      Number(match[2]) - 1,
      Number(match[3]),
      Number(match[4]),
      Number(match[5]),
      0
    )
  );
}

/** Render a `Date` as the timezone-naive `datetime-local` string in UTC. */
export function dateToLocalInput(date: Date): string {
  return (
    `${date.getUTCFullYear()}-${pad2(date.getUTCMonth() + 1)}-${pad2(date.getUTCDate())}` +
    `T${pad2(date.getUTCHours())}:${pad2(date.getUTCMinutes())}`
  );
}

/** RFC3339 / ISO-8601 UTC string without milliseconds (`…:00Z`). */
export function toRfc3339(date: Date | null): string {
  return date ? date.toISOString().replace(/\.\d{3}Z$/, "Z") : "";
}

/** Default `at` value: next round hour, one day out, as a `datetime-local` string. */
export function defaultAtLocal(now: number = Date.now()): string {
  const date = new Date(now + DAY_MS);
  date.setUTCMinutes(0, 0, 0);
  return dateToLocalInput(date);
}

/** Absolute UTC label, e.g. `Mon Jun 3, 09:00`. */
export function formatAbsoluteUtc(date: Date): string {
  return (
    `${DOW_SHORT[date.getUTCDay()]} ${MONTH_SHORT[date.getUTCMonth()]} ${date.getUTCDate()}, ` +
    formatClock(date.getUTCHours(), date.getUTCMinutes())
  );
}

/** Relative label, e.g. `in 2h 30m`, `now`. */
export function formatRelative(date: Date, now: number = Date.now()): string {
  const seconds = Math.round((date.getTime() - now) / 1000);
  if (seconds < 0) {
    return "now";
  }
  if (seconds < 60) {
    return `in ${seconds}s`;
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `in ${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  if (hours < 24) {
    return `in ${hours}h${remMinutes ? ` ${remMinutes}m` : ""}`;
  }
  const days = Math.floor(hours / 24);
  const remHours = hours % 24;
  return `in ${days}d${remHours ? ` ${remHours}h` : ""}`;
}

export const SCHEDULE_CONSTANTS = {
  DOW_LONG,
  DOW_SHORT,
  MONTH_SHORT,
} as const;
