// Docs shell: top nav, 3-column layout, sidebar, TOC, footer.
// Uses global-scoped unique names to avoid collisions.

function DocsHeader() {
  const links = ["Overview", "Runtime", "AGH Network", "Reference", "Examples"];
  return (
    <header
      style={{
        position: "sticky",
        top: 0,
        zIndex: 40,
        background: "rgba(18,18,18,0.92)",
        backdropFilter: "blur(20px)",
        borderBottom: "1px solid #3C3A39",
        padding: "0 20px",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 20, height: 56 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span
            style={{
              fontFamily: "NuixyberNext, Inter, sans-serif",
              fontSize: 22,
              letterSpacing: "-0.02em",
              color: "#E5E5E7",
            }}
          >
            agh
          </span>
          <span
            style={{
              border: "1px solid #3C3A39",
              borderRadius: 3,
              padding: "1px 6px",
              fontFamily: "JetBrains Mono",
              fontSize: 9,
              fontWeight: 500,
              textTransform: "uppercase",
              letterSpacing: "0.14em",
              color: "#8E8E93",
            }}
          >
            Docs
          </span>
        </div>
        <nav style={{ display: "flex", gap: 4 }}>
          {links.map((l, i) => (
            <a
              key={l}
              style={{
                padding: "6px 12px",
                borderRadius: 9999,
                fontSize: 13,
                fontFamily: "Inter",
                textDecoration: "none",
                color: i === 1 ? "#E8572A" : "#8E8E93",
                background: i === 1 ? "rgba(232,87,42,0.12)" : "transparent",
              }}
            >
              {l}
            </a>
          ))}
        </nav>
        <div style={{ flex: 1 }} />
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            background: "rgba(28,28,30,0.92)",
            border: "1px solid #3C3A39",
            borderRadius: 9999,
            padding: "0 10px",
            height: 32,
            color: "#8E8E93",
            fontFamily: "Inter",
            fontSize: 12,
            width: 260,
          }}
        >
          <span>⌕</span>
          <span style={{ flex: 1 }}>Search docs</span>
          <span
            style={{ background: "#2E2C2B", borderRadius: 3, padding: "1px 5px", fontSize: 10 }}
          >
            ⌘K
          </span>
        </div>
      </div>
    </header>
  );
}

function DocsSidebar({ current, onPick }) {
  const sections = [
    {
      heading: "Getting started",
      items: [
        { id: "install", label: "Install" },
        { id: "quickstart", label: "Quickstart" },
        { id: "concepts", label: "Core concepts" },
      ],
    },
    {
      heading: "Runtime",
      items: [
        { id: "sessions", label: "Sessions" },
        { id: "memory", label: "Memory" },
        { id: "skills", label: "Skills" },
        { id: "workspaces", label: "Workspaces" },
        { id: "automation", label: "Automation" },
        { id: "hooks", label: "Hooks" },
      ],
    },
    {
      heading: "AGH Network",
      items: [
        { id: "network-overview", label: "Overview" },
        { id: "network-protocol", label: "Protocol v0" },
        { id: "network-discovery", label: "Discovery" },
        { id: "network-messages", label: "Message kinds" },
      ],
    },
    {
      heading: "Reference",
      items: [
        { id: "cli", label: "CLI commands" },
        { id: "config", label: "Configuration" },
        { id: "events", label: "Event stream" },
      ],
    },
  ];
  return (
    <aside
      style={{
        width: 240,
        flexShrink: 0,
        padding: "32px 16px 32px 20px",
        borderRight: "1px solid #3C3A39",
        position: "sticky",
        top: 56,
        alignSelf: "flex-start",
        height: "calc(100vh - 56px)",
        overflowY: "auto",
      }}
    >
      {sections.map(sec => (
        <div key={sec.heading} style={{ marginBottom: 24 }}>
          <div
            style={{
              fontFamily: "JetBrains Mono",
              fontSize: 10,
              fontWeight: 600,
              textTransform: "uppercase",
              letterSpacing: "0.08em",
              color: "#636366",
              padding: "6px 10px",
            }}
          >
            {sec.heading}
          </div>
          <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
            {sec.items.map(it => (
              <li key={it.id}>
                <a
                  onClick={e => {
                    e.preventDefault();
                    onPick(it.id);
                  }}
                  style={{
                    display: "block",
                    padding: "6px 10px",
                    borderRadius: 6,
                    fontFamily: "Inter",
                    fontSize: 13,
                    cursor: "pointer",
                    textDecoration: "none",
                    color: current === it.id ? "#E8572A" : "#8E8E93",
                    background: current === it.id ? "rgba(232,87,42,0.12)" : "transparent",
                    borderLeft: current === it.id ? "2px solid #E8572A" : "2px solid transparent",
                  }}
                >
                  {it.label}
                </a>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </aside>
  );
}

function DocsToc() {
  const items = [
    { id: "what", label: "What sessions are" },
    { id: "create", label: "Create a session" },
    { id: "resume", label: "Resume & fork" },
    { id: "events", label: "Event stream" },
  ];
  return (
    <aside
      style={{
        width: 200,
        flexShrink: 0,
        padding: "40px 20px 40px 24px",
        position: "sticky",
        top: 56,
        alignSelf: "flex-start",
        height: "calc(100vh - 56px)",
      }}
    >
      <div
        style={{
          fontFamily: "JetBrains Mono",
          fontSize: 10,
          fontWeight: 600,
          textTransform: "uppercase",
          letterSpacing: "0.08em",
          color: "#636366",
          marginBottom: 12,
        }}
      >
        On this page
      </div>
      <ul
        style={{
          listStyle: "none",
          margin: 0,
          padding: 0,
          display: "flex",
          flexDirection: "column",
          gap: 8,
        }}
      >
        {items.map((it, i) => (
          <li key={it.id}>
            <a
              style={{
                fontFamily: "Inter",
                fontSize: 13,
                textDecoration: "none",
                color: i === 0 ? "#E8572A" : "#8E8E93",
              }}
            >
              {it.label}
            </a>
          </li>
        ))}
      </ul>
    </aside>
  );
}

function DocsFooter() {
  return (
    <footer style={{ borderTop: "1px solid #3C3A39", padding: "40px 32px", marginTop: 48 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          gap: 16,
          alignItems: "center",
          flexWrap: "wrap",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span
            style={{
              fontFamily: "NuixyberNext, Inter, sans-serif",
              fontSize: 18,
              color: "#E5E5E7",
            }}
          >
            agh
          </span>
          <span style={{ fontFamily: "JetBrains Mono", fontSize: 11, color: "#636366" }}>
            © 2026 Compozy
          </span>
        </div>
        <div
          style={{ display: "flex", gap: 16, fontFamily: "Inter", fontSize: 13, color: "#8E8E93" }}
        >
          <a style={{ color: "inherit", textDecoration: "none" }}>GitHub</a>
          <a style={{ color: "inherit", textDecoration: "none" }}>Discord</a>
          <a style={{ color: "inherit", textDecoration: "none" }}>Changelog</a>
        </div>
      </div>
    </footer>
  );
}

Object.assign(window, { DocsHeader, DocsSidebar, DocsToc, DocsFooter });
