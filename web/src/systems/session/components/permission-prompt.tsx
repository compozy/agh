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
import type { PermissionRequest } from "../types";
import type { PermissionDecision } from "../adapters/session-api";
import { approveSession } from "../adapters/session-api";

export interface PermissionPromptProps {
  permission: PermissionRequest;
  sessionId: string;
  onResolved?: () => void;
}

export function PermissionPrompt({ permission, sessionId, onResolved }: PermissionPromptProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleDecision = useCallback(
    async (decision: PermissionDecision) => {
      setIsSubmitting(true);
      try {
        await approveSession(sessionId, {
          request_id: permission.requestId,
          turn_id: permission.turnId ?? "",
          decision,
        });
      } catch {
        toast.error("Failed to send permission response. The agent may continue waiting.");
      }
      onResolved?.();
      setIsSubmitting(false);
    },
    [sessionId, permission.requestId, permission.turnId, onResolved]
  );

  return (
    <div className="px-4 py-2" data-testid="permission-prompt">
      <Alert className="max-w-3xl px-3 py-3" variant="warning">
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
