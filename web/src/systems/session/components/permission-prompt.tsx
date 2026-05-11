import { Check, ShieldAlert, ShieldCheck, ShieldOff, X } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

import { Button, CodeBlock, Eyebrow, cn } from "@agh/ui";

import type { PermissionDecision } from "../adapters/session-api";
import { approveSession } from "../adapters/session-api";
import { toPermissionRequest } from "../lib/message-parts";
import type { AghPermissionData, PermissionRequest } from "../types";

export interface PermissionPromptProps {
  permission: PermissionRequest;
  sessionId: string;
  onResolved?: () => void;
}

type PromptTone = "warning" | "danger";

/**
 * High-stakes tools that affect the filesystem or the network.,
 * permission prompts for these tools render with the `danger` tone (tile +
 * tint) so the operator cannot scroll past them without a deliberate decision.
 */
const HIGH_STAKES_TOOLS = new Set([
  "Bash",
  "Write",
  "Edit",
  "NotebookEdit",
  "WebFetch",
  "WebSearch",
]);

function promptToneFor(toolName: string | undefined): PromptTone {
  if (toolName && HIGH_STAKES_TOOLS.has(toolName)) return "danger";
  return "warning";
}

function normalizePermissionDecision(value: string | undefined): PermissionDecision | null {
  switch (value?.trim()) {
    case "allow-once":
      return "allow-once";
    case "allow-always":
      return "allow-always";
    case "reject-once":
      return "reject-once";
    case "reject-always":
      return "reject-always";
    default:
      return null;
  }
}

export function PermissionPrompt({ permission, sessionId, onResolved }: PermissionPromptProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isResolved, setIsResolved] = useState(false);

  const handleDecision = useCallback(
    async (decision: PermissionDecision) => {
      setIsSubmitting(true);
      try {
        await approveSession(sessionId, {
          request_id: permission.requestId,
          turn_id: permission.turnId ?? "",
          decision,
        });
        setIsResolved(true);
        onResolved?.();
      } catch {
        toast.error("Failed to send permission response. The agent may continue waiting.");
      } finally {
        setIsSubmitting(false);
      }
    },
    [sessionId, permission.requestId, permission.turnId, onResolved]
  );

  const tone = promptToneFor(permission.toolName);
  const isHighStakes = tone === "danger";

  return isResolved ? null : (
    <div
      className="sticky top-2 z-10 px-4 py-2"
      data-testid="permission-prompt"
      data-tone={tone}
      data-sticky="true"
      role="region"
      aria-label="Permission required"
    >
      <div
        className={cn(
          "flex max-w-3xl gap-3 rounded-lg p-3 shadow-highlight",
          isHighStakes ? "bg-danger-tint" : "bg-warning-tint"
        )}
        data-testid="permission-prompt-card"
      >
        <PermissionToneTile tone={tone} />
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <header className="flex flex-col gap-1">
            <Eyebrow
              data-testid="permission-prompt-eyebrow"
              className={cn(isHighStakes ? "text-danger" : "text-warning")}
            >
              Permission Required
            </Eyebrow>
            <p className="text-[13px] text-fg">
              The agent is requesting permission before it continues this turn.
            </p>
          </header>
          <dl className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted">
            <div className="flex items-center gap-1.5">
              <dt className="text-subtle">Tool</dt>
              <dd className="font-mono text-fg">{permission.toolName}</dd>
            </div>
            {permission.action ? (
              <div className="flex items-center gap-1.5">
                <dt className="text-subtle">Action</dt>
                <dd>{permission.action}</dd>
              </div>
            ) : null}
            {permission.resource ? (
              <div className="flex min-w-0 items-center gap-1.5">
                <dt className="text-subtle">Resource</dt>
                <dd className="truncate font-mono text-fg">{permission.resource}</dd>
              </div>
            ) : null}
          </dl>
          {Object.keys(permission.toolInput).length > 0 ? (
            <CodeBlock
              code={JSON.stringify(permission.toolInput, null, 2)}
              copyable={false}
              data-testid="permission-tool-input"
              showPrompt={false}
              tone={isHighStakes ? "danger" : "warning"}
              truncateLines={6}
            />
          ) : null}
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <Button
              variant={isHighStakes ? "outline" : "default"}
              size="sm"
              disabled={isSubmitting}
              onClick={() => handleDecision("allow-once")}
              data-testid="permission-allow-once"
            >
              <Check className="size-3" />
              Allow Once
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={isSubmitting}
              onClick={() => handleDecision("allow-always")}
              data-testid="permission-allow-always"
            >
              <ShieldCheck className="size-3" />
              Allow Always
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={isSubmitting}
              onClick={() => handleDecision("reject-once")}
              data-testid="permission-reject-once"
            >
              <X className="size-3" />
              Reject Once
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={isSubmitting}
              onClick={() => handleDecision("reject-always")}
              data-testid="permission-reject-always"
            >
              <ShieldOff className="size-3" />
              Reject Always
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface PermissionToneTileProps {
  tone: PromptTone;
}

function PermissionToneTile({ tone }: PermissionToneTileProps) {
  const isDanger = tone === "danger";
  return (
    <span
      data-testid="permission-prompt-tile"
      data-tone={tone}
      aria-hidden="true"
      className={cn(
        "flex size-6 shrink-0 items-center justify-center rounded",
        "text-canvas",
        isDanger ? "bg-danger" : "bg-warning"
      )}
    >
      <ShieldAlert className="size-3" />
    </span>
  );
}

export function PermissionDataPart({
  data,
  sessionId,
}: {
  data: AghPermissionData;
  sessionId: string;
}) {
  const decision = normalizePermissionDecision(data.decision);
  const permission = toPermissionRequest(data);
  switch (decision) {
    case "allow-once":
    case "allow-always":
      return null;
    case "reject-once":
    case "reject-always":
      return <PermissionRejectedNotice permission={permission} />;
    default:
      return <PermissionPrompt permission={permission} sessionId={sessionId} />;
  }
}

function PermissionRejectedNotice({ permission }: { permission: PermissionRequest }) {
  return (
    <div className="px-4 py-2" data-testid="permission-rejected-notice" role="status">
      <div
        className={cn(
          "flex max-w-3xl items-start gap-2 rounded-md border px-3 py-2",
          "border-danger/30 bg-danger/8",
          "text-xs text-danger"
        )}
      >
        <ShieldOff aria-hidden="true" className="mt-0.5 size-3 shrink-0" />
        <div className="min-w-0 flex-1">
          <div className="font-medium text-fg">Permission Rejected</div>
          <div className="mt-1 flex min-w-0 flex-wrap gap-x-2 gap-y-1 text-muted">
            <span className="font-mono">{permission.toolName}</span>
            {permission.resource ? <span className="truncate">{permission.resource}</span> : null}
          </div>
        </div>
      </div>
    </div>
  );
}
