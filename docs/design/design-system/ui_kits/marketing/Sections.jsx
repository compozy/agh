// Home header + Hero + Features + Agents + Install + Comparison + FinalCta

function HomeHeader() {
  return (
    <header
      style={{
        position: "sticky",
        top: 0,
        zIndex: 40,
        borderBottom: "1px solid #3C3A39",
        background: "rgba(18,18,18,0.92)",
        backdropFilter: "blur(20px)",
        padding: "0 16px",
      }}
    >
      <div
        style={{
          maxWidth: 1200,
          margin: "0 auto",
          height: 56,
          display: "flex",
          alignItems: "center",
          gap: 20,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span
            style={{
              fontFamily: "NuixyberNext, Inter, sans-serif",
              fontSize: 22,
              letterSpacing: "-0.02em",
              color: "#E5E5E7",
              lineHeight: 1,
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
            Alpha
          </span>
        </div>
        <nav style={{ display: "flex", gap: 4, marginLeft: 8 }}>
          {[
            ["Home", true],
            ["Runtime", false],
            ["AGH Network", false],
          ].map(([n, a]) => (
            <a
              key={n}
              style={{
                padding: "6px 12px",
                borderRadius: 9999,
                fontSize: 13,
                fontFamily: "Inter",
                textDecoration: "none",
                color: a ? "#E8572A" : "#8E8E93",
                background: a ? "rgba(232,87,42,0.12)" : "transparent",
              }}
            >
              {n}
            </a>
          ))}
        </nav>
        <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 6 }}>
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
              minWidth: 220,
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
          <button
            style={{
              height: 32,
              width: 32,
              borderRadius: 9999,
              border: "1px solid #3C3A39",
              background: "transparent",
              color: "#8E8E93",
              cursor: "pointer",
              fontFamily: "JetBrains Mono",
              fontSize: 10,
              fontWeight: 600,
              letterSpacing: "0.14em",
            }}
          >
            GH
          </button>
        </div>
      </div>
    </header>
  );
}

function Hero() {
  const signals = [
    {
      label: "Complete agent runtime",
      detail: "Sessions, memory, skills, workspaces, automation, bridges — one binary.",
    },
    {
      label: "Built-in agent network",
      detail: "Agents discover peers, delegate work, and collect receipts across machines.",
    },
    {
      label: "Local-first, self-hosted",
      detail: "No Docker. No Postgres. Start with agh daemon start.",
    },
    {
      label: "Open protocol, open source",
      detail: "agh-network/v0 is an open wire spec. Bring any agent you like.",
    },
  ];
  return (
    <section
      style={{
        position: "relative",
        overflow: "hidden",
        borderBottom: "1px solid #3C3A39",
        padding: "48px 16px 64px",
      }}
    >
      <div
        aria-hidden
        style={{
          position: "absolute",
          inset: 0,
          opacity: 0.15,
          mixBlendMode: "screen",
          background:
            "radial-gradient(800px 400px at 20% 10%, rgba(232,87,42,0.18), transparent 60%)," +
            "radial-gradient(600px 400px at 80% 60%, rgba(232,87,42,0.08), transparent 60%)",
        }}
      />
      <div style={{ position: "relative", maxWidth: 1200, margin: "0 auto" }}>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 540px",
            gap: 56,
            alignItems: "center",
          }}
        >
          <div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                fontFamily: "JetBrains Mono",
                fontSize: 11,
                fontWeight: 500,
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                color: "#636366",
              }}
            >
              <span style={{ color: "#E8572A" }}>AGH</span>
              <span style={{ height: 1, width: 40, background: "#3C3A39" }} />
              <span>Agent Operating System</span>
            </div>
            <h1
              style={{
                margin: "24px 0 0",
                maxWidth: "18ch",
                fontFamily: "Playfair Display, serif",
                fontWeight: 400,
                fontSize: "clamp(2.8rem, 5.5vw, 5rem)",
                lineHeight: 0.96,
                letterSpacing: "-0.035em",
                color: "#E5E5E7",
              }}
            >
              An agent runtime with a network built in.
            </h1>
            <p
              style={{
                margin: "24px 0 0",
                maxWidth: "58ch",
                fontFamily: "Inter",
                fontSize: 17,
                lineHeight: 1.6,
                color: "#8E8E93",
              }}
            >
              Sessions, memory, skills, workspaces, automation, bridges — the whole runtime in a
              single local binary. Then the part nobody else ships: an open protocol so your agents
              discover peers, delegate work, and collect receipts across machines.
            </p>
            <div style={{ marginTop: 32, display: "flex", gap: 12, flexWrap: "wrap" }}>
              <CtaButton variant="primary">Install the runtime</CtaButton>
              <CtaButton variant="ghost">See the network</CtaButton>
            </div>
          </div>
          <div
            style={{
              aspectRatio: "4 / 3",
              borderRadius: 12,
              border: "1px solid #3C3A39",
              background: "#0E0E0F",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              color: "#636366",
              fontFamily: "JetBrains Mono",
              fontSize: 11,
              textTransform: "uppercase",
              letterSpacing: "0.08em",
            }}
          >
            Network protocol visual
          </div>
        </div>
        <dl
          style={{
            marginTop: 40,
            display: "grid",
            gridTemplateColumns: "repeat(4, 1fr)",
            gap: 12,
          }}
        >
          {signals.map(s => (
            <div
              key={s.label}
              style={{
                borderRadius: 12,
                border: "1px solid rgba(255,255,255,0.1)",
                padding: 16,
                backdropFilter: "blur(4px)",
              }}
            >
              <dt
                style={{
                  fontFamily: "JetBrains Mono",
                  fontSize: 12,
                  fontWeight: 600,
                  textTransform: "uppercase",
                  letterSpacing: "0.06em",
                  color: "#E8572A",
                }}
              >
                {s.label}
              </dt>
              <dd
                style={{
                  margin: "6px 0 0",
                  fontFamily: "Inter",
                  fontSize: 12,
                  lineHeight: 1.55,
                  color: "#8E8E93",
                }}
              >
                {s.detail}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}

function IconSvg({ d }) {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {d.map((p, i) => (
        <path key={i} d={p} />
      ))}
    </svg>
  );
}

function Features() {
  const features = [
    {
      d: ["M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5", "M3 12c0 1.66 4 3 9 3s9-1.34 9-3"],
      eyebrow: "Sessions",
      title: "Resume any agent run",
      description:
        "Every agent run is a durable session. Stop, resume, inspect every step, fork from any point.",
    },
    {
      d: [
        "M9.9 15.5A2 2 0 0 0 8.5 14L2.4 12.4a.5.5 0 0 1 0-1L8.5 10A2 2 0 0 0 9.9 8.5l1.6-6.1a.5.5 0 0 1 1 0L14.1 8.5A2 2 0 0 0 15.5 10l6.1 1.6a.5.5 0 0 1 0 1L15.5 14a2 2 0 0 0-1.4 1.5L12.5 21.6a.5.5 0 0 1-1 0z",
      ],
      eyebrow: "Memory",
      title: "Context that survives restarts",
      description:
        "Global and per-workspace memory in plain Markdown. Four types, one index per scope.",
    },
    {
      d: [
        "M14 3v4a1 1 0 0 0 1 1h4",
        "M17 21H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h7l5 5v11a2 2 0 0 1-2 2Z",
        "m9 18 3-3-3-3",
        "M5 14l-1.5 1.5",
        "M5 18l-1.5 1.5",
      ],
      eyebrow: "Skills",
      title: "Reusable playbooks",
      description:
        "Drop-in SKILL.md bundles with YAML frontmatter. Bundled library, workspace overrides.",
    },
    {
      d: [
        "M2.97 12.92A2 2 0 0 0 2 14.63v3.24a2 2 0 0 0 .97 1.71l3 1.8a2 2 0 0 0 2.06 0L12 19",
        "m7 16.5-4.74-2.85",
        "m7 16.5 5-3",
        "M7 16.5v5.17",
        "M12 13.5V19l5 3 5-3v-5.5l-5-3Z",
        "m17 16.5-5-3",
        "m17 16.5 5-3",
        "M17 16.5v5.17",
        "M7.97 4.42A2 2 0 0 0 7 6.13v4.37l5 3 5-3V6.13a2 2 0 0 0-.97-1.71l-3-1.8a2 2 0 0 0-2.06 0z",
        "M12 8 7.26 5.15",
        "m12 8 4.74-2.85",
        "M12 13.5V8",
      ],
      eyebrow: "Workspaces",
      title: "Per-project everything",
      description:
        "Agents, skills, memory, and config overlay per workspace. Switch projects, switch context.",
    },
    {
      d: [
        "M5 22h14",
        "M5 2h14",
        "M17 22v-4.17c0-.53-.21-1.04-.59-1.42L13 13l3.41-3.41c.38-.38.59-.89.59-1.42V4",
        "M7 22v-4.17c0-.53.21-1.04.59-1.42L11 13 7.59 9.59A2 2 0 0 1 7 8.17V4",
      ],
      eyebrow: "Automation",
      title: "Cron + webhooks, durable",
      description:
        "Schedule recurring work. Trigger sessions from external events. Every run tracked in SQLite.",
    },
    {
      d: [
        "M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.5.5 0 0 1-.96 0L8.81 3.18a.5.5 0 0 0-.96 0l-2.35 8.36A2 2 0 0 1 3.58 13H2",
      ],
      eyebrow: "Observability",
      title: "Everything logged, replayable",
      description:
        "Token usage, permission audit, tool calls, errors — streamed over SSE, persisted to disk.",
    },
    {
      d: [
        "M9 2v6",
        "M15 2v6",
        "M12 17v5",
        "M5 8h14a2 2 0 0 1 2 2v2a5 5 0 0 1-5 5H8a5 5 0 0 1-5-5v-2a2 2 0 0 1 2-2",
      ],
      eyebrow: "Hooks",
      title: "Inject logic anywhere",
      description:
        "Run scripts or sub-agents on ~40 lifecycle events — permission checks, tool calls, receipts.",
    },
    {
      d: [
        "M16 22h2a2 2 0 0 0 2-2V7l-5-5H6a2 2 0 0 0-2 2v3",
        "M14 2v4a2 2 0 0 0 2 2h4",
        "M2 15h10",
        "m5 12-3 3 3 3",
        "m9 18 3-3-3-3",
      ],
      eyebrow: "Bridges",
      title: "Slack, Discord, Telegram in",
      description:
        "Platform webhooks become sessions. Response events stream back to the original thread.",
    },
  ];
  return (
    <section style={{ background: "#141312", padding: "80px 16px" }}>
      <div style={{ maxWidth: 1200, margin: "0 auto" }}>
        <SectionHeader
          eyebrow="What you get"
          title="Everything a modern agent runtime should have."
          description="You already know you need sessions, memory, and skills. AGH ships all of it, local-first, with an operator surface you can script."
        />
        <ul
          style={{
            marginTop: 48,
            padding: 0,
            listStyle: "none",
            display: "grid",
            gridTemplateColumns: "repeat(4, 1fr)",
            gap: 16,
          }}
        >
          {features.map(f => (
            <li key={f.eyebrow}>
              <FeatureCard
                icon={<IconSvg d={f.d} />}
                eyebrow={f.eyebrow}
                title={f.title}
                description={f.description}
              />
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function SupportedAgents() {
  const providers = ["claude", "codex", "gemini", "opencode", "copilot", "cursor", "kiro", "pi"];
  return (
    <section style={{ background: "#141312", padding: "40px 16px" }}>
      <div
        style={{
          maxWidth: 1200,
          margin: "0 auto",
          display: "flex",
          gap: 48,
          alignItems: "center",
          flexWrap: "wrap",
        }}
      >
        <div style={{ maxWidth: "38ch" }}>
          <Eyebrow>Works with your agent CLIs</Eyebrow>
          <p style={{ margin: "8px 0 0", fontFamily: "Inter", fontSize: 16, color: "#E5E5E7" }}>
            Bring the CLI you already use. AGH spawns it, manages it, and persists every event.
          </p>
        </div>
        <ul
          style={{
            padding: 0,
            listStyle: "none",
            display: "grid",
            gridTemplateColumns: "repeat(8, 1fr)",
            gap: 8,
            flex: 1,
          }}
        >
          {providers.map(p => (
            <li
              key={p}
              style={{
                height: 64,
                borderRadius: 10,
                border: "1px solid #3C3A39",
                background: "#1E1C1B",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                gap: 4,
              }}
            >
              <div style={{ width: 22, height: 22, borderRadius: 6, background: "#2E2C2B" }} />
              <span
                style={{
                  fontFamily: "JetBrains Mono",
                  fontSize: 9,
                  textTransform: "uppercase",
                  letterSpacing: "0.06em",
                  color: "#636366",
                }}
              >
                {p}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function InstallSection() {
  const [tab, setTab] = React.useState("brew");
  const tabs = [
    {
      id: "brew",
      label: "Homebrew",
      command: "brew install compozy/tap/agh",
      note: "macOS · recommended",
    },
    {
      id: "go",
      label: "go install",
      command: "go install github.com/compozy/agh/cmd/agh@latest",
      note: "Linux + macOS · Go 1.25+",
    },
    {
      id: "binary",
      label: "Binary",
      command: "curl -fsSL https://get.agh.network | sh",
      note: "Linux + macOS · prebuilt",
    },
  ];
  const active = tabs.find(t => t.id === tab);
  const steps = [
    {
      step: "01",
      title: "Start the daemon",
      description: "One local process, detaches to background, logs to $AGH_HOME/logs/agh.log.",
      code: "agh daemon start",
    },
    {
      step: "02",
      title: "Launch a session",
      description:
        "Spawn an ACP agent as a managed subprocess. Events start streaming immediately.",
      code: "agh session new --agent coder --provider claude",
    },
    {
      step: "03",
      title: "Discover peers",
      description:
        "The network runtime starts alongside the daemon. Other AGH peers on the same channel appear here.",
      code: "agh network peers",
      live: true,
    },
  ];
  return (
    <section style={{ background: "#1E1C1B", padding: "80px 16px" }}>
      <div style={{ maxWidth: 1200, margin: "0 auto" }}>
        <SectionHeader
          align="center"
          eyebrow="Getting started"
          title="Three commands. First session in under a minute."
          description="macOS and Linux today. Homebrew, go install, or a prebuilt binary — pick one."
        />
        <div style={{ maxWidth: 760, margin: "40px auto 0" }}>
          <div
            style={{
              display: "flex",
              gap: 4,
              padding: 4,
              borderRadius: 8,
              border: "1px solid #3C3A39",
              background: "#141312",
            }}
          >
            {tabs.map(t => (
              <button
                key={t.id}
                onClick={() => setTab(t.id)}
                style={{
                  flex: 1,
                  height: 30,
                  borderRadius: 5,
                  border: "none",
                  cursor: "pointer",
                  fontFamily: "JetBrains Mono",
                  fontSize: 12,
                  letterSpacing: "0.02em",
                  background: t.id === tab ? "rgba(232,87,42,0.15)" : "transparent",
                  color: t.id === tab ? "#E8572A" : "#8E8E93",
                }}
              >
                {t.label}
              </button>
            ))}
          </div>
          <div style={{ marginTop: 16 }}>
            <CodeBlock code={active.command} caption={active.note} shell />
          </div>
          <div style={{ marginTop: 40, display: "flex", flexDirection: "column", gap: 16 }}>
            {steps.map(s => (
              <div
                key={s.step}
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: 16,
                  borderRadius: 12,
                  border: "1px solid #3C3A39",
                  background: "#141312",
                  padding: 24,
                }}
              >
                <div style={{ display: "flex", gap: 16 }}>
                  <span
                    style={{
                      fontFamily: "JetBrains Mono",
                      fontSize: 18,
                      fontWeight: 500,
                      color: "#E8572A",
                      marginTop: 2,
                    }}
                  >
                    {s.step}
                  </span>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      <h3
                        style={{
                          margin: 0,
                          fontFamily: "Inter",
                          fontSize: 18,
                          fontWeight: 500,
                          color: "#E5E5E7",
                        }}
                      >
                        {s.title}
                      </h3>
                      {s.live && (
                        <span
                          style={{
                            background: "#30D15826",
                            color: "#30D158",
                            borderRadius: 6,
                            padding: "2px 8px",
                            fontFamily: "JetBrains Mono",
                            fontSize: 10,
                            fontWeight: 600,
                            textTransform: "uppercase",
                            letterSpacing: "0.06em",
                            display: "inline-flex",
                            alignItems: "center",
                            gap: 6,
                          }}
                        >
                          <span
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: 999,
                              background: "#30D158",
                            }}
                          />
                          online
                        </span>
                      )}
                    </div>
                    <p
                      style={{
                        margin: "6px 0 0",
                        fontFamily: "Inter",
                        fontSize: 14,
                        lineHeight: 1.55,
                        color: "#8E8E93",
                      }}
                    >
                      {s.description}
                    </p>
                  </div>
                </div>
                <div style={{ marginLeft: 44 }}>
                  <CodeBlock code={s.code} caption="shell" shell />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function Comparison() {
  const rows = [
    {
      approach: "Assistant gateway",
      focus: "Personal AI across chat",
      model: "Single assistant + plugins",
      coord: "None — one agent per user",
      deploy: "Cloud-hosted",
      cross: false,
    },
    {
      approach: "All-in-one agent OS",
      focus: "Broad built-in capabilities",
      model: "Custom agents in platform",
      coord: "Internal only",
      deploy: "Cloud or self-hosted",
      cross: false,
    },
    {
      approach: "Multi-tenant gateway",
      focus: "Enterprise AI platform",
      model: "Managed agents behind an API",
      coord: "Centralized routing",
      deploy: "Cloud-hosted",
      cross: false,
    },
    {
      approach: "AGH",
      focus: "Orchestrate real agent CLIs",
      model: "Your existing ACP agents",
      coord: "agh-network/v0 — shipped",
      deploy: "Local-first, single binary",
      cross: true,
      highlight: true,
    },
  ];
  const cols = [
    ["focus", "Primary focus"],
    ["model", "Agent model"],
    ["coord", "Coordination"],
    ["deploy", "Deployment"],
  ];
  return (
    <section style={{ background: "#141312", padding: "80px 16px" }}>
      <div style={{ maxWidth: 1200, margin: "0 auto" }}>
        <SectionHeader
          eyebrow="Positioning"
          title="Other tools stop at the runtime boundary."
          description="AGH is the only approach with a shipped cross-runtime protocol. The rest centralize coordination or skip it entirely."
        />
        <div
          style={{
            marginTop: 40,
            borderRadius: 12,
            border: "1px solid #3C3A39",
            background: "#1E1C1B",
            overflow: "hidden",
          }}
        >
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "180px repeat(4, 1fr) 60px",
              gap: 16,
              padding: "16px 20px",
              borderBottom: "1px solid #3C3A39",
            }}
          >
            <span
              style={{
                fontFamily: "JetBrains Mono",
                fontSize: 10,
                fontWeight: 600,
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                color: "#636366",
              }}
            >
              Approach
            </span>
            {cols.map(([k, l]) => (
              <span
                key={k}
                style={{
                  fontFamily: "JetBrains Mono",
                  fontSize: 10,
                  fontWeight: 600,
                  textTransform: "uppercase",
                  letterSpacing: "0.06em",
                  color: "#636366",
                }}
              >
                {l}
              </span>
            ))}
            <span
              style={{
                textAlign: "right",
                fontFamily: "JetBrains Mono",
                fontSize: 10,
                fontWeight: 600,
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                color: "#636366",
              }}
            >
              Cross
            </span>
          </div>
          {rows.map(r => (
            <div
              key={r.approach}
              style={{
                display: "grid",
                gridTemplateColumns: "180px repeat(4, 1fr) 60px",
                gap: 16,
                padding: "18px 20px",
                alignItems: "center",
                borderTop: "1px solid #3C3A39",
                ...(r.highlight
                  ? {
                      borderLeft: "4px solid #E8572A",
                      background: "color-mix(in srgb, #E8572A26 40%, transparent)",
                    }
                  : {}),
              }}
            >
              <h3
                style={{
                  margin: 0,
                  fontFamily: "Inter",
                  fontSize: 14,
                  fontWeight: 600,
                  color: r.highlight ? "#E8572A" : "#E5E5E7",
                }}
              >
                {r.approach}
              </h3>
              {cols.map(([k]) => (
                <p
                  key={k}
                  style={{
                    margin: 0,
                    fontFamily: "Inter",
                    fontSize: 13,
                    lineHeight: 1.55,
                    color: r.highlight && k === "coord" ? "#E5E5E7" : "#8E8E93",
                    fontWeight: r.highlight && k === "coord" ? 500 : 400,
                  }}
                >
                  {r[k]}
                </p>
              ))}
              <span
                style={{
                  marginLeft: "auto",
                  width: 24,
                  height: 24,
                  borderRadius: 6,
                  display: "inline-flex",
                  alignItems: "center",
                  justifyContent: "center",
                  background: r.cross ? "#30D15826" : "#2E2C2B",
                  color: r.cross ? "#30D158" : "#636366",
                  fontWeight: 700,
                }}
              >
                {r.cross ? "✓" : "–"}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function FinalCta() {
  return (
    <section style={{ background: "#1E1C1B", padding: "60px 16px" }}>
      <div style={{ maxWidth: 1200, margin: "0 auto" }}>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 340px",
            gap: 32,
            alignItems: "center",
            borderRadius: 12,
            border: "1px solid #3C3A39",
            background: "#141312",
            padding: "40px 40px",
          }}
        >
          <div>
            <Eyebrow>Ship it</Eyebrow>
            <h2
              style={{
                margin: "16px 0 0",
                fontFamily: "Playfair Display, serif",
                fontWeight: 400,
                fontSize: "clamp(2rem, 4vw, 3rem)",
                letterSpacing: "-0.03em",
                lineHeight: 1.02,
                color: "#E5E5E7",
                maxWidth: "18ch",
              }}
            >
              Install AGH. Run a session. Join the network.
            </h2>
            <p
              style={{
                margin: "20px 0 0",
                fontFamily: "Inter",
                fontSize: 14,
                lineHeight: 1.7,
                color: "#8E8E93",
              }}
            >
              One binary. No infrastructure. Shipped today.
            </p>
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 12, alignItems: "stretch" }}>
            <CtaButton variant="primary">Install AGH</CtaButton>
            <CtaButton variant="ghost">Read agh-network/v0 spec</CtaButton>
            <a
              style={{
                marginTop: 4,
                fontFamily: "JetBrains Mono",
                fontSize: 12,
                textTransform: "uppercase",
                letterSpacing: "0.06em",
                color: "#8E8E93",
                textDecoration: "none",
                display: "inline-flex",
                alignItems: "center",
                gap: 8,
              }}
            >
              ★ Star on GitHub
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}

Object.assign(window, {
  HomeHeader,
  Hero,
  Features,
  SupportedAgents,
  InstallSection,
  Comparison,
  FinalCta,
});
