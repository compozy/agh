import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { Button, cn } from "@agh/ui";

import type { CurlLine } from "../../../lib/trigger-preview";
import { PreviewCard } from "./preview-card";

interface WebhookEndpointCardProps {
  url: string;
  curl: CurlLine[];
}

/** Real webhook endpoint URL + signed curl example, with copy-to-clipboard. */
export function WebhookEndpointCard({ url, curl }: WebhookEndpointCardProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    void navigator.clipboard
      ?.writeText(url)
      .then(() => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1200);
      })
      .catch(() => undefined);
  };

  return (
    <PreviewCard label="Webhook endpoint">
      <div className="mb-2.5 flex items-center gap-2 rounded border border-line-soft bg-rail px-2.5 py-2">
        <span className="rounded-xs bg-success-tint px-1.5 py-0.5 font-mono text-badge font-semibold text-success">
          POST
        </span>
        <span className="flex-1 truncate font-mono text-form-hint text-fg">{url}</span>
        <Button
          aria-label="Copy webhook URL"
          onClick={handleCopy}
          size="icon-xs"
          type="button"
          variant="ghost"
        >
          {copied ? (
            <Check aria-hidden="true" className="text-success" />
          ) : (
            <Copy aria-hidden="true" />
          )}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded border border-line-soft bg-rail p-3 font-mono text-form-hint leading-relaxed whitespace-pre text-fg">
        {curl.map(line => (
          <div key={line.map(segment => segment.text).join("")}>
            {line.map(segment => (
              <span
                key={segment.text}
                className={cn(
                  segment.tone === "keyword" && "text-info",
                  segment.tone === "string" && "text-success"
                )}
              >
                {segment.text}
              </span>
            ))}
          </div>
        ))}
      </pre>
    </PreviewCard>
  );
}
