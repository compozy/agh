import { Cloud, RefreshCcw, Terminal } from "lucide-react";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

const CARDS = [
  {
    icon: <Terminal className="h-4 w-4" />,
    eyebrow: "Local",
    title: "Run on the host when isolation is not needed",
    description:
      "The local backend keeps the same tool path and filesystem expectations while still recording sandbox metadata on the session.",
  },
  {
    icon: <Cloud className="h-4 w-4" />,
    eyebrow: "Daytona",
    title: "Move a workspace into a remote sandbox",
    description:
      "Daytona profiles create or reattach cloud sandboxes from an image or snapshot, then connect AGH through the provider tool host.",
  },
  {
    icon: <RefreshCcw className="h-4 w-4" />,
    eyebrow: "Sync",
    title: "Control how files move",
    description:
      "Profiles choose sync mode, persistence, runtime root, network policy, and provider-specific lifecycle settings.",
  },
];

function SandboxDiagram() {
  return (
    <div
      className="relative min-h-[360px] overflow-hidden rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-5"
      aria-label="AGH sandbox lifecycle diagram"
      role="img"
    >
      <div className="absolute inset-x-5 top-5 h-px bg-(--color-divider)" />
      <div className="grid h-full min-h-[320px] grid-rows-[auto_1fr_auto] gap-5">
        <div className="grid grid-cols-3 gap-3">
          {["workspace", "agh daemon", "sandbox provider"].map(label => (
            <div
              key={label}
              className="rounded-[8px] border border-(--color-divider) bg-(--color-canvas) px-3 py-2"
            >
              <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-text-label)">
                {label}
              </p>
            </div>
          ))}
        </div>

        <div className="grid grid-cols-[1fr_auto_1fr_auto_1fr] items-center gap-3">
          <DiagramNode title="Host files" lines={["root_dir", "add_dirs", ".agh/config.toml"]} />
          <div className="h-px w-10 bg-(--color-accent)" />
          <DiagramNode title="Session" lines={["prepare", "sync", "stop"]} active />
          <div className="h-px w-10 bg-(--color-accent)" />
          <DiagramNode title="Daytona" lines={["image/snapshot", "runtime_root", "tool host"]} />
        </div>

        <div className="grid grid-cols-3 gap-3">
          {["sandbox_id", "sandbox_ref", "sandbox.exec"].map(label => (
            <code
              key={label}
              className="rounded-[6px] border border-(--color-divider) bg-(--color-canvas-deep) px-3 py-2 text-center font-mono text-[11px] text-(--color-text-secondary)"
            >
              {label}
            </code>
          ))}
        </div>
      </div>
    </div>
  );
}

function DiagramNode({
  title,
  lines,
  active = false,
}: {
  title: string;
  lines: string[];
  active?: boolean;
}) {
  return (
    <div
      className={`rounded-[8px] border p-4 ${
        active
          ? "border-(--color-accent) bg-[color-mix(in_srgb,var(--color-accent)_8%,var(--color-surface))]"
          : "border-(--color-divider) bg-(--color-canvas)"
      }`}
    >
      <p className="text-sm font-medium text-(--color-text-primary)">{title}</p>
      <ul className="mt-3 space-y-1">
        {lines.map(line => (
          <li key={line} className="font-mono text-[11px] text-(--color-text-tertiary)">
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function SandboxSection() {
  return (
    <SectionFrame background="surface" padY="lg" ariaLabel="Sandbox execution">
      <SectionHeader
        align="start"
        eyebrow="Sandbox"
        title="Run agents away from the host filesystem."
        description="Keep a session local when that is enough, or bind a workspace to a Daytona sandbox with explicit sync, lifecycle, and provider metadata."
      />

      <div className="mt-12 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(360px,0.9fr)] lg:items-stretch">
        <div className="grid gap-4">
          {CARDS.map(card => (
            <article
              key={card.eyebrow}
              className="rounded-(--radius-card) border border-(--color-divider) bg-(--color-canvas) p-5"
            >
              <div className="flex items-center gap-3">
                <span className="inline-flex h-8 w-8 items-center justify-center rounded-[6px] border border-(--color-divider) text-(--color-accent)">
                  {card.icon}
                </span>
                <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
                  {card.eyebrow}
                </p>
              </div>
              <h3 className="mt-4 text-[1.0625rem] font-medium leading-snug text-(--color-text-primary)">
                {card.title}
              </h3>
              <p className="mt-3 text-sm leading-relaxed text-(--color-text-secondary)">
                {card.description}
              </p>
            </article>
          ))}
        </div>

        <SandboxDiagram />
      </div>
    </SectionFrame>
  );
}
