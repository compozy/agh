import * as React from "react";
import {
  Bot,
  BrainCircuit,
  Code,
  GitBranch,
  MessageSquare,
  Send,
  Sparkles,
  SquareKanban,
  Terminal,
  Users,
  type LucideIcon,
} from "lucide-react";

import { cn } from "../../lib/utils";
import { BlackboxLogo } from "../../logos/blackbox";
import { ClaudeLogo } from "../../logos/claude";
import { ClineLogo } from "../../logos/cline";
import { CursorLogo } from "../../logos/cursor";
import { DiscordLogo } from "../../logos/discord";
import { GeminiLogo } from "../../logos/gemini";
import { GithubLogo } from "../../logos/github";
import { GooseLogo } from "../../logos/goose";
import { GoogleChatLogo } from "../../logos/google-chat";
import { GroqLogo } from "../../logos/groq";
import { HermesLogo } from "../../logos/hermes";
import { JunieLogo } from "../../logos/junie";
import { KimiLogo } from "../../logos/kimi";
import { KiroLogo } from "../../logos/kiro";
import { LinearLogo } from "../../logos/linear";
import { MicrosoftTeamsLogo } from "../../logos/microsoft-teams";
import { MinimaxLogo } from "../../logos/minimax";
import { MistralLogo } from "../../logos/mistral";
import { OpenAILogo } from "../../logos/openai";
import { OpenClawLogo } from "../../logos/openclaw";
import { OpenCodeLogo } from "../../logos/opencode";
import { OpenHandsLogo } from "../../logos/openhands";
import { OpenRouterLogo } from "../../logos/openrouter";
import { PiLogo } from "../../logos/pi";
import { QoderLogo } from "../../logos/qoder";
import { QwenLogo } from "../../logos/qwen";
import { SlackLogo } from "../../logos/slack";
import { TelegramLogo } from "../../logos/telegram";
import { WhatsAppLogo } from "../../logos/whatsapp";
import { XAILogo } from "../../logos/xai";
import { ZAILogo } from "../../logos/zai";

type KindIconTone = "default" | "muted" | "accent";
type KindIconSize = "xs" | "sm" | "md";
type KindIconGlyphProps = React.SVGProps<SVGSVGElement>;
type KindIconRenderer = (props: KindIconGlyphProps) => React.ReactNode;
type DataAttributes = {
  [key: `data-${string}`]: string | undefined;
};

type KindIconRegistryEntry =
  | LucideIcon
  | {
      brand?: React.ComponentType<KindIconGlyphProps>;
      fallback?: LucideIcon;
      render?: KindIconRenderer;
    };

type KindIconRegistry<K extends string = string> = Record<K, KindIconRegistryEntry>;

interface KindIconProps<K extends string = string>
  extends Omit<React.ComponentProps<"span">, "children">, DataAttributes {
  fallback?: LucideIcon;
  kind: K | (string & {});
  registry?: KindIconRegistry<K>;
  size?: KindIconSize;
  tone?: KindIconTone;
}

const KIND_ICON_TONE: Record<KindIconTone, string> = {
  default: "text-fg",
  muted: "text-subtle",
  accent: "text-accent",
};

const KIND_ICON_SIZE: Record<KindIconSize, string> = {
  xs: "size-3",
  sm: "size-4",
  md: "size-5",
};

const KIND_ICON_GLYPH_CLASS = "size-full shrink-0";

function OpenAIKindLogo(props: KindIconGlyphProps) {
  return <OpenAILogo {...props} mode="dark" />;
}

function LinearKindLogo(props: KindIconGlyphProps) {
  return <LinearLogo {...props} mode="dark" />;
}

