import { cn } from "@agh/ui";
import Link from "next/link";

export interface CategoryPillProps {
  label: string;
  count?: number;
  href: string;
  active?: boolean;
}

export function CategoryPill({ label, count, href, active = false }: CategoryPillProps) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex h-8 items-center gap-2 rounded-full border px-3.5 font-sans text-[13px] font-medium transition-colors",
        active
          ? "border-(--color-accent) bg-(--color-accent-tint) text-(--color-accent)"
          : "border-(--color-divider) text-(--color-text-secondary) hover:text-(--color-text-primary)"
      )}
    >
      <span>{label}</span>
      {count !== undefined && (
        <span
          className={cn(
            "font-mono text-[10px] tracking-[0.06em]",
            active ? "text-(--color-accent)" : "text-(--color-text-tertiary)"
          )}
        >
          {String(count).padStart(2, "0")}
        </span>
      )}
    </Link>
  );
}
