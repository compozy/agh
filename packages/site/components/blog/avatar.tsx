import { cn } from "@agh/ui";

export interface AvatarProps {
  initial: string;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClass: Record<NonNullable<AvatarProps["size"]>, string> = {
  sm: "size-7 text-xs",
  md: "size-9 text-sm",
  lg: "size-11 text-base",
};

export function Avatar({ initial, size = "sm", className }: AvatarProps) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full bg-(--color-surface-elevated) font-sans font-semibold text-(--color-text-primary)",
        sizeClass[size],
        className
      )}
    >
      {initial}
    </span>
  );
}
