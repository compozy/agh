"use client";

import { Eyebrow } from "@agh/ui";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const detail = error.digest ? `Digest ${error.digest}` : "Runtime boundary failure";

  return (
    <main id="main-content" className="flex min-h-[calc(100dvh-3.5rem)] items-center px-4 py-20">
      <section className="mx-auto w-full max-w-[760px] rounded-(--radius-diagram) border border-(--color-divider) bg-(--color-surface) p-8 md:p-10">
        <Eyebrow case="upper" tone="muted" weight="semibold" className="text-(--color-danger)">
          Render failure
        </Eyebrow>
        <h1 className="mt-5 max-w-[13ch] text-site-error-title leading-none font-normal tracking-tight text-(--color-text-primary)">
          The site hit a recoverable boundary.
        </h1>
        <p className="mt-5 max-w-[58ch] text-base leading-7 text-(--color-text-secondary)">
          The page failed while rendering. Retry the boundary; if it repeats, capture the detail
          below with the current route.
        </p>
        <p className="mt-4 rounded-lg border border-(--color-divider) bg-(--color-canvas-deep) px-4 py-3 font-mono text-xs text-(--color-text-tertiary)">
          {detail}
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-8 inline-flex h-10 items-center justify-center rounded-lg bg-accent px-5 text-sm font-medium text-white transition-colors hover:bg-(--color-accent-hover) active:translate-y-px"
        >
          Retry boundary
        </button>
      </section>
    </main>
  );
}
