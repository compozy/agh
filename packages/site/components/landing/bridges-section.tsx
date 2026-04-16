import { ArrowRight } from "lucide-react";
import type { ReactNode } from "react";
import {
  DiscordLogo,
  GithubLogo,
  GoogleChatLogo,
  LinearLogo,
  MicrosoftTeamsLogo,
  SlackLogo,
  TelegramLogo,
  WhatsAppLogo,
} from "../logos";
import { MonoBadge } from "./primitives/mono-badge";
import { SectionFrame } from "./primitives/section-frame";
import { SectionHeader } from "./primitives/section-header";

type Bridge = {
  id: string;
  name: string;
  logo: ReactNode;
  status: "live" | "next";
};

const BRIDGES: Bridge[] = [
  { id: "slack", name: "Slack", logo: <SlackLogo className="h-7 w-7" />, status: "live" },
  {
    id: "discord",
    name: "Discord",
    logo: <DiscordLogo className="h-7 w-7" />,
    status: "live",
  },
  {
    id: "telegram",
    name: "Telegram",
    logo: <TelegramLogo className="h-7 w-7" />,
    status: "live",
  },
  {
    id: "whatsapp",
    name: "WhatsApp",
    logo: <WhatsAppLogo className="h-7 w-7" />,
    status: "next",
  },
  {
    id: "teams",
    name: "Microsoft Teams",
    logo: <MicrosoftTeamsLogo className="h-7 w-7" />,
    status: "next",
  },
  {
    id: "google-chat",
    name: "Google Chat",
    logo: <GoogleChatLogo className="h-7 w-7" />,
    status: "next",
  },
  {
    id: "github",
    name: "GitHub",
    logo: <GithubLogo className="h-7 w-7 text-(--color-text-primary)" />,
    status: "next",
  },
  {
    id: "linear",
    name: "Linear",
    logo: <LinearLogo className="h-7 w-7" mode="dark" />,
    status: "next",
  },
];

export function BridgesSection() {
  return (
    <SectionFrame background="surface" padY="lg">
      <SectionHeader
        align="start"
        eyebrow="Bridges"
        title="Your users live on these. Now so do your agents."
        description="Webhooks in, sessions out. Responses stream back to the original thread. No serverless glue, no second runtime — the bridge adapter runs inside the daemon."
      />

      <ul className="mt-12 grid grid-cols-2 gap-3 sm:grid-cols-4">
        {BRIDGES.map(bridge => (
          <li key={bridge.id}>
            <article className="group relative flex h-full flex-col items-start gap-3 rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-canvas) p-5 transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_35%,var(--color-divider))]">
              <div className="flex items-center justify-between self-stretch">
                <div className="flex h-10 w-10 items-center justify-center">{bridge.logo}</div>
                {bridge.status === "live" ? (
                  <MonoBadge tone="success">live</MonoBadge>
                ) : (
                  <MonoBadge tone="neutral">next</MonoBadge>
                )}
              </div>
              <p className="text-[14px] font-medium text-(--color-text-primary)">{bridge.name}</p>
              <p className="font-mono text-[10px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
                bridge:{bridge.id}
              </p>
            </article>
          </li>
        ))}
      </ul>

      {/* Flow strip */}
      <div className="mt-10 rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-canvas) p-6">
        <div className="flex items-center justify-between border-b border-(--color-divider) pb-4">
          <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
            How a bridge delivers a session
          </p>
          <MonoBadge tone="accent">inside the daemon</MonoBadge>
        </div>
        <div className="mt-6 grid grid-cols-1 items-center gap-4 md:grid-cols-[auto_1fr_auto_1fr_auto_1fr_auto]">
          <FlowNode title="Platform" sub="slack / discord / tg" />
          <FlowArrow label="webhook" />
          <FlowNode title="agh daemon" sub="verify · route" highlight />
          <FlowArrow label="session" />
          <FlowNode title="Agent" sub="claude / codex / …" />
          <FlowArrow label="stream" />
          <FlowNode title="Thread reply" sub="live updates" />
        </div>
      </div>

      <p className="mt-6 max-w-[64ch] text-[13px] leading-relaxed text-(--color-text-tertiary)">
        Every bridge is a workspace-scoped adapter. One platform message maps to one durable
        session, so a user thread keeps its context across restarts.
      </p>
    </SectionFrame>
  );
}

function FlowNode({
  title,
  sub,
  highlight = false,
}: {
  title: string;
  sub: string;
  highlight?: boolean;
}) {
  return (
    <div
      className={`rounded-[8px] border px-3 py-2 text-center ${
        highlight
          ? "border-(--color-accent) bg-(--color-accent-tint)"
          : "border-(--color-divider) bg-(--color-surface)"
      }`}
    >
      <p
        className={`text-[12px] font-medium ${highlight ? "text-(--color-accent)" : "text-(--color-text-primary)"}`}
      >
        {title}
      </p>
      <p className="font-mono text-[10px] tracking-(--tracking-mono) text-(--color-text-tertiary)">
        {sub}
      </p>
    </div>
  );
}

function FlowArrow({ label }: { label: string }) {
  return (
    <div className="flex flex-col items-center justify-center">
      <span className="font-mono text-[9px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
        {label}
      </span>
      <ArrowRight className="h-4 w-4 text-(--color-accent)" />
    </div>
  );
}
