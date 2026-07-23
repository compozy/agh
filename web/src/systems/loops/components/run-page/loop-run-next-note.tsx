import { ArrowRight } from "lucide-react";

import { LoopRunSection } from "./loop-run-section";

interface LoopRunNextNoteProps {
  /** The graph-derived sentence from `buildNextNote`; null renders nothing. */
  note: string | null;
}

/**
 * The "What happens next" quiet note (§2/§5.3): one graph-derived sentence,
 * rendered only while the run is live and the pinned graph names what's ahead.
 */
export function LoopRunNextNote({ note }: LoopRunNextNoteProps) {
  if (!note) return null;
  return (
    <LoopRunSection label="What happens next" data-testid="loop-run-next">
      <div className="flex items-start gap-2.25 rounded-md border border-line-soft bg-canvas-soft px-3.5 py-3 text-small-body leading-relaxed text-muted">
        <ArrowRight aria-hidden="true" className="mt-0.5 size-3.5 shrink-0 text-subtle" />
        <span>{note}</span>
      </div>
    </LoopRunSection>
  );
}
