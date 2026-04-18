import type { ReactNode } from "react";

import { Button, Empty } from "@agh/ui";

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
      <Empty
        className="max-w-xl"
        icon={icon}
        title={title}
        description={description}
        action={
          actionLabel && onAction ? (
            <Button onClick={onAction} size="lg" type="button">
              {actionLabel}
            </Button>
          ) : undefined
        }
      />
    </div>
  );
}
