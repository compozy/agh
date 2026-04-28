import { Pill } from "@agh/ui";
import * as React from "react";

import { kindColorFor } from "@/lib/kind-colors";

export interface KindChipProps extends Omit<React.ComponentProps<"span">, "children"> {
  kind: string;
  /** Optional explicit label; defaults to `kind`. */
  label?: React.ReactNode;
}

/**
 * Protocol kind marker — transparent surface, neutral border + tertiary
 * label, leading 7px colored dot keyed off the protocol kind. Unknown kinds
 * (platform names, event ids) render without a dot. Composes `Pill` + `Pill.Dot`.
 */
export function KindChip({ kind, label, className, ...props }: KindChipProps) {
  const dotColor = kindColorFor(kind);

  return (
    <Pill
      mono
      size="xs"
      tone="neutral"
      data-slot="kind-chip"
      data-kind={kind}
      uppercase
      className={className}
      {...props}
    >
      {dotColor ? <Pill.Dot color={dotColor} /> : null}
      <span>{label ?? kind}</span>
    </Pill>
  );
}
