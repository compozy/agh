import { AlertTriangle, RefreshCw, X } from "lucide-react";

import {
  Alert,
  AlertActions,
  AlertDescription,
  AlertMeta,
  AlertTitle,
  Button,
  MetadataList,
  Pill,
  Spinner,
} from "@agh/ui";

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
    <Alert
      aria-live="assertive"
      className="mx-4 mt-3 px-4 py-3"
      data-testid="session-resume-failure"
      variant="danger"
    >
      <AlertTriangle aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <AlertTitle
        className="flex items-center gap-2 text-sm font-medium"
        data-testid="session-resume-failure-title"
      >
        {title}
        {hasProviderDetail ? (
          <Pill mono data-testid="session-resume-failure-provider" tone="danger">
            {normalizedMissingProvider}
          </Pill>
        ) : null}
      </AlertTitle>
      <AlertDescription className="text-xs leading-5" data-testid="session-resume-failure-message">
        {hasProviderDetail
          ? `This session was started with provider ${normalizedMissingProvider}, which is not visible in the current workspace configuration. Add the provider back to the workspace or update the agent defaults before retrying.`
          : message}
      </AlertDescription>
      <AlertMeta data-testid="session-resume-failure-meta">
        <MetadataList className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <MetadataList.Row>
            <MetadataList.Term>session</MetadataList.Term>
            <MetadataList.Value className="font-mono">{sessionId}</MetadataList.Value>
          </MetadataList.Row>
          {hasAgentDetail ? (
            <MetadataList.Row>
              <MetadataList.Term>agent</MetadataList.Term>
              <MetadataList.Value className="font-mono">{normalizedAgentName}</MetadataList.Value>
            </MetadataList.Row>
          ) : null}
        </MetadataList>
      </AlertMeta>
      <AlertActions>
        <Button
          data-testid="session-resume-failure-dismiss"
          onClick={onDismiss}
          size="sm"
          type="button"
          variant="ghost"
        >
          <X aria-hidden="true" className="size-3" />
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
            <Spinner className="size-3" />
          ) : (
            <RefreshCw aria-hidden="true" className="size-3" />
          )}
          Retry resume
        </Button>
      </AlertActions>
    </Alert>
  );
}
