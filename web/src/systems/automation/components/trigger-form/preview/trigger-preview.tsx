import { Eyebrow } from "@agh/ui";

import type { TriggerPreviewModel } from "../../../lib/trigger-preview";
import { PreviewSummary } from "./preview-summary";
import { RenderedPrompt } from "./rendered-prompt";
import { SampleEventCard } from "./sample-event-card";
import { WebhookEndpointCard } from "./webhook-endpoint-card";

interface TriggerPreviewProps {
  preview: TriggerPreviewModel;
}

/** Right-hand live preview pane: summary, sample event, rendered prompt, webhook. */
export function TriggerPreview({ preview }: TriggerPreviewProps) {
  return (
    <aside
      className="flex min-h-0 flex-col gap-3 overflow-y-auto border-l border-line-soft bg-canvas px-5 pt-5 pb-6 max-lg:border-t max-lg:border-l-0"
      data-testid="trigger-preview"
    >
      <div className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="size-1.5 rounded-full bg-success ring-[3px] ring-success-tint"
        />
        <Eyebrow className="text-subtle">Live preview</Eyebrow>
      </div>
      <PreviewSummary summary={preview.summary} />
      <SampleEventCard
        json={preview.json}
        matchLabel={preview.matchLabel}
        matchState={preview.matchState}
      />
      <RenderedPrompt rendered={preview.rendered} templateTokens={preview.templateTokens} />
      {preview.webhook ? (
        <WebhookEndpointCard curl={preview.webhook.curl} url={preview.webhook.url} />
      ) : null}
    </aside>
  );
}
