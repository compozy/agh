import type { Metadata } from "next";
import Link from "next/link";
import { Rss } from "lucide-react";
import { ChangelogTocRail } from "@/components/blog/changelog-toc-rail";
import { MonoEyebrow } from "@/components/blog/mono-eyebrow";
import { ReleaseEntry } from "@/components/blog/release-entry";
import { allReleases } from "@/lib/blog";
import { createPageMetadata, siteConfig } from "@/lib/site-config";

export const metadata: Metadata = createPageMetadata({
  title: "Changelog",
  description: "Every alpha receipt and release note for the AGH runtime and agh-network/v0.",
  path: "/changelog",
});

export default function ChangelogPage() {
  const releases = allReleases();
  const githubReleases = `${siteConfig.githubUrl}/releases`;

  return (
    <>
      <section className="border-b border-(--color-divider) px-4 pt-14 pb-12">
        <div className="mx-auto max-w-(--site-layout-width)">
          <div className="flex items-center gap-3">
            <MonoEyebrow tone="accent">CHANGELOG</MonoEyebrow>
            <span className="inline-block h-px w-9 bg-(--color-divider)" />
            <MonoEyebrow>Release receipts</MonoEyebrow>
          </div>
          <h1 className="mt-6 max-w-[24ch] font-display text-[clamp(2.4rem,4.4vw,4rem)] font-normal leading-[1] tracking-[-0.035em] text-(--color-text-primary)">
            Every alpha, on the wire.
          </h1>
          <p className="mt-5 max-w-[58ch] text-lg leading-[1.6] text-(--color-text-secondary)">
            We ship in the open and log every change here. New behavior, breaking moves, and
            engineering notes — sourced from the same git history that ships the binary.
          </p>
          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link
              href="/blog/feed.xml"
              className="inline-flex h-8 items-center gap-1.5 rounded-full border border-(--color-divider) px-3.5 font-sans text-[13px] text-(--color-text-secondary) hover:text-(--color-text-primary)"
            >
              <Rss size={12} aria-hidden />
              <span className="font-mono text-[11px] uppercase tracking-[0.06em]">RSS</span>
            </Link>
            <Link
              href={githubReleases}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex h-8 items-center gap-1.5 rounded-full border border-(--color-divider) px-3.5 font-sans text-[13px] text-(--color-text-secondary) hover:text-(--color-text-primary)"
            >
              <span>Watch on GitHub</span>
            </Link>
          </div>
        </div>
      </section>

      <section className="px-4 pt-6 pb-20">
        <div className="mx-auto grid max-w-(--site-layout-width) gap-12 lg:grid-cols-[minmax(0,1fr)_280px]">
          <div>
            {releases.length === 0 ? (
              <p className="mt-12 rounded-xl border border-(--color-divider) bg-(--color-surface) p-6 text-sm text-(--color-text-secondary)">
                Releases appear here as they ship.
              </p>
            ) : (
              releases.map(release => <ReleaseEntry key={release.version} release={release} />)
            )}
          </div>
          <ChangelogTocRail releases={releases} />
        </div>
      </section>
    </>
  );
}