const providerKindIconRegistry = {
  blackbox: { brand: BlackboxLogo, fallback: Bot },
  claude: { brand: ClaudeLogo, fallback: BrainCircuit },
  cline: { brand: ClineLogo, fallback: Code },
  codex: { render: props => <OpenAIKindLogo {...props} />, fallback: Code },
  cursor: { brand: CursorLogo, fallback: Code },
  gemini: { brand: GeminiLogo, fallback: Sparkles },
  goose: { brand: GooseLogo, fallback: Terminal },
  groq: { brand: GroqLogo, fallback: Sparkles },
  hermes: { brand: HermesLogo, fallback: BrainCircuit },
  junie: { brand: JunieLogo, fallback: Sparkles },
  "kimi-cli": { brand: KimiLogo, fallback: Terminal },
  kiro: { brand: KiroLogo, fallback: Terminal },
  minimax: { brand: MinimaxLogo, fallback: Sparkles },
  mistral: { brand: MistralLogo, fallback: Sparkles },
  moonshot: { brand: KimiLogo, fallback: Sparkles },
  ollama: Terminal,
  openai: { render: props => <OpenAIKindLogo {...props} />, fallback: Bot },
  openclaw: { brand: OpenClawLogo, fallback: Bot },
  opencode: { brand: OpenCodeLogo, fallback: Terminal },
  openhands: { brand: OpenHandsLogo, fallback: Code },
  openrouter: { brand: OpenRouterLogo, fallback: Sparkles },
  pi: { brand: PiLogo, fallback: BrainCircuit },
  qoder: { brand: QoderLogo, fallback: Code },
  "qwen-code": { brand: QwenLogo, fallback: Sparkles },
  xai: { brand: XAILogo, fallback: Sparkles },
  zai: { brand: ZAILogo, fallback: Sparkles },
} satisfies KindIconRegistry;

const bridgeKindIconRegistry = {
  discord: { brand: DiscordLogo, fallback: MessageSquare },
  github: { brand: GithubLogo, fallback: GitBranch },
  "google-chat": { brand: GoogleChatLogo, fallback: MessageSquare },
  google_chat: { brand: GoogleChatLogo, fallback: MessageSquare },
  linear: { render: props => <LinearKindLogo {...props} />, fallback: SquareKanban },
  "microsoft-teams": { brand: MicrosoftTeamsLogo, fallback: Users },
  microsoft_teams: { brand: MicrosoftTeamsLogo, fallback: Users },
  slack: { brand: SlackLogo, fallback: MessageSquare },
  telegram: { brand: TelegramLogo, fallback: Send },
  whatsapp: { brand: WhatsAppLogo, fallback: MessageSquare },
} satisfies KindIconRegistry;

function normalizeKind(kind: string): string {
  return kind.trim().toLowerCase();
}

interface KindIconGlyphPropsForEntry {
  className: string;
  entry: KindIconRegistryEntry | undefined;
  fallback: LucideIcon;
}

function KindIconGlyph({ className, entry, fallback }: KindIconGlyphPropsForEntry) {
  if (typeof entry === "function") {
    const Icon = entry;
    return <Icon aria-hidden="true" className={className} />;
  }

  if (entry?.render) {
    return entry.render({ "aria-hidden": true, className });
  }

  if (entry?.brand) {
    const Brand = entry.brand;
    return <Brand aria-hidden="true" className={className} />;
  }

  const Icon = entry?.fallback ?? fallback;
  return <Icon aria-hidden="true" className={className} />;
}

function KindIcon<K extends string = string>({
  className,
  fallback = Bot,
  kind,
  registry = providerKindIconRegistry as KindIconRegistry<K>,
  size = "sm",
  tone = "muted",
  "data-slot": dataSlot = "kind-icon",
  ...props
}: KindIconProps<K>) {
  const key = normalizeKind(String(kind));
  const entry = registry[key as K];
  return (
    <span
      data-slot={dataSlot}
      data-kind={key}
      className={cn(
        "inline-flex shrink-0 items-center justify-center",
        KIND_ICON_SIZE[size],
        KIND_ICON_TONE[tone],
        className
      )}
      {...props}
    >
      <KindIconGlyph className={KIND_ICON_GLYPH_CLASS} entry={entry} fallback={fallback} />
    </span>
  );
}

export { KindIcon, bridgeKindIconRegistry, providerKindIconRegistry };
export type { KindIconProps, KindIconRegistry, KindIconRegistryEntry, KindIconSize, KindIconTone };
