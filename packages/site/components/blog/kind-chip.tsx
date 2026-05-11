import { cn } from "@agh/ui";

export const WIRE_KINDS = ["greet", "whois", "say", "capability", "receipt", "trace"] as const;

export type WireKind = (typeof WIRE_KINDS)[number];

const dotClass: Record<WireKind, string> = {
  greet: "bg-(--color-kind-greet)",
  whois: "bg-(--color-kind-whois)",
  say: "bg-(--color-kind-say)",
  capability: "bg-(--color-kind-capability)",
  receipt: "bg-(--color-kind-receipt)",
  trace: "bg-(--color-kind-trace)",
};

export interface KindChipProps {
  kind: WireKind;
  label?: string;
  className?: string;
}

export function KindChip({ kind, label, className }: KindChipProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-chip border border-(--line) px-1.5 py-px eyebrow font-semibold! text-(--subtle)",
        className
      )}
    >
      <span className={cn("inline-block h-[7px] w-[7px] rounded-full", dotClass[kind])} />
      {label ?? kind}
    </span>
  );
}
