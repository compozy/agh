import { Puzzle } from "lucide-react";

import {
  cn,
  DescriptionCard,
  MonoId,
  PageHead,
  PAGE_CONTENT_GUTTER,
  Pill,
  Section,
  Time,
} from "@agh/ui";

import { formatUptimeSeconds } from "@/lib/format-time";

import type { useExtensionDetailState } from "../hooks/use-extension-detail-state";
import {
  BundlesProvided,
  DetailBlock,
  Diagnostics,
  EnvironmentState,
  RailBlock,
  TokenBlock,
} from "./extension-detail-sections";
import {
  ExtensionProvenanceDialog,
  RemoveExtensionDialog,
  VerifiedMark,
} from "./extension-dialogs";

type ExtensionDetailState = ReturnType<typeof useExtensionDetailState>;

interface ExtensionDetailBodyProps {
  name: string;
  state: ExtensionDetailState;
}

/**
 * Loaded extension detail surface: PageHead + two-column body + dialogs.
 * Kept separate so `ExtensionDetail` stays under the giant-component lint cap.
 */
export function ExtensionDetailBody({ name, state }: ExtensionDetailBodyProps) {
  const {
    bundles,
    detail,
    navigate,
    provenanceOpen,
    removeOpen,
    setProvenanceOpen,
    setRemoveOpen,
  } = state;
  const data = detail.data;
  if (!data) return null;

  const { extension, listing } = data;
  const provenance = extension.provenance;
  const description =
    listing?.description ?? "No catalog description is available for this installed extension.";

  return (
    <div className="min-h-0 flex-1 overflow-y-auto" data-testid="extension-detail">
      <div className={cn(PAGE_CONTENT_GUTTER, "flex flex-col")}>
        <div className="pt-5">
          <PageHead
            data-testid="extension-detail-header"
            leading={
              <span className="grid size-(--size-provider-logo-well) place-items-center rounded-lg bg-elevated text-muted">
                <Puzzle aria-hidden="true" className="size-5" />
              </span>
            }
            title={name}
            variant="detail"
            meta={
              <>
                {extension.health_message ? (
                  <>
                    <span>{extension.health_message}</span>
                    <span>·</span>
                  </>
                ) : null}
                <span>{extension.source}</span>
                {provenance?.installed_at ? (
                  <>
                    <span>·</span>
                    <span>
                      installed <Time iso={provenance.installed_at} />
                    </span>
                  </>
                ) : null}
              </>
            }
            pills={
              <>
                <Pill mono size="xs">
                  {extension.type}
                </Pill>
                <Pill mono size="xs">
                  v{extension.version}
                </Pill>
                <Pill mono size="xs" tone={extension.daemon_running ? "success" : "neutral"}>
                  {extension.state}
                </Pill>
                {extension.health ? (
                  <Pill
                    mono
                    size="xs"
                    tone={extension.health === "healthy" ? "success" : "warning"}
                  >
                    {extension.health}
                  </Pill>
                ) : null}
              </>
            }
          />
        </div>
        <div className="grid gap-8 py-7 lg:grid-cols-[minmax(0,1fr)_20rem]">
          <main className="space-y-7">
            <Section label="Description">
              <DescriptionCard>{description}</DescriptionCard>
            </Section>
            <TokenBlock label="Capabilities" values={extension.capabilities ?? []} />
            <TokenBlock label="Actions" values={extension.actions ?? []} />
            <DetailBlock label="Environment">
              <EnvironmentState
                required={extension.requires_env ?? []}
                missing={extension.missing_env ?? []}
              />
            </DetailBlock>
            <DetailBlock label="Diagnostics">
              <Diagnostics
                diagnostics={extension.diagnostics ?? []}
                lastError={extension.last_error}
              />
            </DetailBlock>
          </main>
          <aside className="space-y-7">
            <RailBlock
              label="Runtime"
              rows={[
                {
                  term: "Daemon",
                  value: (
                    <Pill mono size="xs" tone={extension.daemon_running ? "success" : "neutral"}>
                      {extension.daemon_running ? "running" : "stopped"}
                    </Pill>
                  ),
                },
                {
                  term: "PID",
                  value: (
                    <code className="font-mono text-xs text-fg">
                      {extension.pid ? String(extension.pid) : "—"}
                    </code>
                  ),
                },
                {
                  term: "Uptime",
                  value: (
                    <code className="font-mono text-xs text-fg">
                      {formatUptimeSeconds(extension.uptime_seconds)}
                    </code>
                  ),
                },
                {
                  term: "Last error",
                  value: extension.last_error ? (
                    <span className="text-xs text-danger">{extension.last_error}</span>
                  ) : (
                    <code className="font-mono text-xs text-fg">—</code>
                  ),
                },
              ]}
            />
            <RailBlock
              label="Provenance"
              rows={[
                {
                  term: "Installed from",
                  value: (
                    <code className="break-all font-mono text-xs text-fg">
                      {provenance?.installed_from ?? extension.source}
                    </code>
                  ),
                },
                {
                  term: "Source",
                  value: (
                    <code className="break-all font-mono text-xs text-fg">
                      {provenance?.source_url ?? provenance?.slug ?? extension.source}
                    </code>
                  ),
                },
                {
                  term: "Checksum",
                  value: provenance?.checksum_sha256 ? (
                    <MonoId value={provenance.checksum_sha256} />
                  ) : (
                    <code className="font-mono text-xs text-fg">—</code>
                  ),
                },
                {
                  term: "Verified",
                  value: (
                    <>
                      <Pill
                        mono
                        size="xs"
                        tone={provenance?.checksum_verified ? "success" : "warning"}
                      >
                        {provenance?.checksum_verified ? "verified" : "unverified"}
                      </Pill>
                      <VerifiedMark verified={Boolean(provenance?.checksum_verified)} />
                    </>
                  ),
                },
                {
                  term: "Registry tier",
                  value: (
                    <code className="font-mono text-xs text-fg">
                      {provenance?.registry_tier ?? "—"}
                    </code>
                  ),
                },
              ]}
            />
            <RailBlock
              label="Trust"
              rows={[
                {
                  term: "Decision",
                  value: (
                    <code className="font-mono text-xs text-fg">
                      {extension.trust?.decision ?? "—"}
                    </code>
                  ),
                },
                {
                  term: "Unverified allowed",
                  value: (
                    <code className="font-mono text-xs text-fg">
                      {extension.trust?.allow_unverified ? "yes" : "no"}
                    </code>
                  ),
                },
              ]}
            />
            <BundlesProvided
              active={bundles.data}
              error={bundles.error}
              extensionName={extension.name}
              isLoading={bundles.isLoading}
              onRetry={() => void bundles.refetch()}
              provided={extension.bundles ?? []}
            />
          </aside>
        </div>
      </div>
      <ExtensionProvenanceDialog
        extension={extension}
        onOpenChange={setProvenanceOpen}
        open={provenanceOpen}
      />
      <RemoveExtensionDialog
        activeBundles={bundles.data}
        dependencyError={bundles.error}
        dependencyLoading={bundles.isLoading}
        extension={extension}
        onOpenChange={setRemoveOpen}
        onRemoved={() => void navigate({ to: "/extensions" })}
        onRetryDependencies={() => void bundles.refetch()}
        open={removeOpen}
      />
    </div>
  );
}
