import { cn } from "@agh/ui";

export interface AvatarProps {
  initial: string;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClass: Record<NonNullable<AvatarProps["size"]>, string> = {
  sm: "h-7 w-7 text-[12px]",
  md: "h-9 w-9 text-[14px]",
  lg: "h-11 w-11 text-[16px]",
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
