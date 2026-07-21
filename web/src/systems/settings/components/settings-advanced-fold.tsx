import { ChevronRight } from "lucide-react";
import { useState, type ReactNode } from "react";

import { cn } from "@agh/ui";

/**
 * The one Advanced fold per page (design system §08): chevron toggle, closed
 * by default, hosting the operator layer. Provenance chips are allowed only
 * inside this fold.
 */
export function SettingsAdvancedFold({
  children,
  defaultOpen = false,
  padded = false,
  "data-testid": testId,
}: {
  children: ReactNode;
  defaultOpen?: boolean;
  /** Pad the body when it hosts whole groups instead of flush rows. */
  padded?: boolean;
  "data-testid"?: string;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const bodyId = "settings-advanced-body";

  return (
    <section
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid={testId ?? "settings-advanced"}
    >
      <button
        aria-controls={bodyId}
        aria-expanded={open}
        className={cn(
          "flex w-full items-center gap-2 px-4 py-3 text-left text-ws-name font-medium text-muted",
          "transition-colors duration-base hover:bg-row-hover hover:text-fg",
          "focus-visible:outline-none focus-visible:shadow-focus-ring"
        )}
        data-testid="settings-advanced-toggle"
        onClick={() => setOpen(current => !current)}
        type="button"
      >
        <ChevronRight
          aria-hidden="true"
          className={cn(
            "size-3.5 text-faint transition-transform duration-base",
            open && "rotate-90"
          )}
        />
        Advanced
      </button>
      {open ? (
        <div
          className={cn("border-t border-line-soft", padded && "flex flex-col gap-6 p-4")}
          id={bodyId}
        >
          {children}
        </div>
      ) : null}
    </section>
  );
}

/** Mono provenance chip naming the real config key — Advanced-only. */
export function SettingsProvChip({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex h-4 items-center rounded-xxs bg-badge-fill px-1.5 font-mono text-micro font-medium text-subtle">
      {children}
    </span>
  );
}
