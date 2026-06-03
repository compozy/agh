import { Eyebrow } from "@agh/ui";

import type { JobPreviewModel } from "../../../lib/job-preview";
import { NextRunsCard } from "./next-runs-card";
import { PayloadCard } from "./payload-card";
import { RunDigestCard } from "./run-digest-card";
import { SummaryCard } from "./summary-card";

interface JobPreviewProps {
  preview: JobPreviewModel;
}

/** Right-hand live preview pane: summary, next runs, run digest, request payload. */
export function JobPreview({ preview }: JobPreviewProps) {
  return (
    <aside
      className="flex min-h-0 flex-col gap-3 overflow-y-auto border-l border-line-soft bg-canvas px-5 pt-5 pb-6 max-lg:border-t max-lg:border-l-0"
      data-testid="job-preview"
    >
      <div className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="size-1.5 rounded-full bg-success ring-[3px] ring-success-tint"
        />
        <Eyebrow className="text-subtle">Live preview</Eyebrow>
      </div>
      <SummaryCard summary={preview.summary} />
      <NextRunsCard emptyReason={preview.nextRunsEmptyReason} nextRuns={preview.nextRuns} />
      <RunDigestCard runDigest={preview.runDigest} />
      <PayloadCard payload={preview.payload} />
    </aside>
  );
}
