import Link from "next/link";
import type { ReactNode } from "react";
import { ArrowUpRight } from "lucide-react";
import {
  ClaudeLogo,
  CursorLogo,
  GeminiLogo,
  GithubLogo,
  KiroLogo,
  OpenAILogo,
  OpenCodeLogo,
  PiLogo,
} from "@agh/ui/logos";
import { SectionFrame } from "./primitives/section-frame";

type Provider = { id: string; name: string; logo: ReactNode };

const PROVIDERS: Provider[] = [
  { id: "claude", name: "Claude Code", logo: <ClaudeLogo className="h-6 w-6" /> },
  {
    id: "codex",
    name: "Codex",
    logo: <OpenAILogo className="h-6 w-6" mode="dark" />,
  },
  { id: "gemini", name: "Gemini CLI", logo: <GeminiLogo className="h-6 w-6" /> },
  {
    id: "opencode",
    name: "OpenCode",
    logo: <OpenCodeLogo className="h-6 w-6 text-(--color-text-primary)" />,
  },
  {
    id: "copilot",
    name: "Copilot CLI",
    logo: <GithubLogo className="h-6 w-6 text-(--color-text-primary)" />,
  },
  {
    id: "cursor",
    name: "Cursor",
    logo: <CursorLogo className="h-6 w-6 text-(--color-text-primary)" />,
  },
  { id: "kiro", name: "Kiro CLI", logo: <KiroLogo className="h-6 w-6" /> },
  { id: "pi", name: "Pi", logo: <PiLogo className="h-6 w-6" /> },
];

/**
 * Compact strip showing which agent CLIs work today. Kept intentionally
 * low-key: the runtime + network are the primary story, the agent list is
 * reassurance that whatever CLI you use, it plugs in.
 */
export function SupportedAgents() {
  return (
    <SectionFrame background="canvas" padY="md">
      <div className="flex flex-col gap-8 lg:flex-row lg:items-center lg:justify-between lg:gap-12">
        <div className="max-w-[38ch]">
          <p className="font-mono text-[10px] font-semibold uppercase tracking-(--tracking-mono) text-(--color-accent)">
            Works with your agent CLIs
          </p>
          <p className="mt-2 text-[1rem] leading-snug text-(--color-text-primary)">
            Bring the CLI you already use. AGH spawns it, manages it, and persists every event.
          </p>
          <Link
            href="/runtime"
            className="mt-3 inline-flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-(--tracking-mono) text-(--color-text-secondary) transition-colors hover:text-(--color-accent)"
          >
            Configure providers
            <ArrowUpRight className="h-3 w-3" />
          </Link>
        </div>

        <ul className="grid w-full grid-cols-4 gap-2 sm:grid-cols-8 lg:w-auto lg:grid-cols-8">
          {PROVIDERS.map(provider => (
            <li key={provider.id}>
              <div
                className="flex h-16 w-full min-w-[76px] flex-col items-center justify-center gap-1 rounded-[10px] border border-(--color-divider) bg-(--color-surface) px-2 transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_35%,var(--color-divider))]"
                title={provider.name}
              >
                <span aria-hidden="true" className="flex h-6 w-6 items-center justify-center">
                  {provider.logo}
                </span>
                <span className="font-mono text-[9px] uppercase tracking-(--tracking-mono) text-(--color-text-tertiary)">
                  {provider.id}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </SectionFrame>
  );
}
