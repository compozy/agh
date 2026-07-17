import { Link } from "@tanstack/react-router";
import { MoreHorizontal, Puzzle } from "lucide-react";

import {
  Button,
  DescriptionCard,
  DetailHeader,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  MonoId,
  Pill,
  Section,
  Switch,
  Time,
} from "@agh/ui";

import { formatUptimeSeconds } from "@/lib/format-time";
import {
  BundlesProvided,
  DetailBlock,
  Diagnostics,
  EnvironmentState,
  ExtensionDetailSkeleton,
  RailBlock,
  TokenBlock,
} from "./extension-detail-sections";
import {
  ExtensionProvenanceDialog,
  RemoveExtensionDialog,
  VerifiedMark,
} from "./extension-dialogs";
import { useExtensionDetailState } from "../hooks/use-extension-detail-state";

export function ExtensionDetail({ name }: { name: string }) {
  const state = useExtensionDetailState(name);
  const {
    bundles,
    detail,
    navigate,
    provenanceOpen,
    removeOpen,
    setProvenanceOpen,
    setRemoveOpen,
    toggle,
    update,
  } = state;
  if (detail.isLoading) return <ExtensionDetailSkeleton />;
  if (detail.error)
    return (
      <Empty description={detail.error.message} icon={Puzzle} title="Unable to load extension" />
    );
  if (!detail.data)
    return (
      <Empty
        description={`No installed extension is named ${name}.`}
        icon={Puzzle}
        title="Extension not found"
      />
    );

  const { extension, listing, updateAvailable } = detail.data;
  const provenance = extension.provenance;
  const acting = toggle.isPending || update.isPending;
  const description =
    listing?.description ?? "No catalog description is available for this installed extension.";
  return (
    <div className="min-h-0 flex-1 overflow-y-auto" data-testid="extension-detail">
      <DetailHeader
        actions={
          <>
            <span className="flex items-center gap-2 text-xs text-muted">
              {extension.enabled ? "Enabled" : "Disabled"}
              <Switch
                aria-label={`${extension.enabled ? "Disable" : "Enable"} ${name}`}
                checked={extension.enabled}
                disabled={acting}
                onCheckedChange={enabled => toggle.mutate({ name, enabled })}
                size="sm"
              />
            </span>
            {updateAvailable ? (
              <Button
                disabled={acting}
                onClick={() => update.mutate(name)}
                size="sm"
                variant="outline"
              >
                Update
              </Button>
            ) : null}
            {listing ? (
              <Button
                render={
                  <Link
                    params={{ entryId: listing.entry_id, kind: "extension" }}
                    to="/marketplace/$kind/$entryId"
                  />
                }
                nativeButton={false}
                size="sm"
                variant="ghost"
              >
                View in marketplace →
              </Button>
            ) : null}
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button aria-label={`Actions for ${name}`} size="icon-sm" variant="ghost" />
                }
              >
                <MoreHorizontal className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setProvenanceOpen(true)}>
                  Provenance
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="text-danger" onClick={() => setRemoveOpen(true)}>
                  Remove…
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </>
        }
        crumbs={[
          { id: "extensions", label: "Extensions", to: "/extensions" },
          { id: name, label: name },
        ]}
        leading={
          <span className="grid size-(--size-provider-logo-well) place-items-center rounded-lg bg-elevated text-muted">
            <Puzzle aria-hidden="true" className="size-5" />
          </span>
        }
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
              <Pill mono size="xs" tone={extension.health === "healthy" ? "success" : "warning"}>
                {extension.health}
              </Pill>
            ) : null}
          </>
        }
        title={name}
      />
      <div className="mx-auto grid w-full max-w-[1320px] gap-8 px-9 py-7 lg:grid-cols-[minmax(0,1fr)_20rem]">
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
