import type { ReactNode } from "react";

interface WorkspacePageShellProps {
  title: string;
  icon: ReactNode;
  count: number;
  controls?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
}

export function WorkspacePageShell({
  title,
  icon,
  count,
  controls,
  meta,
  children,
}: WorkspacePageShellProps) {
  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <div className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <span className="text-[color:var(--color-text-primary)]">{icon}</span>
        <h1 className="text-base font-semibold text-[color:var(--color-text-primary)]">{title}</h1>
        <span className="inline-flex h-5 items-center rounded-md bg-[color:var(--color-surface-panel)] px-1.5 font-mono text-[0.64rem] text-[color:var(--color-text-secondary)]">
          {count}
        </span>
        {controls && <div className="ml-4 flex items-center gap-1.5">{controls}</div>}
        {meta && <div className="ml-auto flex items-center gap-2">{meta}</div>}
      </div>

      <div className="flex min-h-0 flex-1 overflow-hidden">{children}</div>
    </div>
  );
}
