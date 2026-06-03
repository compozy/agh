/**
 * Pure schedule math for the automation Job create/preview flow.
 *
 * The runtime scheduler (internal/automation) evaluates a standard 5-field cron
 * in UTC — no seconds field and no `@daily`-style macros. This module mirrors
 * those semantics so the in-form readout and the live "next runs" preview match
 * what the daemon will actually do. Everything here is a faithful derivation of
 * the saved expression/interval/time, never a fabricated metric.
 *
 * All `now` parameters default to `Date.now()` but are injectable so unit tests
 * stay deterministic.
 */

const DOW_LONG = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
] as const;
const DOW_SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"] as const;
const MONTH_SHORT = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
] as const;

const MINUTE_MS = 60_000;
const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;
/** Upper bound on the minute-by-minute scan for the next fire (≈ 367 days). */
const CRON_SCAN_CAP = 367 * 24 * 60;

export type CronFrequency = "minutes" | "hourly" | "daily" | "weekly" | "monthly" | "custom";

/**
 * The friendly builder model layered on top of the raw 5-field string. The raw
 * expression remains the source of truth; this model only drives the visual
 * controls. `decodeCron` reverses a string into this model; `compileCron`
 * projects the model back to a string for every frequency except `custom`.
 */
export interface CronModel {
  frequency: CronFrequency;
  /** Minute interval N for the every-N-minutes form (`star-slash-N * * * *`). */
  everyMinutes: number;
  /** Minute-of-hour for the `hourly` frequency. */
  hourlyMinute: number;
  /** Hour-of-day (0–23) for daily/weekly/monthly. */
  hour: number;
  /** Minute-of-hour (0–59) for daily/weekly/monthly. */
  minute: number;
  /** Selected weekdays (0=Sun … 6=Sat) for the `weekly` frequency. */
  weekdays: number[];
  /** Day-of-month (1–31) for the `monthly` frequency. */
  monthDay: number;
}

interface CronField {
  any: boolean;
  set: Set<number>;
}

interface ParsedCron {
  minute: CronField;
  hour: CronField;
  dayOfMonth: CronField;
  month: CronField;
  dayOfWeek: CronField;
}

const DEFAULT_CRON_MODEL: CronModel = {
  frequency: "daily",
  everyMinutes: 15,
  hourlyMinute: 0,
  hour: 9,
  minute: 0,
  weekdays: [1, 2, 3, 4, 5],
  monthDay: 1,
};

export function fullCronModel(partial?: Partial<CronModel>): CronModel {
  return {
    ...DEFAULT_CRON_MODEL,
    ...partial,
    weekdays: partial?.weekdays ? [...partial.weekdays] : [...DEFAULT_CRON_MODEL.weekdays],
  };
}

export function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

/** "HH:MM" wall-clock string (UTC). */
export function formatClock(hour: number, minute: number): string {
  return `${pad2(hour)}:${pad2(minute)}`;
}

function parseCronField(raw: string, lo: number, hi: number): CronField {
  if (raw === "*") {
    return { any: true, set: new Set() };
  }
  const set = new Set<number>();
  for (const part of raw.split(",")) {
    let rangeText = part;
    let step = 1;
    const slash = part.indexOf("/");
    if (slash >= 0) {
      rangeText = part.slice(0, slash);
      step = Number.parseInt(part.slice(slash + 1), 10);
      if (!Number.isInteger(step) || step < 1) {
        throw new Error("invalid cron step");
      }
    }
    let start: number;
    let end: number;
    if (rangeText === "*") {
      start = lo;
      end = hi;
    } else if (rangeText.indexOf("-") >= 0) {
      const [a, b] = rangeText.split("-");
      start = Number.parseInt(a, 10);
      end = Number.parseInt(b, 10);
    } else {
      start = Number.parseInt(rangeText, 10);
      end = start;
    }
    if (Number.isNaN(start) || Number.isNaN(end) || start < lo || end > hi || start > end) {
      throw new Error("cron field out of range");
    }
    for (let value = start; value <= end; value += step) {
      set.add(value);
    }
  }
  if (set.size === 0) {
    throw new Error("empty cron field");
  }
  return { any: false, set };
}

