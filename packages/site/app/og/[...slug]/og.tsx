import type { ImageResponseOptions } from "next/dist/compiled/@vercel/og/types";
import { ImageResponse } from "next/og";

export interface OGImageOptions {
  eyebrow: string;
  title: string;
  description?: string;
}

const SIZE = { width: 1200, height: 630 } as const;

const COLORS = {
  background: "#141312",
  surface: "#1E1C1B",
  border: "#3C3A39",
  accent: "#E8572A",
  textPrimary: "#E5E5E7",
  textSecondary: "#8E8E93",
} as const;

function truncate(value: string | undefined, max: number): string {
  if (!value) return "";
  if (value.length <= max) return value;
  return `${value.slice(0, max - 1).trimEnd()}…`;
}

export function generateOGImage({
  eyebrow,
  title,
  description,
  ...rest
}: OGImageOptions & ImageResponseOptions): ImageResponse {
  const subtitle = truncate(description, 165);

  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        background: COLORS.background,
        color: COLORS.textPrimary,
        fontFamily: "Inter, sans-serif",
        padding: "72px",
      }}
    >
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          width: "100%",
          border: `1px solid ${COLORS.border}`,
          borderRadius: "28px",
          background: COLORS.surface,
          padding: "56px",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "18px",
            color: COLORS.accent,
            fontSize: "24px",
            letterSpacing: "0.14em",
            textTransform: "uppercase",
          }}
        >
          <span>AGH</span>
          <span style={{ width: "96px", height: "1px", background: COLORS.border }} />
          <span style={{ color: COLORS.textSecondary }}>{eyebrow}</span>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
          <div
            style={{
              maxWidth: "920px",
              fontSize: "72px",
              lineHeight: 0.96,
              letterSpacing: "-0.045em",
              fontWeight: 500,
            }}
          >
            {title}
          </div>
          {subtitle ? (
            <div
              style={{
                maxWidth: "920px",
                color: COLORS.textSecondary,
                fontSize: "26px",
                lineHeight: 1.4,
              }}
            >
              {subtitle}
            </div>
          ) : null}
        </div>
      </div>
    </div>,
    {
      ...SIZE,
      ...rest,
    }
  );
}
