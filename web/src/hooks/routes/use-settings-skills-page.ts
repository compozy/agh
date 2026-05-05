import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useSettingsPage } from "@/hooks/routes/use-settings-page";
import { useAgents, type AgentPayload } from "@/systems/agent";
import {
  SettingsApiError,
  useSettingsSkills,
  useUpdateSettingsSkills,
  type SettingsSkillsFilter,
  type SettingsSkillsSection,
  type SettingsUpdateSkillsRequest,
} from "@/systems/settings";
import { useWorkspaces, type WorkspacePayload } from "@/systems/workspace";

type SkillsConfig = SettingsSkillsSection["config"];

export type SkillsScopeSelection =
  | { scope: "global" }
  | { scope: "agent"; agentName: string; workspaceId?: string };

function cloneDisabled(config: SkillsConfig): string[] {
  return [...(config.disabled_skills ?? [])];
}

function sameDisabled(a: string[] | undefined, b: string[] | undefined): boolean {
  const left = a ?? [];
  const right = b ?? [];
  if (left.length !== right.length) return false;
  for (let i = 0; i < left.length; i += 1) {
    if (left[i] !== right[i]) return false;
  }
  return true;
}

function samePolicy(a: SkillsConfig, b: SkillsConfig): boolean {
  if (a.enabled !== b.enabled) return false;
  if (a.poll_interval !== b.poll_interval) return false;
  if (a.marketplace.registry !== b.marketplace.registry) return false;
  if ((a.marketplace.base_url ?? "") !== (b.marketplace.base_url ?? "")) return false;
  if (
    JSON.stringify(a.allowed_marketplace_hooks ?? []) !==
    JSON.stringify(b.allowed_marketplace_hooks ?? [])
  ) {
    return false;
  }
  if (
    JSON.stringify(a.allowed_marketplace_mcp ?? []) !==
    JSON.stringify(b.allowed_marketplace_mcp ?? [])
  ) {
    return false;
  }
  return true;
}

function errorMessage(error: unknown): string | null {
  if (error instanceof SettingsApiError) return error.message;
  if (error instanceof Error) return error.message;
  return null;
}

function selectionToFilter(selection: SkillsScopeSelection): SettingsSkillsFilter {
  if (selection.scope === "agent") {
    return {
      scope: "agent",
      agent_name: selection.agentName,
      workspace_id: selection.workspaceId,
    };
  }

  return { scope: "global" };
}

function envelopeScopeKey(envelope: SettingsSkillsSection): string {
  return [envelope.scope, envelope.agent_name ?? "", envelope.workspace_id ?? ""].join(":");
}

function sortAgents(agents: AgentPayload[]): AgentPayload[] {
  return [...agents].sort((left, right) => left.name.localeCompare(right.name));
}

function pickDefaultAgentName(agents: AgentPayload[]): string {
  return agents[0]?.name ?? "";
}

