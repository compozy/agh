import { Code } from "lucide-react";

import type { CreateAutomationJobRequest } from "../../../types";
import { PreviewCard } from "../../trigger-form/preview/preview-card";

interface PayloadCardProps {
  payload: CreateAutomationJobRequest;
}

/** The exact request body that will be POSTed, as plain pretty-printed JSON. */
export function PayloadCard({ payload }: PayloadCardProps) {
  return (
    <PreviewCard
      icon={Code}
      label="Request payload"
      right={
        <span className="font-mono text-form-hint text-subtle">POST /api/automation/jobs</span>
      }
    >
      <pre className="overflow-x-auto rounded border border-line-soft bg-rail p-3 font-mono text-form-hint leading-relaxed whitespace-pre text-fg">
        {JSON.stringify(payload, null, 2)}
      </pre>
    </PreviewCard>
  );
}
