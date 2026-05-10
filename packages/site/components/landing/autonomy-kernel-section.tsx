import Image from "next/image";
import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import { CodeBlock } from "./primitives/code-block";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";
import { Eyebrow } from "@agh/ui";

const AUTONOMY_CODE = `agh task create
agh task list --status queued
agh task next --wait                # claimed by an idle agent
agh task heartbeat <run-id>         # held by claim_token
agh task complete <run-id>`;

export function AutonomyKernelSection() {
  return (
    <SectionFrame
      background="deep"
      padY="lg"
      className="border-b border-(--color-divider)"
      ariaLabel="AGH autonomy kernel"
    >
      <SectionHeader
        align="start"
        eyebrow="Autonomy"
        size="lg"
        title="A real autonomy kernel, not a fork-and-pray loop."
        description="AGH owns the loop. Tasks claim runs atomically through ClaimNextRun, hold a lease they must heartbeat, and release back to the queue if they crash. One queue. Shared between humans and agents. Claim tokens never logged in raw form."
      />

      {/* Wide landscape storyboard — three-act lifecycle: ONE QUEUE → TOKEN-FENCED → LEASE RECOVERY. */}
      <figure className="mt-12">
        <Image
          src="/images/runtime/autonomy-overview-storyboard-v1.png"
          alt="AGH autonomy storyboard, task_runs queue, an agent claiming a run with a claim_token and heartbeat, and lease recovery on daemon restart."
          width={1440}
          height={760}
          decoding="async"
          sizes="100vw"
          unoptimized
          className="block select-none opacity-95"
        />
      </figure>

      {/* Asymmetric 2-col below — large narrative + code, narrow value list. No 3-col card grid. */}
      <div className="mt-12 grid min-w-0 gap-10 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] lg:items-start lg:gap-12">
        <div className="flex min-w-0 flex-col gap-6">
          <div className="min-w-0 rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-5 sm:p-7">
            <Eyebrow
              case="upper"
              tone="muted"
              size="badge"
              weight="semibold"
              className="text-accent"
            >
              Token-fenced ownership
            </Eyebrow>
            <h3 className="mt-3 font-display text-2xl leading-tight tracking-tight text-(--color-text-primary)">
              No double-execution, ever.
            </h3>
            <p className="mt-3 max-w-[60ch] text-sm leading-relaxed text-(--color-text-secondary)">
              Only the agent holding the claim token can heartbeat or complete a run. Sessions
              cannot reach into runs they don&apos;t own. Tokens are hashed before they touch the
              event ledger; raw values never leave the daemon.
            </p>
          </div>
          <CodeBlock code={AUTONOMY_CODE} caption="agh task" shell />
        </div>

        <ul className="flex min-w-0 flex-col divide-y divide-(--color-divider) border-y border-(--color-divider)">
          <li className="py-6">
            <Eyebrow
              case="upper"
              tone="muted"
              size="badge"
              weight="semibold"
              className="text-accent"
            >
              Lease recovery
            </Eyebrow>
            <h4 className="mt-2 text-base font-medium leading-snug text-(--color-text-primary)">
              Daemon crashes don&apos;t orphan work.
            </h4>
            <p className="mt-2 text-sm leading-relaxed text-(--color-text-secondary)">
              Leases expire on a TTL. Runs re-enter the queue automatically. The next idle agent
              picks them up.
            </p>
          </li>
          <li className="py-6">
            <Eyebrow
              case="upper"
              tone="muted"
              size="badge"
              weight="semibold"
              className="text-accent"
            >
              One shared queue
            </Eyebrow>
            <h4 className="mt-2 text-base font-medium leading-snug text-(--color-text-primary)">
              Operators and agents hit task_runs.
            </h4>
            <p className="mt-2 text-sm leading-relaxed text-(--color-text-secondary)">
              <code className="font-mono text-(--color-text-primary)">agh task create</code> (you)
              and the coordinator agent (them) write to the same SQLite table. Same primitives, same
              audit trail.
            </p>
          </li>
          <li className="py-6">
            <Eyebrow
              case="upper"
              tone="muted"
              size="badge"
              weight="semibold"
              className="text-accent"
            >
              Permission narrowing
            </Eyebrow>
            <h4 className="mt-2 text-base font-medium leading-snug text-(--color-text-primary)">
              Children cannot widen parents.
            </h4>
            <p className="mt-2 text-sm leading-relaxed text-(--color-text-secondary)">
              Lineage, TTLs, and permission scopes are part of the spawn contract; enforced in code,
              not in the prompt.
            </p>
          </li>
        </ul>
      </div>

      <div className="mt-10">
        <Link
          href="/runtime"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-accent transition-colors hover:text-(--color-accent-hover)"
        >
          Read the autonomy kernel guide
          <ArrowUpRight aria-hidden className="size-4" />
        </Link>
      </div>
    </SectionFrame>
  );
}
