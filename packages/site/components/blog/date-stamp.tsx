import { cn } from "@agh/ui";
import type { ComponentProps } from "react";
import { formatDate, formatDateCompact } from "./format";
import type { MonoEyebrowTone } from "./mono-eyebrow";

export interface DateStampProps extends Omit<ComponentProps<"time">, "children" | "dateTime"> {
  date: string;
  format?: "default" | "compact" | "compact-year";
  tone?: MonoEyebrowTone;
  tracking?: "default" | "wide";
}

const toneClass: Record<MonoEyebrowTone, string> = {
  neutral: "text-(--color-text-label)",
  accent: "text-(--color-accent)",
  success: "text-(--color-success)",
  danger: "text-(--color-danger)",
  warning: "text-(--color-warning)",
  info: "text-(--color-info)",
};

function displayDate(date: string, format: NonNullable<DateStampProps["format"]>) {
  if (format === "compact") {
    return formatDateCompact(date);
  }
  if (format === "compact-year") {
    return `${formatDateCompact(date)} · ${new Date(date).getUTCFullYear()}`;
  }
  return formatDate(date);
}

export function DateStamp({
  date,
  format = "default",
  tone = "neutral",
  tracking = "default",
  className,
  ...props
}: DateStampProps) {
  return (
    <time
      dateTime={date}
      {...props}
      className={cn(
        "font-mono text-[11px] font-semibold uppercase",
        tracking === "wide" ? "tracking-[0.08em]" : "tracking-[0.06em]",
        toneClass[tone],
        className
      )}
    >
      {displayDate(date, format)}
    </time>
  );
}
