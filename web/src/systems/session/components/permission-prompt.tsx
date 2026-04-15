import { useCallback, useState } from "react";
import { ShieldAlert, Check, X, ShieldCheck, ShieldOff } from "lucide-react";
import { toast } from "sonner";

import { Button, Card, CardContent, CardFooter, CardHeader, CardTitle } from "@agh/ui";
import { cn } from "@/lib/utils";
import type { PermissionRequest } from "../types";
import type { PermissionDecision } from "../adapters/session-api";
import { approveSession } from "../adapters/session-api";

export interface PermissionPromptProps {
  permission: PermissionRequest;
  sessionId: string;
  onResolved: () => void;
}

export function PermissionPrompt({ permission, sessionId, onResolved }: PermissionPromptProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleDecision = useCallback(
    async (decision: PermissionDecision) => {
      setIsSubmitting(true);
      try {
        await approveSession(sessionId, {
          request_id: permission.requestId,
          turn_id: "",
          decision,
        });
      } catch {
        toast.error("Failed to send permission response. The agent may continue waiting.");
      }
      onResolved();
      setIsSubmitting(false);
    },
    [sessionId, permission.requestId, onResolved]
  );

  return (
    <div className="px-4 py-2" data-testid="permission-prompt">
      <Card className={cn("border-amber-500/40 bg-amber-500/5", "")}>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-sm font-medium">
            <ShieldAlert className="size-4 text-amber-500" />
            Permission Required
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 pb-3">
          <div className="flex flex-col gap-1 text-xs">
            <div className="flex gap-2">
              <span className="font-medium text-[color:var(--color-text-tertiary)]">Tool:</span>
              <span className="font-mono">{permission.toolName}</span>
            </div>
            {permission.action && (
              <div className="flex gap-2">
                <span className="font-medium text-[color:var(--color-text-tertiary)]">Action:</span>
                <span>{permission.action}</span>
              </div>
            )}
            {permission.resource && (
              <div className="flex gap-2">
                <span className="font-medium text-[color:var(--color-text-tertiary)]">
                  Resource:
                </span>
                <span className="truncate font-mono">{permission.resource}</span>
              </div>
            )}
          </div>
          {Object.keys(permission.toolInput).length > 0 && (
            <pre
              className={cn(
                "max-h-32 overflow-auto rounded-md p-2 text-[0.7rem] leading-relaxed",
                "bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
              )}
              data-testid="permission-tool-input"
            >
              {JSON.stringify(permission.toolInput, null, 2)}
            </pre>
          )}
        </CardContent>
        <CardFooter className="flex flex-wrap gap-2 pt-0">
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
        </CardFooter>
      </Card>
    </div>
  );
}
