import { ImageResponse } from "next/og";
import { siteConfig } from "@/lib/site-config";

export const size = {
  width: 1200,
  height: 630,
};

export const contentType = "image/png";
export const dynamic = "force-static";

export default function Image() {
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        background: "#141312",
        color: "#E5E5E7",
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
          border: "1px solid #3C3A39",
          borderRadius: "28px",
          background: "#1E1C1B",
          padding: "56px",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "18px",
            color: "#E8572A",
            fontSize: "24px",
            letterSpacing: "0.14em",
            textTransform: "uppercase",
          }}
        >
          <span>AGH</span>
          <span style={{ width: "96px", height: "1px", background: "#3C3A39" }} />
          <span style={{ color: "#8E8E93" }}>Agent Operating System</span>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: "28px" }}>
          <div
            style={{
              maxWidth: "760px",
              fontSize: "74px",
              lineHeight: 0.94,
              letterSpacing: "-0.055em",
            }}
          >
            An agent runtime with a network built in.
          </div>
          <div
            style={{
              maxWidth: "770px",
              color: "#8E8E93",
              fontSize: "28px",
              lineHeight: 1.45,
            }}
          >
            {siteConfig.description}
          </div>
        </div>
      </div>
    </div>,
    size
  );
}
