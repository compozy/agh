import type { ComponentType, ReactNode } from "react";

import { PageHeader, Section } from "@agh/ui";

interface WorkspacePageShellProps {
  title: string;
  icon: ComponentType<{ className?: string; size?: number }>;
  count: number;
  controls?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
}

// Shared workspace-domain page scaffold exported through the workspace system barrel
// for route composition, stories, and test harnesses that need a consistent shell.
export function WorkspacePageShell({
  title,
  icon,
  count,
  controls,
  meta,
  children,
}: WorkspacePageShellProps) {
  return (
    <div className="flex flex-1 flex-col overflow-hidden" data-testid="workspace-page-shell">
      <PageHeader title={title} icon={icon} count={count} controls={controls} meta={meta} />
      <Section className="min-h-0 flex-1 overflow-hidden">
        <div className="flex min-h-0 flex-1 overflow-hidden">{children}</div>
      </Section>
    </div>
  );
}
