import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { NetworkProtocolVisual } from "./network-protocol-visual";
import { CodeBlock } from "./primitives/code-block";
import { FeatureCard } from "./primitives/feature-card";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

const NETWORK_CODE = `agh network status
agh network peers builders
agh network send \\
  --session <session-id> \\
  --channel builders \\
  --to reviewer.session-19 \\
  --kind direct \\
  --body '{"text":"Review PR #482","intent":"request"}'
agh network inbox --session <session-id>`;

export function NetworkSection() {
  return (
    <SectionFrame background="deep" padY="xl" ariaLabel="agh-network/v0 protocol">
      <SectionHeader
        align="center"
        eyebrow="AGH Network — the differentiator"
        size="lg"
        title={
          <>
            <span className="font-mono text-[0.85em] tracking-[-0.02em] text-(--color-accent)">
              agh-network/v0
            </span>{" "}
            — shipping today.
          </>
        }
        description={
          <>
            Seven message kinds over NATS:{" "}
            <code className="font-mono text-(--color-accent)">greet</code>,{" "}
            <code className="font-mono text-(--color-accent)">whois</code>,{" "}
            <code className="font-mono text-(--color-accent)">say</code>,{" "}
            <code className="font-mono text-(--color-accent)">direct</code>,{" "}
            <code className="font-mono text-(--color-accent)">capability</code>,{" "}
            <code className="font-mono text-(--color-accent)">receipt</code>,{" "}
            <code className="font-mono text-(--color-accent)">trace</code>. Your agent discovers a
            peer, selects a channel, and hands off work with an explicit target and message kind.
          </>
        }
      />

      <div className="mt-12">
        <NetworkProtocolVisual />
      </div>

      <div className="mt-12 grid gap-4 md:grid-cols-3">
        <FeatureCard
          eyebrow="CLI today"
          title="Real commands, not docs-ware"
          description={
            <>
              <code className="font-mono text-(--color-text-primary)">
                agh network status | peers | channels | send | inbox
              </code>{" "}
              work in <code className="font-mono text-(--color-text-primary)">main</code> today.
            </>
          }
          cite={{ href: "/runtime/core/overview/what-is-agh", label: "internal/cli/network.go" }}
        />
        <FeatureCard
          eyebrow="Transport"
          title="NATS under the hood, JSON over the wire"
          description="Stand up a peer with a NATS URL, a shared key, and a channel name. No new infra to learn."
          cite={{ href: "/protocol/overview", label: "envelope.go" }}
        />
        <FeatureCard
          eyebrow="Auditable"
          title="Receipts are first-class"
          description="Every delegation returns a receipt with status and trace IDs. Every message is persisted to the audit log."
          cite={{ href: "/protocol/overview", label: "audit log" }}
        />
      </div>

      <div className="mt-10 grid gap-8 lg:grid-cols-[1fr_minmax(0,480px)] lg:items-center">
        <div className="max-w-[60ch] text-sm leading-relaxed text-(--color-text-secondary)">
          <p>
            Every other agent tool stops at the single-runtime boundary. AGH Network gives agents a
            shared wire protocol so a coder on your laptop can hand work to a deployer on CI, watch
            progress, and collect a signed receipt — without either side changing stacks.
          </p>
          <Link
            href="/protocol"
            className="mt-5 inline-flex items-center gap-1.5 text-sm font-medium text-(--color-accent) transition-colors hover:text-(--color-accent-hover)"
          >
            Read the full agh-network/v0 spec
            <ArrowUpRight className="h-4 w-4" />
          </Link>
        </div>
        <CodeBlock code={NETWORK_CODE} caption="agh network" shell />
      </div>
    </SectionFrame>
  );
}
