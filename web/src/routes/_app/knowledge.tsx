import { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Book, Loader2 } from "lucide-react";

import {
  useMemories,
  useMemory,
  useDeleteMemory,
  KnowledgeListPanel,
  KnowledgeDetailPanel,
} from "@/systems/knowledge";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app/knowledge")({
  component: KnowledgePage,
});

// ---------------------------------------------------------------------------
// Tab type
// ---------------------------------------------------------------------------

type Tab = "all" | "global" | "workspace";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function deriveScope(filename: string): string {
  if (filename.startsWith("workspace/") || filename.startsWith("ws/")) {
    return "workspace";
  }
  return "global";
}

function formatDreamStatus(lastConsolidation?: string): string {
  if (!lastConsolidation) return "Dream: never";
  try {
    const date = new Date(lastConsolidation);
    const diffMs = Date.now() - date.getTime();
    const diffH = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffH < 1) return "Dream: <1h ago";
    if (diffH < 24) return `Dream: ${diffH}h ago`;
    const diffD = Math.floor(diffH / 24);
    return `Dream: ${diffD}d ago`;
  } catch {
    return "Dream: unknown";
  }
}

// ---------------------------------------------------------------------------
// Knowledge Page
// ---------------------------------------------------------------------------

function KnowledgePage() {
  const [activeTab, setActiveTab] = useState<Tab>("all");
  const [selectedFilename, setSelectedFilename] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Active workspace
  const { data: workspaces } = useWorkspaces();
  const activeWorkspaceId = workspaces?.[0]?.id ?? "";

  // Determine scope filter for API
  const scopeFilter = activeTab === "all" ? undefined : activeTab;

  // Data hooks
  const {
    data: memories,
    isLoading,
    error,
  } = useMemories(scopeFilter, activeWorkspaceId || undefined);

  const deleteMutation = useDeleteMemory();

  const memoryCount = memories?.length ?? 0;

  // Auto-select first memory if none selected
  const effectiveSelectedFilename = useMemo(() => {
    if (selectedFilename && memories?.some(m => m.filename === selectedFilename)) {
      return selectedFilename;
    }
    return memories?.[0]?.filename ?? null;
  }, [selectedFilename, memories]);

  // Find the selected memory header
  const selectedMemory = useMemo(
    () => memories?.find(m => m.filename === effectiveSelectedFilename),
    [memories, effectiveSelectedFilename]
  );

  // Load content for selected memory
  const selectedScope = effectiveSelectedFilename ? deriveScope(effectiveSelectedFilename) : "";
  const {
    data: selectedContent,
    isLoading: isContentLoading,
    error: contentError,
  } = useMemory(selectedScope, effectiveSelectedFilename ?? "", activeWorkspaceId || undefined);

  const handleDelete = (scope: string, filename: string) => {
    deleteMutation.mutate({ scope, filename, workspace: activeWorkspaceId || undefined });
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="knowledge-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load knowledge"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Page header bar */}
      <div className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <Book className="size-4 text-[color:var(--color-text-primary)]" />
        <h1 className="text-base font-semibold text-[color:var(--color-text-primary)]">
          Knowledge
        </h1>
        <span className="inline-flex h-[22px] items-center rounded-md bg-[color:var(--color-surface-elevated)] px-2 text-xs text-[color:var(--color-text-secondary)]">
          {memoryCount}
        </span>

        {/* Tab pills */}
        <div className="ml-4 flex items-center gap-1.5" data-testid="tab-pills">
          {(["all", "global", "workspace"] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={
                activeTab === tab
                  ? "inline-flex h-8 items-center rounded-full px-3.5 text-sm transition-colors bg-[#E8572A] text-white"
                  : "inline-flex h-8 items-center rounded-full px-3.5 text-sm transition-colors border border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]"
              }
              data-testid={`tab-${tab}`}
            >
              {tab.toUpperCase()}
            </button>
          ))}
        </div>

        {/* Dream status indicator */}
        <div className="ml-auto flex items-center gap-1.5" data-testid="dream-status">
          <span className="size-2 rounded-full bg-[color:var(--color-success)]" />
          <span className="text-xs text-[color:var(--color-text-tertiary)]">
            {formatDreamStatus()}
          </span>
        </div>
      </div>

      {/* Content area: list + detail */}
      <div className="flex flex-1 overflow-hidden">
        <KnowledgeListPanel
          memories={memories ?? []}
          selectedFilename={effectiveSelectedFilename}
          onSelectMemory={setSelectedFilename}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
        />
        <KnowledgeDetailPanel
          memory={selectedMemory}
          content={selectedContent}
          isLoading={isContentLoading && effectiveSelectedFilename !== null}
          error={contentError}
          onDelete={handleDelete}
          isDeletePending={deleteMutation.isPending}
        />
      </div>
    </div>
  );
}
