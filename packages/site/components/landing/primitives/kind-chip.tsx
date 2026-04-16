import { cn } from "@agh/ui/utils";

export type NetworkKind = "greet" | "whois" | "say" | "direct" | "recipe" | "receipt" | "trace";

/** One-line purpose for every kind — tooltip copy, alt text, and copy audit source. */
export const KIND_MEANING = {
  greet: "Announce presence + capabilities to a channel",
  whois: "Ask the network which peers match a capability",
  say: "Free-form operator chat to a channel",
  direct: "Send a structured task to a named peer",
  recipe: "Bundle multi-step delegation across peers",
  receipt: "Confirm completion with status and trace IDs",
  trace: "Stream progress updates during a task",
} as const satisfies Record<NetworkKind, string>;

interface KindChipProps {
  kind: NetworkKind;
  className?: string;
  /** Force a visual "active" / highlighted state. */
  active?: boolean;
}

export function KindChip({ kind, className, active = false }: KindChipProps) {
  return (
    <span
      title={KIND_MEANING[kind]}
      className={cn(
        "inline-flex items-center rounded-[5px] px-2 py-[3px] font-mono text-[10.5px] font-semibold uppercase tracking-(--tracking-mono)",
        active
          ? "bg-(--color-accent) text-white"
          : "bg-(--color-accent-tint) text-(--color-accent)",
        className
      )}
    >
      {kind}
    </span>
  );
}
