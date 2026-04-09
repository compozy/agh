import { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Loader2, Wrench } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  useSkills,
  useSkill,
  useDisableSkill,
  useEnableSkill,
  SkillListPanel,
  SkillDetailPanel,
  MarketplaceView,
} from "@/systems/skill";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app/skills")({
  component: SkillsPage,
});

// ---------------------------------------------------------------------------
// Tab type
// ---------------------------------------------------------------------------

type Tab = "installed" | "marketplace";

// ---------------------------------------------------------------------------
// Skills Page
// ---------------------------------------------------------------------------

function SkillsPage() {
  const [activeTab, setActiveTab] = useState<Tab>("installed");
  const [selectedSkillName, setSelectedSkillName] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Active workspace (same logic as sidebar: first workspace)
  const { data: workspaces } = useWorkspaces();
  const activeWorkspaceId = workspaces?.[0]?.id ?? "";

  // Data hooks
  const { data: skills, isLoading, error } = useSkills(activeWorkspaceId);
  const {
    data: selectedSkill,
    isLoading: isLoadingDetail,
    error: detailError,
  } = useSkill(selectedSkillName ?? "", activeWorkspaceId);

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();

  const skillCount = skills?.length ?? 0;

  const installedSkillNames = useMemo(() => {
    if (!skills) return new Set<string>();
    return new Set(skills.map(s => s.name));
  }, [skills]);

  // Auto-select first skill if none selected
  const effectiveSelectedName = useMemo(() => {
    if (selectedSkillName && skills?.some(s => s.name === selectedSkillName)) {
      return selectedSkillName;
    }
    return skills?.[0]?.name ?? null;
  }, [selectedSkillName, skills]);

  const handleDisable = (name: string) => {
    disableMutation.mutate({ name, workspace: activeWorkspaceId });
  };

  const handleEnable = (name: string) => {
    enableMutation.mutate({ name, workspace: activeWorkspaceId });
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="skills-loading">
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-1 items-center justify-center" data-testid="skills-error">
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {error.message ?? "Failed to load skills"}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Page header bar */}
      <div className="flex items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3">
        <Wrench className="size-4 text-[color:var(--color-text-primary)]" />
        <h1 className="text-base font-semibold text-[color:var(--color-text-primary)]">Skills</h1>
        <span className="inline-flex h-[22px] items-center rounded-md bg-[color:var(--color-surface-elevated)] px-2 text-xs text-[color:var(--color-text-secondary)]">
          {skillCount}
        </span>

        {/* Tab pills */}
        <div className="ml-4 flex items-center gap-1.5" data-testid="tab-pills">
          <button
            onClick={() => setActiveTab("installed")}
            className={cn(
              "inline-flex h-8 items-center rounded-full px-3.5 text-sm transition-colors",
              activeTab === "installed"
                ? "bg-[#E8572A] text-white"
                : "border border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]"
            )}
            data-testid="tab-installed"
          >
            INSTALLED
          </button>
          <button
            onClick={() => setActiveTab("marketplace")}
            className={cn(
              "inline-flex h-8 items-center rounded-full px-3.5 text-sm transition-colors",
              activeTab === "marketplace"
                ? "bg-[#E8572A] text-white"
                : "border border-[color:var(--color-divider)] text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]"
            )}
            data-testid="tab-marketplace"
          >
            MARKETPLACE
          </button>
        </div>
      </div>

      {/* Content area */}
      {activeTab === "installed" ? (
        <div className="flex flex-1 overflow-hidden">
          <SkillListPanel
            skills={skills ?? []}
            selectedSkillName={effectiveSelectedName}
            onSelectSkill={setSelectedSkillName}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
          />
          <SkillDetailPanel
            skill={
              effectiveSelectedName
                ? (selectedSkill ?? skills?.find(s => s.name === effectiveSelectedName))
                : undefined
            }
            isLoading={isLoadingDetail && effectiveSelectedName !== null}
            error={detailError}
            onDisable={handleDisable}
            onEnable={handleEnable}
            isActionPending={disableMutation.isPending || enableMutation.isPending}
          />
        </div>
      ) : (
        <MarketplaceView
          skills={skills ?? []}
          installedSkillNames={installedSkillNames}
          onInstall={() => {}}
          isInstalling={false}
        />
      )}
    </div>
  );
}
