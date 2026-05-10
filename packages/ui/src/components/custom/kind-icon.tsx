import * as React from "react";
import { Bot, BrainCircuit, Code, Sparkles, Terminal, type LucideIcon } from "lucide-react";

import { cn } from "../../lib/utils";
import {
  BlackboxLogo,
  ClaudeLogo,
  ClineLogo,
  GeminiLogo,
  GooseLogo,
  HermesLogo,
  JunieLogo,
  KimiLogo,
  OpenAILogo,
  OpenClawLogo,
  OpenHandsLogo,
  QoderLogo,
  QwenLogo,
} from "../../logos";

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
  default: "text-(--fg)",
  muted: "text-(--subtle)",
  accent: "text-(--accent)",
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

const providerKindIconRegistry = {
  blackbox: { brand: BlackboxLogo, fallback: Bot },
  claude: { brand: ClaudeLogo, fallback: BrainCircuit },
  cline: { brand: ClineLogo, fallback: Code },
  codex: { render: props => <OpenAIKindLogo {...props} />, fallback: Code },
  gemini: { brand: GeminiLogo, fallback: Sparkles },
  goose: { brand: GooseLogo, fallback: Terminal },
  hermes: { brand: HermesLogo, fallback: BrainCircuit },
  junie: { brand: JunieLogo, fallback: Sparkles },
  "kimi-cli": { brand: KimiLogo, fallback: Terminal },
  ollama: Terminal,
  openai: { render: props => <OpenAIKindLogo {...props} />, fallback: Bot },
  openclaw: { brand: OpenClawLogo, fallback: Bot },
  openhands: { brand: OpenHandsLogo, fallback: Code },
  qoder: { brand: QoderLogo, fallback: Code },
  "qwen-code": { brand: QwenLogo, fallback: Sparkles },
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

export { KindIcon, providerKindIconRegistry };
export type { KindIconProps, KindIconRegistry, KindIconRegistryEntry, KindIconSize, KindIconTone };
