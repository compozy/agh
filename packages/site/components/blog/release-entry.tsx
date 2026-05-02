import type { Release } from "#site/content";
import { ArrowUpRight } from "lucide-react";
import { formatDate } from "./format";
import { MonoBadge, type MonoBadgeTone } from "./mono-badge";
import { MonoEyebrow } from "./mono-eyebrow";

export interface ReleaseEntryProps {
  release: Release;
}

const statusTone: Record<Release["status"], MonoBadgeTone> = {
  stable: "success",
  beta: "info",
  alpha: "accent",
  breaking: "danger",
};

const sectionLabel: Record<"added" | "changed" | "fixed" | "breaking", string> = {
  added: "ADDED",
  changed: "CHANGED",
  fixed: "FIXED",
  breaking: "BREAKING",
};

const sectionTone: Record<"added" | "changed" | "fixed" | "breaking", MonoBadgeTone> = {
  added: "success",
  changed: "info",
  fixed: "warning",
  breaking: "danger",
};

export function ReleaseEntry({ release }: ReleaseEntryProps) {
  const sections = (["added", "changed", "fixed", "breaking"] as const).filter(
    key => release[key].length > 0
  );

  return (
    <article
      id={release.version}
      className="grid scroll-mt-24 gap-8 border-t border-(--color-divider) py-12 lg:grid-cols-[160px_minmax(0,1fr)] lg:gap-12"
    >
      <div className="flex flex-col gap-3 lg:sticky lg:top-24 lg:self-start">
        <MonoEyebrow tracking="wide">{formatDate(release.date)}</MonoEyebrow>
        <div className="flex items-center gap-2">
          <MonoBadge tone={statusTone[release.status]}>{release.version}</MonoBadge>
          <MonoBadge tone="neutral">{release.status.toUpperCase()}</MonoBadge>
        </div>
        {release.compareUrl && (
          <a
            href={release.compareUrl}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex items-center gap-1.5 text-xs text-(--color-text-secondary) hover:text-(--color-text-primary)"
          >
            Compare on GitHub <ArrowUpRight size={12} aria-hidden />
          </a>
        )}
      </div>
      <div>
        <h2 className="font-sans text-[clamp(1.6rem,3vw,2.1rem)] font-semibold leading-[1.1] tracking-[-0.025em] text-(--color-text-primary)">
          {release.summary}
        </h2>
        <div className="mt-8 flex flex-col gap-7">
          {sections.map(key => (
            <section key={key}>
              <MonoBadge tone={sectionTone[key]}>{sectionLabel[key]}</MonoBadge>
              <ul className="mt-4 flex flex-col gap-2.5">
                {release[key].map((item, idx) => (
                  <li
                    key={`${key}-${idx}`}
                    className="flex items-start gap-3 font-sans text-[15px] leading-[1.6] text-(--color-text-secondary)"
                  >
                    <span className="mt-2 inline-block h-1 w-1 shrink-0 rounded-[1px] bg-(--color-accent)" />
                    <span>{item}</span>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </div>
    </article>
  );
}
