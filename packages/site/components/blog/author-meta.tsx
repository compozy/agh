import { cn } from "@agh/ui";
import { Avatar } from "./avatar";

export interface AuthorMetaProps {
  handle: string;
  initial: string;
  role?: string;
  size?: "sm" | "md" | "lg";
  layout?: "row" | "stacked";
  className?: string;
}

export function AuthorMeta({
  handle,
  initial,
  role,
  size = "sm",
  layout = "row",
  className,
}: AuthorMetaProps) {
  if (layout === "stacked") {
    return (
      <div className={cn("flex items-center gap-3", className)}>
        <Avatar initial={initial} size={size} />
        <div>
          <p className="font-sans text-[14px] font-medium text-(--color-text-primary)">{handle}</p>
          {role && (
            <p className="font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-label)">
              {role}
            </p>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("inline-flex items-center gap-2.5", className)}>
      <Avatar initial={initial} size={size} />
      <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-(--color-text-label)">
        {handle}
      </span>
    </div>
  );
}
