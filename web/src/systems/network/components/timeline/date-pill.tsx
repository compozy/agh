import { formatDatePill } from "../../lib/format-timestamp";

export interface DatePillProps {
  timestamp: string;
  /** Optional reference moment, primarily for tests. */
  now?: Date;
}

export function DatePill({ timestamp, now }: DatePillProps) {
  const label = formatDatePill(timestamp, { now });
  if (!label) {
    return null;
  }

  return (
    <div
      className="my-6 flex items-center gap-3 px-5"
      data-testid="network-timeline-date-pill"
      data-label={label}
      role="separator"
    >
      <span aria-hidden="true" className="h-px flex-1 bg-(--color-divider)" />
      <span className="font-mono text-eyebrow uppercase tracking-mono text-(--color-text-tertiary)">
        {label}
      </span>
      <span aria-hidden="true" className="h-px flex-1 bg-(--color-divider)" />
    </div>
  );
}
