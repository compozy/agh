import { ImageResponse } from "next/og";
import { loadOGFonts } from "../fonts";
import { SymbolGlyph } from "../logo";
import { COLORS, FONTS, SIZE, truncate } from "../tokens";

export type DocsTree = "runtime" | "protocol";

export interface RenderDocsOGInput {
  tree: DocsTree;
  title: string;
  description?: string;
  path: string;
}

const TREE_LABELS: Record<DocsTree, { eyebrow: string; chip: string }> = {
  runtime: { eyebrow: "AGH RUNTIME", chip: "RUNTIME" },
  protocol: { eyebrow: "AGH NETWORK PROTOCOL", chip: "PROTOCOL" },
};

export async function renderDocsOG({
  tree,
  title,
  description,
  path,
}: RenderDocsOGInput): Promise<ImageResponse> {
  const fonts = await loadOGFonts();
  const labels = TREE_LABELS[tree];
  const safeTitle = truncate(title, 120);
  const safeDescription = truncate(description, 165);

  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        background: COLORS.canvas,
        color: COLORS.textPrimary,
        fontFamily: FONTS.inter,
        padding: "80px",
      }}
    >
      <div
        style={{
          display: "flex",
          flexDirection: "row",
          alignItems: "center",
          paddingBottom: "32px",
          borderBottom: `1px solid ${COLORS.border}`,
          gap: "24px",
        }}
      >
        <SymbolGlyph size={56} radius={14} />
        <div
          style={{
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
            gap: "20px",
            fontFamily: FONTS.mono,
            fontSize: "20px",
            letterSpacing: "0.14em",
            fontWeight: 500,
          }}
        >
          <span style={{ color: COLORS.accent, textTransform: "uppercase" }}>{labels.eyebrow}</span>
          <span style={{ width: "48px", height: "1px", background: COLORS.border }} />
          <span style={{ color: COLORS.textTertiary }}>{path}</span>
        </div>
      </div>

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          flexGrow: 1,
          gap: "28px",
          paddingTop: "40px",
          paddingBottom: "40px",
        }}
      >
        <div
          style={{
            fontFamily: FONTS.inter,
            fontSize: "64px",
            lineHeight: 0.98,
            letterSpacing: "-0.04em",
            color: COLORS.textPrimary,
            fontWeight: 600,
            maxWidth: "1000px",
          }}
        >
          {safeTitle}
        </div>
        {safeDescription ? (
          <div
            style={{
              fontFamily: FONTS.inter,
              fontSize: "24px",
              lineHeight: 1.4,
              color: COLORS.textSecondary,
              maxWidth: "1000px",
              fontWeight: 400,
            }}
          >
            {safeDescription}
          </div>
        ) : null}
      </div>

      <div
        style={{
          display: "flex",
          flexDirection: "row",
          alignItems: "center",
          justifyContent: "space-between",
          borderTop: `1px solid ${COLORS.border}`,
          paddingTop: "28px",
        }}
      >
        <div
          style={{
            display: "flex",
            fontFamily: FONTS.mono,
            fontSize: "20px",
            color: COLORS.textSecondary,
            fontWeight: 500,
            letterSpacing: "0.04em",
          }}
        >
          agh.network
        </div>
        <div
          style={{
            display: "flex",
            flexDirection: "row",
            gap: "12px",
            alignItems: "center",
          }}
        >
          <Chip>DOCS</Chip>
          <Chip accent>{labels.chip}</Chip>
        </div>
      </div>
    </div>,
    {
      ...SIZE,
      fonts: fonts.map(font => ({
        name: font.name,
        data: font.data,
        weight: font.weight,
        style: font.style,
      })),
    }
  );
}

interface ChipProps {
  children: string;
  accent?: boolean;
}

function Chip({ children, accent = false }: ChipProps) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        padding: "8px 14px",
        border: `1px solid ${accent ? COLORS.accent : COLORS.border}`,
        borderRadius: "5px",
        fontFamily: FONTS.mono,
        fontSize: "16px",
        letterSpacing: "0.14em",
        textTransform: "uppercase",
        color: accent ? COLORS.accent : COLORS.textSecondary,
        fontWeight: 500,
      }}
    >
      {children}
    </div>
  );
}
