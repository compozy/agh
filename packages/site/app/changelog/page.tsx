import Link from "next/link";
import { Rss } from "lucide-react";
import { ChangelogTocRail } from "@/components/blog/changelog-toc-rail";
import { MonoEyebrow } from "@/components/blog/mono-eyebrow";
import { ReleaseEntry } from "@/components/blog/release-entry";
import { allReleases } from "@/lib/blog";
import { changelogMetadata } from "./metadata";

export const metadata = changelogMetadata;

export default function ChangelogPage() {
  const releases = allReleases();

  return (
    <>
      <section className="border-b border-(--color-divider) px-4 pt-14 pb-12">
        <div className="mx-auto max-w-(--site-layout-width)">
          <div className="flex items-center gap-3">
            <MonoEyebrow tone="accent">CHANGELOG</MonoEyebrow>
            <span className="inline-block h-px w-9 bg-(--color-divider)" />
            <MonoEyebrow>Release receipts</MonoEyebrow>
          </div>
          <h1 className="mt-6 max-w-[24ch] font-display text-site-page-title font-normal leading-none tracking-tight text-(--color-text-primary)">
            Every alpha, on the wire.
          </h1>
          <p className="mt-5 max-w-[58ch] text-lg leading-7 text-(--color-text-secondary)">
            AGH ships in the open and logs every change here. New behavior, breaking moves, and
            engineering notes are sourced from the same git history that ships the binary.
          </p>
          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link
              href="/blog/feed.xml"
              className="inline-flex h-8 items-center gap-1.5 rounded-full border border-(--color-divider) px-3.5 font-sans text-small-body text-(--color-text-secondary) hover:text-(--color-text-primary)"
            >
              <Rss size={12} aria-hidden />
              <span className="font-mono text-eyebrow uppercase tracking-mono">RSS</span>
            </Link>
          </div>
        </div>
      </section>

      <section className="px-4 pt-6 pb-20">
        <div className="mx-auto grid max-w-(--site-layout-width) gap-12 lg:grid-cols-[minmax(0,1fr)_280px]">
          <div>
            {releases.length === 0 ? (
              <section className="mt-12 rounded-xl border border-(--color-divider) bg-(--color-surface) p-6">
                <MonoEyebrow tone="accent">Release notes pending</MonoEyebrow>
                <h2 className="mt-4 max-w-[24ch] font-sans text-site-empty-title font-semibold leading-tight tracking-tight text-(--color-text-primary)">
                  Follow the alpha while the first changelog entries are prepared.
                </h2>
                <p className="mt-4 max-w-[62ch] text-sm leading-7 text-(--color-text-secondary)">
                  Published entries will appear here once tagged release notes land in the content
                  layer. Until then, use the install guide for the current runtime path, read the
                  launch post for product context, or subscribe to the RSS feed for new release
                  notes.
                </p>
                <div className="mt-6 flex flex-wrap gap-3">
                  <Link
                    href="/runtime/core/getting-started/installation"
                    className="inline-flex h-9 items-center justify-center rounded-lg border border-(--color-divider) px-3.5 font-sans text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
                  >
                    Install the runtime
                  </Link>
                  <Link
                    href="/blog/introducing-agh-the-first-agent-network-protocol"
                    className="inline-flex h-9 items-center justify-center rounded-lg border border-(--color-divider) px-3.5 font-sans text-sm font-medium text-(--color-text-primary) transition-colors hover:bg-(--color-hover)"
                  >
                    Read the launch post
                  </Link>
                </div>
              </section>
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