/** Parse a 5-field cron into matchable fields, or `null` when malformed. */
export function parseCron(expr: string): ParsedCron | null {
  const parts = String(expr).trim().split(/\s+/);
  if (parts.length !== 5) {
    return null;
  }
  try {
    return {
      minute: parseCronField(parts[0], 0, 59),
      hour: parseCronField(parts[1], 0, 23),
      dayOfMonth: parseCronField(parts[2], 1, 31),
      month: parseCronField(parts[3], 1, 12),
      dayOfWeek: parseCronField(parts[4], 0, 6),
    };
  } catch {
    return null;
  }
}

export function isValidCron(expr: string): boolean {
  return parseCron(expr) !== null;
}

function matches(field: CronField, value: number): boolean {
  return field.any || field.set.has(value);
}

/**
 * Compute up to `count` upcoming fire times (UTC) for a cron expression.
 * Returns `null` when the expression is invalid, `[]` when nothing fires within
 * the scan horizon. Day-of-month and day-of-week combine with OR semantics when
 * both are restricted, matching Vixie/robfig cron.
 */
export function cronNext(expr: string, count: number, now: number = Date.now()): Date[] | null {
  const parsed = parseCron(expr);
  if (!parsed) {
    return null;
  }
  const out: Date[] = [];
  // Start at the top of the next minute.
  let cursor = new Date(now);
  cursor.setUTCSeconds(0, 0);
  cursor = new Date(cursor.getTime() + MINUTE_MS);

  for (let step = 0; step < CRON_SCAN_CAP && out.length < count; step += 1) {
    const minute = cursor.getUTCMinutes();
    const hour = cursor.getUTCHours();
    const dom = cursor.getUTCDate();
    const month = cursor.getUTCMonth() + 1;
    const dow = cursor.getUTCDay();

    const dayMatches =
      !parsed.dayOfMonth.any && !parsed.dayOfWeek.any
        ? parsed.dayOfMonth.set.has(dom) || parsed.dayOfWeek.set.has(dow)
        : matches(parsed.dayOfMonth, dom) && matches(parsed.dayOfWeek, dow);

    if (
      matches(parsed.minute, minute) &&
      matches(parsed.hour, hour) &&
      matches(parsed.month, month) &&
      dayMatches
    ) {
      out.push(new Date(cursor.getTime()));
    }
    cursor = new Date(cursor.getTime() + MINUTE_MS);
  }
  return out;
}

function singleValue(field: string): number | null {
  return /^\d+$/.test(field) ? Number.parseInt(field, 10) : null;
}

/** Reverse a cron string into the builder model, or `null` for custom shapes. */
export function decodeCron(expr: string): Partial<CronModel> | null {
  const parts = String(expr).trim().split(/\s+/);
  if (parts.length !== 5 || !parseCron(expr)) {
    return null;
  }
  const [minuteField, hourField, domField, monthField, dowField] = parts;

  const everyMatch = minuteField.match(/^\*\/(\d+)$/);
  if (
    everyMatch &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return { frequency: "minutes", everyMinutes: Number.parseInt(everyMatch[1], 10) };
  }

  const minute = singleValue(minuteField);
  const hour = singleValue(hourField);
  const monthDay = singleValue(domField);

  if (
    minute !== null &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return { frequency: "hourly", hourlyMinute: minute };
  }
  if (
    minute !== null &&
    hour !== null &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return { frequency: "daily", hour, minute };
  }
  if (
    minute !== null &&
    hour !== null &&
    domField === "*" &&
    monthField === "*" &&
    dowField !== "*"
  ) {
    try {
      const field = parseCronField(dowField, 0, 6);
      const days: number[] = [];
      for (let day = 0; day <= 6; day += 1) {
        if (field.set.has(day)) {
          days.push(day);
        }
      }
      if (days.length > 0) {
        return { frequency: "weekly", hour, minute, weekdays: days };
      }
    } catch {
      return null;
    }
  }
  if (
    minute !== null &&
    hour !== null &&
    monthDay !== null &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return { frequency: "monthly", hour, minute, monthDay };
  }
  return null;
}

