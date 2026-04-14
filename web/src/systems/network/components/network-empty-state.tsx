import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

interface NetworkEmptyStateProps {
  actionLabel?: string;
  description: string;
  icon: ReactNode;
  onAction?: () => void;
  testId: string;
  title: string;
}

export function NetworkEmptyState({
  actionLabel,
  description,
  icon,
  onAction,
  testId,
  title,
}: NetworkEmptyStateProps) {
  return (
    <div className="flex flex-1 items-center justify-center px-6 py-8" data-testid={testId}>
      <Empty className="max-w-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-8 py-10">
        <EmptyHeader className="gap-4">
          <EmptyMedia className="flex size-12 items-center justify-center rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
            {icon}
          </EmptyMedia>
          <div className="space-y-2">
            <EmptyTitle className="text-base font-semibold text-[color:var(--color-text-primary)]">
              {title}
            </EmptyTitle>
            <EmptyDescription className="max-w-md text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
              {description}
            </EmptyDescription>
          </div>
        </EmptyHeader>
        {actionLabel && onAction ? (
          <EmptyContent className="mt-2">
            <Button onClick={onAction} size="lg" type="button">
              {actionLabel}
            </Button>
          </EmptyContent>
        ) : null}
      </Empty>
    </div>
  );
}
