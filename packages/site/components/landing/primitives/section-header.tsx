import { cn } from "@agh/ui/lib/utils";
import type { ReactNode } from "react";

interface SectionHeaderProps {
  eyebrow?: string;
  title: ReactNode;
  description?: ReactNode;
  align?: "start" | "center";
  /** Larger display type for hero-style sections. */
  size?: "md" | "lg";
  className?: string;
}

export function SectionHeader({
  eyebrow,
  title,
  description,
  align = "start",
  size = "md",
  className,
}: SectionHeaderProps) {
  const alignClass = align === "center" ? "text-center mx-auto" : "text-left";
  const maxWidth = align === "center" ? "max-w-[750px]" : "max-w-[700px]";
  const titleClass =
    size === "lg"
      ? "text-site-hero-section leading-none font-normal tracking-tight"
      : "text-site-section-title leading-none font-normal tracking-tight";

  return (
    <div className={cn(maxWidth, alignClass, className)}>
      {eyebrow ? (
        <p className="font-mono text-eyebrow font-medium uppercase tracking-mono text-(--color-text-tertiary)">
          {eyebrow}
        </p>
      ) : null}
      <h2
        className={cn(
          "mt-5 text-(--color-text-primary)",
          titleClass,
          align === "center" && "mx-auto"
        )}
      >
        {title}
      </h2>
      {description ? (
        <p
          className={cn(
            "mt-5 text-base leading-relaxed text-(--color-text-secondary)",
            align === "center" ? "mx-auto max-w-[58ch]" : "max-w-[62ch]"
          )}
        >
          {description}
        </p>
      ) : null}
    </div>
  );
}
