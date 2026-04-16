import { Activity, Database, ShieldCheck, Terminal } from "lucide-react";
import { CodeBlock } from "./primitives/code-block";
import { FeatureCard } from "./primitives/feature-card";
import { SectionFrame } from "./primitives/section-frame";
import { RuntimeMicroDiagram } from "./runtime-micro-diagram";

const FEATURES = [
  {
    icon: <Database className="h-4 w-4" />,
    eyebrow: "Sessions",
    title: "Durable sessions in SQLite",
    description:
      "Every session gets a per-session event DB plus an entry in the global catalog. Resume after a restart, re-read the full history, or fork a new session from any point.",
    cite: {
      href: "/runtime/core/overview/what-is-agh",
      label: "sessions lifecycle",
    },
  },
  {
    icon: <Activity className="h-4 w-4" />,
    eyebrow: "Events",
    title: "Replayable event stream",
    description:
      "Every prompt, tool call, permission decision, and agent message is persisted with a monotonic sequence. SSE replay at /api/sessions/:id/stream.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "event catalog" },
  },
  {
    icon: <Terminal className="h-4 w-4" />,
    eyebrow: "Surfaces",
    title: "Three operator surfaces, one daemon",
    description:
      "CLI over a Unix socket. HTTP + SSE API on :2123. A React 19 web UI with ten feature modules. All read from the same state.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "daemon surfaces" },
  },
  {
    icon: <ShieldCheck className="h-4 w-4" />,
    eyebrow: "Permissions",
    title: "Permissioned tools per agent",
    description:
      "Allowlists, workspace scoping, and MCP server wiring live in config.toml. The audit log records every decision.",
    cite: { href: "/runtime/core/overview/what-is-agh", label: "permissions" },
  },
];

const RUNTIME_CODE = `agh daemon start
agh session new --agent coder --provider claude
agh session events <session-id> --follow
agh session resume <session-id>`;

export function RuntimeSection() {
  return (
    <SectionFrame background="canvas" padY="lg">
      <div className="grid gap-12 lg:grid-cols-[minmax(0,260px)_1fr] lg:items-start lg:gap-16">
        <div className="h-full flex flex-col justify-between lg:sticky lg:top-24">
          <div>
            <p className="font-mono text-[11px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
              Runtime
            </p>
            <h2 className="mt-3 text-[clamp(1.9rem,3.4vw,2.6rem)] leading-[1.05] font-normal tracking-[-0.025em] text-(--color-text-primary)">
              A daemon built for sessions, not chats.
            </h2>
            <p className="mt-4 max-w-[50ch] text-sm leading-relaxed text-(--color-text-secondary)">
              Start <code className="font-mono text-(--color-text-primary)">agh daemon</code>. Every
              agent run becomes a session with a durable event log, an SSE stream, resumable state,
              and a web UI — for humans and automation alike.
            </p>
          </div>
          <div className="hidden lg:block">
            <RuntimeMicroDiagram />
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <div className="grid gap-4 sm:grid-cols-2">
            {FEATURES.map(f => (
              <FeatureCard
                key={f.eyebrow}
                icon={f.icon}
                eyebrow={f.eyebrow}
                title={f.title}
                description={f.description}
                cite={f.cite}
              />
            ))}
          </div>

          <CodeBlock code={RUNTIME_CODE} caption="agh session" shell />
        </div>
      </div>
    </SectionFrame>
  );
}
