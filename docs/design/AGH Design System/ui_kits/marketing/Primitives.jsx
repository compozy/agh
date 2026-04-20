// Shared primitives for the marketing UI kit. Global-scoped style objects use
// unique names to avoid the global `styles` collision.

const mkEyebrow = accent => ({
  fontFamily: "JetBrains Mono, monospace",
  fontSize: 11,
  fontWeight: 600,
  textTransform: "uppercase",
  letterSpacing: "0.08em",
  color: accent ? "#E8572A" : "#636366",
});

function Eyebrow({ children, accent = true, style }) {
  return <p style={{ ...mkEyebrow(accent), margin: 0, ...style }}>{children}</p>;
}

function MonoBadge({ children, tone = "accent" }) {
  const tones = {
    accent: { background: "#E8572A26", color: "#E8572A" },
    neutral: { background: "#2E2C2B", color: "#636366" },
    success: { background: "#30D15826", color: "#30D158" },
  };
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        fontFamily: "JetBrains Mono, monospace",
        fontSize: 10,
        fontWeight: 600,
        textTransform: "uppercase",
        letterSpacing: "0.08em",
        padding: "3px 8px",
        borderRadius: 6,
        ...tones[tone],
      }}
    >
      {children}
    </span>
  );
}

function CtaButton({ children, variant = "primary", onClick }) {
  const base = {
    fontFamily: "Inter, sans-serif",
    fontSize: 14,
    fontWeight: 500,
    height: 40,
    padding: "0 20px",
    borderRadius: 8,
    display: "inline-flex",
    alignItems: "center",
    gap: 6,
    border: "1px solid transparent",
    cursor: "pointer",
    transition: "all .15s",
    textDecoration: "none",
  };
  const variants = {
    primary: { background: "#E8572A", color: "#17110F" },
    ghost: { background: "transparent", borderColor: "#3C3A39", color: "#E5E5E7" },
  };
  return (
    <button onClick={onClick} style={{ ...base, ...variants[variant] }}>
      {children}
    </button>
  );
}

function FeatureCard({ icon, eyebrow, title, description, cite }) {
  const [hover, setHover] = React.useState(false);
  return (
    <article
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 12,
        borderRadius: 12,
        padding: 24,
        background: "#1E1C1B",
        border: `1px solid ${hover ? "color-mix(in srgb, #E8572A 40%, #3C3A39)" : "#3C3A39"}`,
        transition: "border-color .15s",
        height: "100%",
        boxSizing: "border-box",
      }}
    >
      <div
        style={{
          width: 40,
          height: 40,
          borderRadius: 10,
          background: "#2E2C2B",
          color: "#E8572A",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        {icon}
      </div>
      <Eyebrow>{eyebrow}</Eyebrow>
      <h3
        style={{
          margin: 0,
          fontFamily: "Inter",
          fontSize: 16,
          fontWeight: 500,
          color: "#E5E5E7",
          lineHeight: 1.3,
        }}
      >
        {title}
      </h3>
      <p
        style={{ margin: 0, fontFamily: "Inter", fontSize: 14, lineHeight: 1.55, color: "#8E8E93" }}
      >
        {description}
      </p>
      {cite && (
        <a
          style={{
            marginTop: "auto",
            paddingTop: 8,
            fontFamily: "JetBrains Mono",
            fontSize: 10,
            textTransform: "uppercase",
            letterSpacing: "0.08em",
            color: "#636366",
            display: "inline-flex",
            alignItems: "center",
            gap: 4,
            textDecoration: "none",
          }}
        >
          {cite} ↗
        </a>
      )}
    </article>
  );
}

function SectionHeader({ eyebrow, title, description, align = "start", size = "md" }) {
  const titleStyle =
    size === "lg"
      ? { fontSize: "clamp(2.6rem, 5.5vw, 4.2rem)", lineHeight: 0.98, letterSpacing: "-0.035em" }
      : { fontSize: "clamp(2.2rem, 4.6vw, 3.6rem)", lineHeight: 1.02, letterSpacing: "-0.03em" };
  return (
    <div
      style={{
        maxWidth: align === "center" ? 720 : 640,
        marginLeft: align === "center" ? "auto" : 0,
        marginRight: align === "center" ? "auto" : 0,
        textAlign: align === "center" ? "center" : "left",
      }}
    >
      {eyebrow && <Eyebrow accent={false}>{eyebrow}</Eyebrow>}
      <h2
        style={{
          margin: "20px 0 0",
          color: "#E5E5E7",
          fontWeight: 400,
          fontFamily: "Playfair Display, serif",
          ...titleStyle,
        }}
      >
        {title}
      </h2>
      {description && (
        <p
          style={{
            margin: "20px 0 0",
            fontFamily: "Inter",
            fontSize: 16,
            lineHeight: 1.7,
            color: "#8E8E93",
            maxWidth: 62 * 8,
            marginLeft: align === "center" ? "auto" : 0,
            marginRight: align === "center" ? "auto" : 0,
          }}
        >
          {description}
        </p>
      )}
    </div>
  );
}

function CodeBlock({ code, caption = "shell", shell = true }) {
  const lines = code.split("\n");
  return (
    <div
      style={{
        borderRadius: 12,
        border: "1px solid #3C3A39",
        background: "#0E0E0F",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "10px 16px",
          borderBottom: "1px solid #3C3A39",
        }}
      >
        <span
          style={{
            fontFamily: "JetBrains Mono",
            fontSize: 10,
            fontWeight: 500,
            textTransform: "uppercase",
            letterSpacing: "0.06em",
            color: "#636366",
          }}
        >
          {caption}
        </span>
        <span style={{ color: "#636366", fontFamily: "JetBrains Mono", fontSize: 11 }}>⧉</span>
      </div>
      <pre
        style={{
          margin: 0,
          padding: "14px 16px",
          fontFamily: "JetBrains Mono",
          fontSize: 13,
          lineHeight: 1.7,
          color: "#E5E5E7",
          overflowX: "auto",
        }}
      >
        {lines.map((l, i) => (
          <div key={i}>
            {shell && l && !l.startsWith("#") && (
              <span style={{ color: "#E8572A", userSelect: "none" }}>$ </span>
            )}
            {l.startsWith("#") ? <span style={{ color: "#636366" }}>{l}</span> : l}
          </div>
        ))}
      </pre>
    </div>
  );
}

Object.assign(window, { Eyebrow, MonoBadge, CtaButton, FeatureCard, SectionHeader, CodeBlock });
