import { useCallback, useState } from "react";
import { ShieldAlert, Check, X, ShieldCheck, ShieldOff } from "lucide-react";
import { toast } from "sonner";

import {
  Alert,
  AlertActions,
  AlertDescription,
  AlertMeta,
  AlertTitle,
  Button,
  CodeBlock,
  MetadataList,
} from "@agh/ui";
import { cn } from "@/lib/utils";
import { toPermissionRequest } from "../lib/message-parts";
import type { AghPermissionData, PermissionRequest } from "../types";
import type { PermissionDecision } from "../adapters/session-api";
import { approveSession } from "../adapters/session-api";

export interface PermissionPromptProps {
  permission: PermissionRequest;
  sessionId: string;
  onResolved?: () => void;
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

  if (isResolved) {
    return null;
  }

  return (
    <div className="px-4 py-2" data-testid="permission-prompt">
      <Alert className="max-w-3xl p-3" variant="warning">
        <ShieldAlert className="size-4" />
        <AlertTitle>Permission Required</AlertTitle>
        <AlertDescription>
          The agent is requesting permission before it continues this turn.
        </AlertDescription>
        <AlertMeta>
          <MetadataList className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <MetadataList.Row>
              <MetadataList.Term>Tool</MetadataList.Term>
              <MetadataList.Value className="font-mono">{permission.toolName}</MetadataList.Value>
            </MetadataList.Row>
            {permission.action ? (
              <MetadataList.Row>
                <MetadataList.Term>Action</MetadataList.Term>
                <MetadataList.Value>{permission.action}</MetadataList.Value>
              </MetadataList.Row>
            ) : null}
            {permission.resource ? (
              <MetadataList.Row>
                <MetadataList.Term>Resource</MetadataList.Term>
                <MetadataList.Value className="truncate font-mono">
                  {permission.resource}
                </MetadataList.Value>
              </MetadataList.Row>
            ) : null}
          </MetadataList>
        </AlertMeta>
        <div className="mt-2 group-has-[>svg]/alert:col-start-2">
          {Object.keys(permission.toolInput).length > 0 && (
            <CodeBlock
              code={JSON.stringify(permission.toolInput, null, 2)}
              copyable={false}
              data-testid="permission-tool-input"
              showPrompt={false}
              tone="warning"
              truncateLines={6}
            />
          )}
        </div>
        <AlertActions>
          <Button
            variant="default"
            size="sm"
            disabled={isSubmitting}
            onClick={() => handleDecision("allow-once")}
            data-testid="permission-allow-once"
          >
            <Check className="size-3.5" />
            Allow Once
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={isSubmitting}
            onClick={() => handleDecision("allow-always")}
            data-testid="permission-allow-always"
          >
            <ShieldCheck className="size-3.5" />
            Allow Always
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={isSubmitting}
            onClick={() => handleDecision("reject-once")}
            data-testid="permission-reject-once"
          >
            <X className="size-3.5" />
            Reject Once
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={isSubmitting}
            onClick={() => handleDecision("reject-always")}
            data-testid="permission-reject-always"
          >
            <ShieldOff className="size-3.5" />
            Reject Always
          </Button>
        </AlertActions>
      </Alert>
    </div>
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
          "flex max-w-3xl items-start gap-2 rounded-[var(--radius-md)] border px-3 py-2",
          "border-[color:var(--color-danger)]/30 bg-[color:var(--color-danger)]/8",
          "text-xs text-[color:var(--color-danger)]"
        )}
      >
        <ShieldOff aria-hidden="true" className="mt-0.5 size-3.5 shrink-0" />
        <div className="min-w-0 flex-1">
          <div className="font-medium text-[color:var(--color-text-primary)]">
            Permission Rejected
          </div>
          <div className="mt-1 flex min-w-0 flex-wrap gap-x-2 gap-y-1 text-[color:var(--color-text-secondary)]">
            <span className="font-mono">{permission.toolName}</span>
            {permission.resource ? <span className="truncate">{permission.resource}</span> : null}
          </div>
        </div>
      </div>
    </div>
  );
}
