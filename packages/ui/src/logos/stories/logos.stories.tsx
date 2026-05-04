import type { ComponentType, SVGProps } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  BlackboxLogo,
  ClaudeLogo,
  ClineLogo,
  CursorLogo,
  DiscordLogo,
  GeminiLogo,
  GithubLogo,
  GoogleChatLogo,
  GooseLogo,
  GroqLogo,
  HermesLogo,
  JunieLogo,
  KimiLogo,
  KiroLogo,
  LinearLogo,
  MicrosoftTeamsLogo,
  MinimaxLogo,
  MistralLogo,
  OpenAILogo,
  OpenClawLogo,
  OpenCodeLogo,
  OpenHandsLogo,
  OpenRouterLogo,
  PiLogo,
  QoderLogo,
  QwenLogo,
  SlackLogo,
  TelegramLogo,
  VercelLogo,
  WhatsAppLogo,
  XAILogo,
  ZAILogo,
} from "../index";

type LogoProps = Pick<SVGProps<SVGSVGElement>, "aria-label" | "className">;
type LogoComponent = ComponentType<LogoProps>;
type LogoGroup = "all" | "agents" | "bridges";

interface LogoGalleryProps {
  group?: LogoGroup;
}

const AGENT_LOGOS: Array<{ label: string; Logo: LogoComponent }> = [
  { label: "Blackbox", Logo: BlackboxLogo },
  { label: "Claude", Logo: ClaudeLogo },
  { label: "Cline", Logo: ClineLogo },
  { label: "Cursor", Logo: CursorLogo },
  { label: "Gemini", Logo: GeminiLogo },
  { label: "Goose", Logo: GooseLogo },
  { label: "Groq", Logo: GroqLogo },
  { label: "Hermes", Logo: HermesLogo },
  { label: "Junie", Logo: JunieLogo },
  { label: "Kimi", Logo: KimiLogo },
  { label: "Kiro", Logo: KiroLogo },
  { label: "Minimax", Logo: MinimaxLogo },
  { label: "Mistral", Logo: MistralLogo },
  { label: "OpenAI", Logo: OpenAILogo },
  { label: "OpenClaw", Logo: OpenClawLogo },
  { label: "OpenCode", Logo: OpenCodeLogo },
  { label: "OpenHands", Logo: OpenHandsLogo },
  { label: "OpenRouter", Logo: OpenRouterLogo },
  { label: "Pi", Logo: PiLogo },
  { label: "Qoder", Logo: QoderLogo },
  { label: "Qwen", Logo: QwenLogo },
  { label: "Vercel", Logo: VercelLogo },
  { label: "xAI", Logo: XAILogo },
  { label: "Z.ai", Logo: ZAILogo },
];

const BRIDGE_LOGOS: Array<{ label: string; Logo: LogoComponent }> = [
  { label: "Discord", Logo: DiscordLogo },
  { label: "GitHub", Logo: GithubLogo },
  { label: "Google Chat", Logo: GoogleChatLogo },
  { label: "Linear", Logo: LinearLogo },
  { label: "Microsoft Teams", Logo: MicrosoftTeamsLogo },
  { label: "Slack", Logo: SlackLogo },
  { label: "Telegram", Logo: TelegramLogo },
  { label: "WhatsApp", Logo: WhatsAppLogo },
];

function LogoSection({
  title,
  logos,
}: {
  title: string;
  logos: Array<{ label: string; Logo: LogoComponent }>;
}) {
  return (
    <section className="grid gap-4">
      <h2 className="font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-label)]">
        {title}
      </h2>
      <div className="grid grid-cols-2 gap-px overflow-hidden rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-divider)] sm:grid-cols-3 md:grid-cols-4">
        {logos.map(({ label, Logo }) => (
          <div
            key={label}
            className="grid min-h-28 place-items-center gap-3 bg-[color:var(--color-surface)] p-4 text-center"
          >
            <Logo
              aria-label={`${label} logo`}
              className="size-8 text-[color:var(--color-text-primary)]"
            />
            <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-secondary)]">
              {label}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}

function LogoGallery({ group = "all" }: LogoGalleryProps) {
  const showAgents = group === "all" || group === "agents";
  const showBridges = group === "all" || group === "bridges";

  return (
    <div className="grid w-[min(960px,calc(100vw-2rem))] gap-8 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-6 text-[color:var(--color-text-primary)]">
      <div className="grid gap-2">
        <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-accent)]">
          Logo registry
        </p>
        <h1 className="text-xl font-semibold">Agent and bridge logos</h1>
        <p className="max-w-[62ch] text-sm leading-6 text-[color:var(--color-text-secondary)]">
          Brand SVGs exported by `@agh/ui/logos` for AGH site and runtime surfaces.
        </p>
      </div>
      {showAgents ? <LogoSection title="Agent providers" logos={AGENT_LOGOS} /> : null}
      {showBridges ? <LogoSection title="Bridge surfaces" logos={BRIDGE_LOGOS} /> : null}
    </div>
  );
}

const meta: Meta<typeof LogoGallery> = {
  title: "ui/Logos",
  component: LogoGallery,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Shared brand logo registry for agent providers and bridge surfaces consumed by AGH public pages.",
      },
    },
  },
  argTypes: {
    group: {
      control: "select",
      options: ["all", "agents", "bridges"],
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Full logo registry used by the AGH site.
 */
export const Default: Story = {
  args: {
    group: "all",
  },
};

/**
 * Agent-provider logos used by the supported agents section.
 */
export const AgentProviders: Story = {
  args: {
    group: "agents",
  },
};

/**
 * Bridge logos used by bridge and integration sections.
 */
export const BridgeSurfaces: Story = {
  args: {
    group: "bridges",
  },
};
