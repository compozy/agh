import { useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Loader2, Wrench } from "lucide-react";

import { PillButton } from "@/components/design-system";
import {
  useSkills,
  useSkill,
  useSkillContent,
  useDisableSkill,
  useEnableSkill,
  SkillListPanel,
  SkillDetailPanel,
  MarketplaceView,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";
import { WorkspacePageShell } from "@/systems/workspace/components/workspace-page-shell";

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
  const [requestedSkillContentName, setRequestedSkillContentName] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";

  // Data hooks
  const { data: skills, isLoading, error } = useSkills(workspaceId);
  const {
    data: selectedSkill,
    isLoading: isLoadingDetail,
    error: detailError,
  } = useSkill(selectedSkillName ?? "", workspaceId);

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

  const shouldLoadSelectedContent =
    effectiveSelectedName !== null && requestedSkillContentName === effectiveSelectedName;

  const {
    data: selectedSkillContent,
    isLoading: isLoadingContent,
    error: contentError,
    refetch: refetchSkillContent,
  } = useSkillContent(effectiveSelectedName ?? "", workspaceId, shouldLoadSelectedContent);

  const handleDisable = (name: string) => {
    disableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleEnable = (name: string) => {
    enableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleViewContent = (name: string) => {
    setRequestedSkillContentName(name);
  };

  const handleRetryContent = () => {
    void refetchSkillContent();
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
    <WorkspacePageShell
      title="Skills"
      icon={<Wrench className="size-4" />}
      count={skillCount}
      controls={
        <div className="flex items-center gap-1.5" data-testid="tab-pills">
          <PillButton
            active={activeTab === "installed"}
            data-testid="tab-installed"
            onClick={() => setActiveTab("installed")}
          >
            INSTALLED
          </PillButton>
          <PillButton
            active={activeTab === "marketplace"}
            data-testid="tab-marketplace"
            onClick={() => setActiveTab("marketplace")}
          >
            MARKETPLACE
          </PillButton>
        </div>
      }
    >
      {activeTab === "installed" ? (
        <>
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
            content={shouldLoadSelectedContent ? selectedSkillContent : undefined}
            isContentLoading={shouldLoadSelectedContent && isLoadingContent}
            contentError={shouldLoadSelectedContent ? contentError : null}
            onViewContent={handleViewContent}
            onRetryContent={handleRetryContent}
            onDisable={handleDisable}
            onEnable={handleEnable}
            isActionPending={disableMutation.isPending || enableMutation.isPending}
          />
        </>
      ) : (
        <MarketplaceView
          skills={skills ?? []}
          installedSkillNames={installedSkillNames}
          installUnavailableReason="Marketplace install is not implemented yet"
          isInstalling={false}
        />
      )}
    </WorkspacePageShell>
  );
}