export function useSettingsSkillsPage() {
  const page = useSettingsPage({ currentSlug: "skills" });
  const agentsQuery = useAgents();
  const workspaceQuery = useWorkspaces();
  const disabledMutation = useUpdateSettingsSkills();
  const policyMutation = useUpdateSettingsSkills();

  const agents = useMemo(() => sortAgents(agentsQuery.data ?? []), [agentsQuery.data]);
  const workspaces: WorkspacePayload[] = workspaceQuery.data ?? [];

  const [selection, setSelection] = useState<SkillsScopeSelection>({ scope: "global" });
  const filter = useMemo(() => selectionToFilter(selection), [selection]);
  const query = useSettingsSkills(filter);
  const envelope = query.data ?? null;

  const [draft, setDraft] = useState<SkillsConfig | null>(null);
  const [lastDisabledLabel, setLastDisabledLabel] = useState<string | null>(null);
  const [lastPolicyLabel, setLastPolicyLabel] = useState<string | null>(null);
  const lastEnvelopeKeyRef = useRef("");

  useEffect(() => {
    if (selection.scope !== "agent") {
      return;
    }

    if (agents.length === 0) {
      setSelection({ scope: "global" });
      return;
    }

    if (agents.some(agent => agent.name === selection.agentName)) {
      return;
    }

    setSelection({
      scope: "agent",
      agentName: pickDefaultAgentName(agents),
      workspaceId: selection.workspaceId,
    });
  }, [agents, selection]);

  useEffect(() => {
    if (!envelope) {
      return;
    }

    const nextKey = envelopeScopeKey(envelope);
    if (lastEnvelopeKeyRef.current === nextKey && draft !== null) {
      return;
    }

    lastEnvelopeKeyRef.current = nextKey;
    setDraft(envelope.config);
    setLastDisabledLabel(null);
    setLastPolicyLabel(null);
  }, [draft, envelope]);

  const availableScopes = envelope?.available_scopes ?? ["global"];
  const selectedAgent = useMemo(
    () =>
      selection.scope === "agent"
        ? (agents.find(agent => agent.name === selection.agentName) ?? null)
        : null,
    [agents, selection]
  );
  const selectedWorkspaceContext = useMemo(
    () =>
      selection.scope === "agent" && selection.workspaceId
        ? (workspaces.find(workspace => workspace.id === selection.workspaceId) ?? null)
        : null,
    [selection, workspaces]
  );

  const isDisabledDirty = useMemo(() => {
    if (!envelope || !draft) return false;
    return !sameDisabled(envelope.config.disabled_skills, draft.disabled_skills);
  }, [draft, envelope]);

  const isPolicyDirty = useMemo(() => {
    if (!envelope || !draft || selection.scope === "agent") return false;
    return !samePolicy(envelope.config, draft);
  }, [draft, envelope, selection.scope]);

  const resetScopedState = useCallback(() => {
    disabledMutation.reset();
    policyMutation.reset();
    setLastDisabledLabel(null);
    setLastPolicyLabel(null);
  }, [disabledMutation, policyMutation]);

  const handleResetDisabled = useCallback(() => {
    if (!envelope || !draft) return;
    setDraft({ ...draft, disabled_skills: cloneDisabled(envelope.config) });
  }, [draft, envelope]);

  const handleResetPolicy = useCallback(() => {
    if (!envelope || !draft || selection.scope === "agent") return;
    setDraft({
      ...envelope.config,
      disabled_skills: draft.disabled_skills,
    });
  }, [draft, envelope, selection.scope]);

  const handleSaveDisabled = useCallback(() => {
    if (!envelope || !draft) return;
    const body: SettingsUpdateSkillsRequest = {
      config: {
        ...envelope.config,
        disabled_skills: draft.disabled_skills ?? [],
      },
    };
    disabledMutation.mutate(
      { body, filter },
      {
        onSuccess: () => {
          setLastDisabledLabel("Saved · applied immediately");
        },
      }
    );
  }, [disabledMutation, draft, envelope, filter]);

  const handleSavePolicy = useCallback(() => {
    if (!envelope || !draft || selection.scope === "agent") return;
    const body: SettingsUpdateSkillsRequest = {
      config: {
        ...draft,
        disabled_skills: envelope.config.disabled_skills ?? [],
      },
    };
    policyMutation.mutate(
      { body, filter },
      {
        onSuccess: () => {
          setLastPolicyLabel("Saved · restart required to apply");
        },
      }
    );
  }, [draft, envelope, filter, policyMutation, selection.scope]);

  const toggleDisabled = useCallback(
    (name: string) => {
      if (!draft) return;
      const current = draft.disabled_skills ?? [];
      const next = current.includes(name)
        ? current.filter(entry => entry !== name)
        : [...current, name].sort();
      setDraft({ ...draft, disabled_skills: next });
    },
    [draft]
  );

  const handleRetry = useCallback(() => {
    void query.refetch();
  }, [query]);

  const selectGlobal = useCallback(() => {
    resetScopedState();
    setSelection({ scope: "global" });
  }, [resetScopedState]);

  const selectAgentScope = useCallback(() => {
    if (agents.length === 0) {
      return;
    }
    resetScopedState();
    setSelection(current => ({
      scope: "agent",
      agentName:
        current.scope === "agent" && current.agentName.trim().length > 0
          ? current.agentName
          : pickDefaultAgentName(agents),
      workspaceId: current.scope === "agent" ? current.workspaceId : undefined,
    }));
  }, [agents, resetScopedState]);

  const selectAgent = useCallback(
    (agentName: string) => {
      if (agentName.trim().length === 0) {
        return;
      }
      resetScopedState();
      setSelection(current => ({
        scope: "agent",
        agentName,
        workspaceId: current.scope === "agent" ? current.workspaceId : undefined,
      }));
    },
    [resetScopedState]
  );

  const selectWorkspaceContext = useCallback(
    (workspaceId: string) => {
      resetScopedState();
      setSelection(current => {
        if (current.scope !== "agent") {
          return current;
        }
        return {
          ...current,
          workspaceId: workspaceId.trim().length > 0 ? workspaceId : undefined,
        };
      });
    },
    [resetScopedState]
  );

  return {
    isLoading: query.isLoading || (selection.scope === "agent" && agentsQuery.isLoading),
    error: query.error,
    envelope,
    draft,
    setDraft,
    toggleDisabled,
    isDisabledDirty,
    isPolicyDirty,
    handleResetDisabled,
    handleResetPolicy,
    handleSaveDisabled,
    handleSavePolicy,
    isSavingDisabled: disabledMutation.isPending,
    isSavingPolicy: policyMutation.isPending,
    saveDisabledError: errorMessage(disabledMutation.error),
    savePolicyError: errorMessage(policyMutation.error),
    disabledWarnings: disabledMutation.data?.warnings,
    policyWarnings: policyMutation.data?.warnings,
    lastDisabledLabel,
    lastPolicyLabel,
    handleRetry,
    restart: page.restart,
    availableScopes,
    selection,
    agents,
    workspaces,
    selectedAgent,
    selectedWorkspaceContext,
    selectGlobal,
    selectAgentScope,
    selectAgent,
    selectWorkspaceContext,
  };
}
