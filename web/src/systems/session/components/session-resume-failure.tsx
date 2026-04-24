import { AlertTriangle, Loader2, RefreshCw, X } from "lucide-react";

import { Button, MonoBadge } from "@agh/ui";

import { cn } from "@/lib/utils";

export interface SessionResumeFailureProps {
  sessionId: string;
  message: string;
  missingProvider: string | null;
  agentName?: string | null;
  isRetrying: boolean;
  onRetry: () => void;
  onDismiss: () => void;
}

export function SessionResumeFailure({
  sessionId,
  message,
  missingProvider,
  agentName,
  isRetrying,
  onRetry,
  onDismiss,
}: SessionResumeFailureProps) {
  const normalizedMissingProvider = missingProvider?.trim() ?? "";
  const normalizedAgentName = agentName?.trim() ?? "";
  const hasProviderDetail = normalizedMissingProvider.length > 0;
  const hasAgentDetail = normalizedAgentName.length > 0;
  const title = hasProviderDetail ? "Resume failed: provider no longer available" : "Resume failed";

  return (
    <div
      aria-live="assertive"
      className={cn(
        "mx-4 mt-3 flex flex-col gap-3 rounded-[var(--radius-md)] border border-[color:var(--color-danger)]/60",
        "bg-[color:var(--color-surface-panel)] px-4 py-3"
      )}
      data-testid="session-resume-failure"
      role="alert"
    >
      <div className="flex items-start gap-3">
        <AlertTriangle
          aria-hidden="true"
          className="mt-0.5 size-4 shrink-0 text-[color:var(--color-danger)]"
        />
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <span
              className="text-sm font-semibold text-[color:var(--color-text-primary)]"
              data-testid="session-resume-failure-title"
            >
              {title}
            </span>
            {hasProviderDetail ? (
              <MonoBadge data-testid="session-resume-failure-provider" tone="danger">
                {normalizedMissingProvider}
              </MonoBadge>
            ) : null}
          </div>
          <p
            className="text-xs leading-5 text-[color:var(--color-text-secondary)]"
            data-testid="session-resume-failure-message"
          >
            {hasProviderDetail
              ? `This session was started with provider ${normalizedMissingProvider}, which is not visible in the current workspace configuration. Add the provider back to the workspace or update the agent defaults before retrying.`
              : message}
          </p>
          <dl
            className="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-xs uppercase tracking-[var(--tracking-mono)] text-[color:var(--color-text-tertiary)]"
            data-testid="session-resume-failure-meta"
          >
            <div className="flex items-center gap-1.5">
              <dt>session</dt>
              <dd className="normal-case tracking-normal text-[color:var(--color-text-secondary)]">
                {sessionId}
              </dd>
            </div>
            {hasAgentDetail ? (
              <div className="flex items-center gap-1.5">
                <dt>agent</dt>
                <dd className="normal-case tracking-normal text-[color:var(--color-text-secondary)]">
                  {normalizedAgentName}
                </dd>
              </div>
            ) : null}
          </dl>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-end gap-2">
        <Button
          data-testid="session-resume-failure-dismiss"
          onClick={onDismiss}
          size="sm"
          type="button"
          variant="ghost"
        >
          <X aria-hidden="true" className="size-3.5" />
          Dismiss
        </Button>
        <Button
          data-testid="session-resume-failure-retry"
          disabled={isRetrying}
          onClick={onRetry}
          size="sm"
          type="button"
          variant="outline"
        >
          {isRetrying ? (
            <Loader2 aria-hidden="true" className="size-3.5 animate-spin" />
          ) : (
            <RefreshCw aria-hidden="true" className="size-3.5" />
          )}
          Retry resume
        </Button>
      </div>
    </div>
  );
}
