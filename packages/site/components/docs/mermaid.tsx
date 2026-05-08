"use client";

import { useEffect, useId, useState } from "react";

let mermaidLoader: Promise<typeof import("mermaid").default> | null = null;

function loadMermaid() {
  if (!mermaidLoader) {
    mermaidLoader = import("mermaid")
      .then(({ default: mermaid }) => {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          theme: "base",
          themeVariables: {
            background: "#0E0E0F",
            primaryColor: "#1E1C1B",
            primaryBorderColor: "#E8572A",
            primaryTextColor: "#E5E5E7",
            secondaryColor: "#2E2C2B",
            tertiaryColor: "#17110F",
            lineColor: "#8E8E93",
            textColor: "#E5E5E7",
            mainBkg: "#1E1C1B",
            nodeBorder: "#E8572A",
            clusterBkg: "#17110F",
            clusterBorder: "#3C3A39",
            edgeLabelBackground: "#17110F",
            actorBkg: "#1E1C1B",
            actorBorder: "#E8572A",
            actorTextColor: "#E5E5E7",
            noteBkgColor: "#1E1C1B",
            noteBorderColor: "#3C3A39",
            noteTextColor: "#8E8E93",
            fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif",
          },
        });
        return mermaid;
      })
      .catch(error => {
        mermaidLoader = null;
        throw error;
      });
  }

  return mermaidLoader;
}

export function Mermaid({ chart, caption }: { chart: string; caption?: string }) {
  const reactId = useId();
  const diagramId = `agh-mermaid-${reactId.replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const [svg, setSVG] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    setSVG("");
    setError("");

    void loadMermaid()
      .then(async mermaid => {
        const rendered = await mermaid.render(diagramId, chart);
        if (!active) return;
        setSVG(
          rendered.svg.replace(
            "<svg ",
            '<svg aria-hidden="true" class="agh-mermaid-svg" data-theme="agh" '
          )
        );
      })
      .catch(err => {
        if (!active) return;
        setError(err instanceof Error ? err.message : String(err));
      });

    return () => {
      active = false;
    };
  }, [chart, diagramId]);

  return (
    <figure className="not-prose my-6 overflow-x-auto rounded-lg border border-(--color-divider) bg-(--color-surface) p-4">
      {svg ? (
        <div
          aria-label={caption ? `Diagram: ${caption}` : "Mermaid diagram"}
          className="agh-mermaid [&_svg]:h-auto [&_svg]:max-w-full"
          role="img"
          dangerouslySetInnerHTML={{ __html: svg }}
        />
      ) : error ? (
        <div>
          <p className="font-mono text-xs font-semibold uppercase tracking-mono text-accent">
            Diagram source
          </p>
          <p className="mt-2 text-sm leading-6 text-(--color-text-secondary)">
            Mermaid could not render this diagram in the current browser session.
          </p>
          <pre className="mt-4 overflow-x-auto rounded-md border border-(--color-divider) bg-(--color-canvas-deep) p-3 text-xs leading-6 text-(--color-text-secondary)">
            <code>{chart}</code>
          </pre>
          <p className="mt-3 text-sm leading-6 text-(--color-text-tertiary)">{error}</p>
        </div>
      ) : (
        <p className="text-sm leading-6 text-(--color-text-secondary)">Rendering diagram...</p>
      )}

      {caption ? (
        <figcaption className="mt-3 text-sm leading-6 text-(--color-text-secondary)">
          {caption}
        </figcaption>
      ) : null}
    </figure>
  );
}