/** Collapse a weekday list to the most compact cron day-of-week token. */
export function compressWeekdays(days: number[]): string {
  const sorted = [...new Set(days)].sort((a, b) => a - b);
  if (sorted.length >= 7) {
    return "*";
  }
  let contiguous = true;
  for (let i = 1; i < sorted.length; i += 1) {
    if (sorted[i] !== sorted[i - 1] + 1) {
      contiguous = false;
      break;
    }
  }
  if (contiguous && sorted.length >= 3) {
    return `${sorted[0]}-${sorted[sorted.length - 1]}`;
  }
  return sorted.join(",");
}

/**
 * Project the builder model to a 5-field cron string. Returns `null` for the
 * `custom` frequency, where the caller keeps the raw user-entered expression.
 */
export function compileCron(model: CronModel): string | null {
  switch (model.frequency) {
    case "minutes":
      return `*/${model.everyMinutes} * * * *`;
    case "hourly":
      return `${model.hourlyMinute} * * * *`;
    case "daily":
      return `${model.minute} ${model.hour} * * *`;
    case "weekly":
      return `${model.minute} ${model.hour} * * ${compressWeekdays(model.weekdays)}`;
    case "monthly":
      return `${model.minute} ${model.hour} ${model.monthDay} * *`;
    case "custom":
      return null;
  }
}

function weekdayPhrase(set: Set<number>): string {
  const days = [...set].sort((a, b) => a - b);
  if (days.length === 5 && days.join(",") === "1,2,3,4,5") {
    return "every weekday";
  }
  if (days.length === 2 && days.join(",") === "0,6") {
    return "weekends";
  }
  if (days.length === 1) {
    return `every ${DOW_LONG[days[0]]}`;
  }
  return `on ${days.map(day => DOW_SHORT[day]).join(", ")}`;
}

/** Plain-language rendering of a cron expression, or `null` for custom shapes. */
export function humanCron(expr: string): string | null {
  const parts = String(expr).trim().split(/\s+/);
  const parsed = parseCron(expr);
  if (!parsed) {
    return null;
  }
  const [minuteField, hourField, domField, monthField, dowField] = parts;
  const token = (field: string): number | null =>
    field.indexOf(",") < 0 && field.indexOf("-") < 0 && field.indexOf("/") < 0 && field !== "*"
      ? Number.parseInt(field, 10)
      : null;
  const minute = token(minuteField);
  const hour = token(hourField);

  const everyMatch = minuteField.match(/^\*\/(\d+)$/);
  if (
    everyMatch &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return `every ${everyMatch[1]} minutes`;
  }
  if (
    minuteField === "*" &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return "every minute";
  }
  if (
    minute !== null &&
    hourField === "*" &&
    domField === "*" &&
    monthField === "*" &&
    dowField === "*"
  ) {
    return minute === 0 ? "every hour, on the hour" : `every hour at :${pad2(minute)}`;
  }
  if (minute !== null && hour !== null && domField === "*" && monthField === "*") {
    const isMidnight = hour === 0 && minute === 0;
    const time = isMidnight ? "midnight" : `at ${formatClock(hour, minute)}`;
    if (dowField === "*") {
      return isMidnight ? "every day at midnight" : `every day ${time}`;
    }
    return `${weekdayPhrase(parsed.dayOfWeek.set)} ${isMidnight ? "at midnight" : time}`;
  }
  if (
    minute !== null &&
    hour !== null &&
    monthField === "*" &&
    dowField === "*" &&
    domField !== "*"
  ) {
    const day = token(domField);
    if (day !== null) {
      return `on day ${day} of every month at ${formatClock(hour, minute)}`;
    }
  }
  return null;
}

/** Parse a positive Go-style duration (`30m`, `1h`, `2h30m`, `45s`) to ms. */
export function parseDuration(value: string): number | null {
  const text = String(value).trim();
  if (!/^(\d+(h|m|s))+$/.test(text)) {
    return null;
  }
  let ms = 0;
  const re = /(\d+)(h|m|s)/g;
  let match: RegExpExecArray | null;
  while ((match = re.exec(text)) !== null) {
    const amount = Number.parseInt(match[1], 10);
    ms +=
      match[2] === "h" ? amount * HOUR_MS : match[2] === "m" ? amount * MINUTE_MS : amount * 1000;
  }
  return ms > 0 ? ms : null;
}

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
