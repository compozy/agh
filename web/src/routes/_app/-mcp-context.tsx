import { Eyebrow, Pill, SkeletonRows } from "@agh/ui";

const LEGEND = [
  { label: "ready or confirmed", tone: "success" as const },
  { label: "operator action", tone: "warning" as const },
  { label: "repair required", tone: "danger" as const },
  { label: "not run or not used", tone: "neutral" as const },
];

interface MCPContextHeaderProps {
  scope: "workspace" | "global";
  workspaceName: string;
}

export function MCPContextHeader({ scope, workspaceName }: MCPContextHeaderProps) {
  const scopeContext =
    scope === "global" ? "global · operator home" : `workspace · ${workspaceName}`;
  return (
    <div className="mb-4 flex items-start gap-4" data-testid="settings-page-mcp-context">
      <div className="min-w-0 flex-1">
        <Eyebrow className="text-muted">Runtime inventory</Eyebrow>
        <h2 className="mt-1 text-lg font-medium tracking-tight text-fg-strong">Server status</h2>
        <p className="mt-1 max-w-2xl text-small-body text-muted">
          Configuration, authorization, runtime, and probe results are independent signals. A
          configured server is not assumed ready.
        </p>
        <div
          className="mt-1 font-mono text-caption text-muted"
          data-testid="settings-page-mcp-servers-scope-label"
        >
          {scopeContext}
        </div>
      </div>
      <div className="hidden max-w-[440px] flex-wrap justify-end gap-x-3 gap-y-1.5 pt-1 md:flex">
        {LEGEND.map(item => (
          <span key={item.label} className="flex items-center gap-1.5 text-caption text-subtle">
            <Pill.Dot tone={item.tone} />
            {item.label}
          </span>
        ))}
      </div>
    </div>
  );
}

export function MCPMatrixSkeleton() {
  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="settings-page-mcp-servers-loading"
    >
      <SkeletonRows count={5} className="p-3.5" />
    </div>
  );
}
