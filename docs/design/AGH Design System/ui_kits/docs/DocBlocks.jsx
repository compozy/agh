// Doc content blocks: Breadcrumb, H1, Prose, Callout, InlineCode, CommandTable, Card nav pair, Page.

function Breadcrumb({ trail }) {
  return (
    <nav
      aria-label="Breadcrumb"
      style={{
        display: "flex",
        alignItems: "center",
        gap: 6,
        fontFamily: "JetBrains Mono",
        fontSize: 11,
        color: "#636366",
        textTransform: "uppercase",
        letterSpacing: "0.06em",
      }}
    >
      {trail.map((t, i) => (
        <React.Fragment key={t}>
          <span style={{ color: i === trail.length - 1 ? "#E8572A" : "#8E8E93" }}>{t}</span>
          {i < trail.length - 1 && <span>/</span>}
        </React.Fragment>
      ))}
    </nav>
  );
}

function DocH1({ children }) {
  return (
    <h1
      style={{
        margin: "20px 0 0",
        fontFamily: "Inter",
        fontWeight: 600,
        fontSize: "clamp(2.4rem, 4vw, 3rem)",
        lineHeight: 1.04,
        letterSpacing: "-0.045em",
        color: "#E5E5E7",
      }}
    >
      {children}
    </h1>
  );
}

function DocH2({ id, children }) {
  return (
    <h2
      id={id}
      style={{
        margin: "48px 0 0",
        fontFamily: "Inter",
        fontWeight: 600,
        fontSize: 26,
        letterSpacing: "-0.02em",
        color: "#E5E5E7",
        scrollMarginTop: 80,
      }}
    >
      {children}
    </h2>
  );
}

function DocP({ children }) {
  return (
    <p
      style={{
        margin: "16px 0 0",
        fontFamily: "Inter",
        fontSize: 15,
        lineHeight: 1.75,
        color: "#E5E5E7",
      }}
    >
      {children}
    </p>
  );
}

function InlineCode({ children }) {
  return (
    <code
      style={{
        fontFamily: "JetBrains Mono",
        fontSize: "0.88em",
        background: "#2E2C2B",
        color: "#E8572A",
        padding: "1px 6px",
        borderRadius: 4,
      }}
    >
      {children}
    </code>
  );
}

function Callout({ kind = "info", title, children }) {
  const palettes = {
    info: { color: "#0A84FF", bg: "#0A84FF1F" },
    warn: { color: "#FFD60A", bg: "#FFD60A1F" },
    success: { color: "#30D158", bg: "#30D1581F" },
  };
  const p = palettes[kind];
  return (
    <aside
      style={{
        marginTop: 24,
        borderRadius: 10,
        padding: "16px 18px",
        background: p.bg,
        borderLeft: `3px solid ${p.color}`,
      }}
    >
      <div
        style={{
          fontFamily: "JetBrains Mono",
          fontSize: 10,
          fontWeight: 600,
          textTransform: "uppercase",
          letterSpacing: "0.08em",
          color: p.color,
        }}
      >
        {title}
      </div>
      <div
        style={{
          marginTop: 8,
          fontFamily: "Inter",
          fontSize: 14,
          lineHeight: 1.65,
          color: "#E5E5E7",
        }}
      >
        {children}
      </div>
    </aside>
  );
}

function CommandTable({ rows }) {
  return (
    <div
      style={{
        marginTop: 24,
        borderRadius: 10,
        border: "1px solid #3C3A39",
        background: "#1E1C1B",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1.6fr",
          padding: "10px 14px",
          borderBottom: "1px solid #3C3A39",
        }}
      >
        {["Command", "Description"].map(c => (
          <span
            key={c}
            style={{
              fontFamily: "JetBrains Mono",
              fontSize: 10,
              fontWeight: 600,
              textTransform: "uppercase",
              letterSpacing: "0.06em",
              color: "#636366",
            }}
          >
            {c}
          </span>
        ))}
      </div>
      {rows.map(r => (
        <div
          key={r.cmd}
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 1.6fr",
            gap: 12,
            padding: "12px 14px",
            borderTop: "1px solid #3C3A39",
            alignItems: "baseline",
          }}
        >
          <code style={{ fontFamily: "JetBrains Mono", fontSize: 13, color: "#E8572A" }}>
            {r.cmd}
          </code>
          <p
            style={{
              margin: 0,
              fontFamily: "Inter",
              fontSize: 13,
              lineHeight: 1.55,
              color: "#8E8E93",
            }}
          >
            {r.desc}
          </p>
        </div>
      ))}
    </div>
  );
}

function PageNav({ prev, next }) {
  const cell = (item, dir) =>
    item ? (
      <a
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          gap: 6,
          padding: 20,
          borderRadius: 10,
          border: "1px solid #3C3A39",
          background: "#1E1C1B",
          textDecoration: "none",
          textAlign: dir === "next" ? "right" : "left",
        }}
      >
        <span
          style={{
            fontFamily: "JetBrains Mono",
            fontSize: 10,
            fontWeight: 600,
            textTransform: "uppercase",
            letterSpacing: "0.08em",
            color: "#636366",
          }}
        >
          {dir === "prev" ? "← Previous" : "Next →"}
        </span>
        <span style={{ fontFamily: "Inter", fontSize: 14, fontWeight: 500, color: "#E8572A" }}>
          {item}
        </span>
      </a>
    ) : (
      <div style={{ flex: 1 }} />
    );
  return (
    <div style={{ marginTop: 48, display: "flex", gap: 16 }}>
      {cell(prev, "prev")}
      {cell(next, "next")}
    </div>
  );
}

Object.assign(window, {
  Breadcrumb,
  DocH1,
  DocH2,
  DocP,
  InlineCode,
  Callout,
  CommandTable,
  PageNav,
});
