// Tasks, Network, Automation, Bridges, Knowledge, Skills, Session, Settings page components
const { Icon, PageHeader, Pills, SearchInput, Empty, Section, Metric } = window;

// ====================== TASKS ======================
const TASKS = [
  {
    id: "t-1",
    name: "Refactor event mapper",
    agent: "claude",
    status: "running",
    priority: "high",
    created: "2h ago",
    runs: 4,
    description: "Extract tool call grouping into pure helpers; add replay coverage.",
  },
  {
    id: "t-2",
    name: "Add NATS retry backoff",
    agent: "codex",
    status: "pending",
    priority: "med",
    created: "1d ago",
    runs: 0,
    description: "Jittered exponential backoff on transient publisher failures.",
  },
  {
    id: "t-3",
    name: "Streaming buffer leak",
    agent: "claude",
    status: "failed",
    priority: "high",
    created: "3h ago",
    runs: 2,
    description: "Investigate unreleased chunks in the persisted tool-state store.",
  },
  {
    id: "t-4",
    name: "Bridge health telemetry",
    agent: "gemini",
    status: "done",
    priority: "med",
    created: "5h ago",
    runs: 1,
    description: "Emit span per provider fan-out; tag delivery latency.",
  },
  {
    id: "t-5",
    name: "Rewrite permission prompt",
    agent: "claude",
    status: "pending",
    priority: "low",
    created: "8h ago",
    runs: 0,
    description: "Align copy with operator voice; reduce surface area.",
  },
  {
    id: "t-6",
    name: "Docs masthead spacing",
    agent: "codex",
    status: "running",
    priority: "low",
    created: "20m ago",
    runs: 1,
    description: "Tighten fumadocs H1 bottom padding on mobile.",
  },
  {
    id: "t-7",
    name: "Memory GC policy",
    agent: "claude",
    status: "done",
    priority: "high",
    created: "2d ago",
    runs: 3,
    description: "TTL+size-cap compaction; rebuild rotates quietly.",
  },
];

function TasksPage() {
  const [mode, setMode] = React.useState("list");
  const [selId, setSelId] = React.useState("t-1");
  const [q, setQ] = React.useState("");

  const filtered = TASKS.filter(t => !q || t.name.toLowerCase().includes(q.toLowerCase()));
  const sel = TASKS.find(t => t.id === selId);

  return (
    <>
      <PageHeader
        title="Tasks"
        icon={Icon.ListChecks}
        count={TASKS.length}
        controls={
          <Pills
            value={mode}
            onChange={setMode}
            items={[
              { value: "list", label: "List" },
              { value: "kanban", label: "Kanban" },
              { value: "dashboard", label: "Dashboard" },
              { value: "inbox", label: "Inbox", badge: 3 },
            ]}
          />
        }
        meta={
          <button className="btn btn-ghost">
            <Icon.Plus size={13} />
            Task
          </button>
        }
      />
      {mode === "list" && (
        <div className="split">
          <div className="split-list">
            <SearchInput value={q} onChange={setQ} placeholder="Filter tasks…" />
            <div className="scroll-y" style={{ flex: 1 }}>
              {filtered.map(t => (
                <div
                  key={t.id}
                  className={`list-row ${selId === t.id ? "active" : ""}`}
                  onClick={() => setSelId(t.id)}
                >
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className="flex items-center gap-2" style={{ marginBottom: 6 }}>
                      <StatusDot status={t.status} />
                      <span
                        className="truncate"
                        style={{
                          fontSize: 13,
                          fontWeight: 500,
                          color: "var(--color-text-primary)",
                        }}
                      >
                        {t.name}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="mono-chip">{t.agent}</span>
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        {t.created}
                      </span>
                      <span
                        className="mono"
                        style={{
                          fontSize: 10,
                          color: "var(--color-text-tertiary)",
                          marginLeft: "auto",
                        }}
                      >
                        {t.runs} runs
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="split-detail">
            {sel ? (
              <TaskDetail task={sel} />
            ) : (
              <Empty
                icon={Icon.ListChecks}
                title="Select a task"
                description="Pick an item from the list to see its runs, dependencies, and preview."
              />
            )}
          </div>
        </div>
      )}
      {mode === "kanban" && <TasksKanban />}
      {mode === "dashboard" && <TasksDashboard />}
      {mode === "inbox" && <TasksInbox />}
    </>
  );
}

function StatusDot({ status }) {
  const map = {
    running: { cls: "accent pulse", label: "Running" },
    pending: { cls: "", label: "Pending" },
    done: { cls: "success", label: "Done" },
    failed: { cls: "danger", label: "Failed" },
  };
  const m = map[status] || map.pending;
  return <span className={`dot ${m.cls}`} title={m.label} />;
}

function TaskDetail({ task }) {
  return (
    <div className="scroll-y" style={{ padding: 24 }}>
      <div className="flex items-center gap-3">
        <StatusDot status={task.status} />
        <h1 style={{ margin: 0, fontSize: 20, fontWeight: 500, letterSpacing: "-0.01em" }}>
          {task.name}
        </h1>
        <span className="mono-chip accent" style={{ marginLeft: "auto" }}>
          {task.status}
        </span>
      </div>
      <div className="flex gap-2" style={{ marginTop: 10 }}>
        <span className="mono-chip">{task.agent}</span>
        <span className="mono-chip">priority · {task.priority}</span>
        <span className="mono-chip">{task.runs} runs</span>
      </div>
      <p
        style={{
          marginTop: 16,
          color: "var(--color-text-secondary)",
          fontSize: 14,
          lineHeight: 1.7,
          maxWidth: 62 * 8,
        }}
      >
        {task.description}
      </p>

      <div className="flex gap-2" style={{ marginTop: 20 }}>
        <button className="btn btn-primary">
          <Icon.Play size={12} />
          Run task
        </button>
        <button className="btn btn-ghost">
          <Icon.FileCode size={12} />
          Edit
        </button>
        <button className="btn btn-ghost">
          <Icon.Copy size={12} />
          Duplicate
        </button>
      </div>

      <div
        style={{ marginTop: 28, display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 12 }}
      >
        <Metric label="Total runs" value={task.runs} detail="lifetime" />
        <Metric label="Avg latency" value="4.2s" tone="accent" />
        <Metric label="Success rate" value="87%" detail="last 30" />
      </div>

      <Section label="Recent runs">
        <div style={{ padding: "0 4px" }}>
          {[0, 1, 2, 3].map(i => (
            <div
              key={i}
              style={{
                display: "grid",
                gridTemplateColumns: "14px 1fr 100px 80px 60px",
                gap: 12,
                padding: "10px 16px",
                borderBottom: "1px solid var(--color-divider)",
                alignItems: "center",
              }}
            >
              <span
                className={`dot ${i === 0 ? "accent pulse" : i === 2 ? "danger" : "success"}`}
              />
              <span className="mono" style={{ fontSize: 12, color: "var(--color-text-primary)" }}>
                run-{(8471 - i).toString(16)}
              </span>
              <span className="mono" style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}>
                {i * 17 + 3}m ago
              </span>
              <span className="mono" style={{ fontSize: 11, color: "var(--color-text-secondary)" }}>
                {(2 + Math.random() * 4).toFixed(1)}s
              </span>
              <Icon.ChevronRight
                size={12}
                style={{ color: "var(--color-text-tertiary)", marginLeft: "auto" }}
              />
            </div>
          ))}
        </div>
      </Section>

      <Section label="Preview">
        <div
          style={{
            margin: "4px 20px 20px",
            background: "var(--color-canvas-deep)",
            border: "1px solid var(--color-divider)",
            borderRadius: 10,
            padding: 16,
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            color: "var(--color-text-secondary)",
            lineHeight: 1.7,
          }}
        >
          <div>
            <span style={{ color: "var(--color-text-tertiary)" }}># scope</span>{" "}
            workspace://agh-core
          </div>
          <div>
            <span style={{ color: "var(--color-text-tertiary)" }}># agent</span> {task.agent}
          </div>
          <div>
            <span style={{ color: "var(--color-text-tertiary)" }}># prompt</span>
          </div>
          <div style={{ color: "var(--color-text-primary)" }}>{task.description}</div>
        </div>
      </Section>
    </div>
  );
}

function TasksKanban() {
  const cols = [
    { key: "pending", label: "Pending", tone: "" },
    { key: "running", label: "Running", tone: "accent" },
    { key: "done", label: "Done", tone: "success" },
    { key: "failed", label: "Failed", tone: "danger" },
  ];
  return (
    <div className="scroll-x" style={{ flex: 1, display: "flex", gap: 14, padding: 20 }}>
      {cols.map(c => {
        const list = TASKS.filter(t => t.status === c.key);
        return (
          <div
            key={c.key}
            style={{ width: 280, flexShrink: 0, display: "flex", flexDirection: "column", gap: 8 }}
          >
            <div className="flex items-center gap-2" style={{ padding: "0 4px" }}>
              <span className={`dot ${c.tone}`} />
              <span className="eyebrow">{c.label}</span>
              <span
                className="mono"
                style={{ fontSize: 10, color: "var(--color-text-tertiary)", marginLeft: "auto" }}
              >
                {list.length}
              </span>
            </div>
            {list.map(t => (
              <div key={t.id} className="card" style={{ padding: 14, cursor: "pointer" }}>
                <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 6 }}>{t.name}</div>
                <div
                  style={{
                    fontSize: 12,
                    color: "var(--color-text-secondary)",
                    lineHeight: 1.5,
                    marginBottom: 10,
                  }}
                >
                  {t.description.slice(0, 60)}…
                </div>
                <div className="flex items-center gap-2">
                  <span className="mono-chip">{t.agent}</span>
                  <span
                    className="mono"
                    style={{
                      fontSize: 10,
                      color: "var(--color-text-tertiary)",
                      marginLeft: "auto",
                    }}
                  >
                    {t.created}
                  </span>
                </div>
              </div>
            ))}
          </div>
        );
      })}
    </div>
  );
}

function TasksDashboard() {
  return (
    <div className="scroll-y" style={{ flex: 1, padding: 24 }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12 }}>
        <Metric label="Tasks total" value="142" detail="+12 wk" />
        <Metric label="Runs today" value="38" tone="accent" />
        <Metric label="Success rate" value="91%" detail="24h" />
        <Metric label="p95 latency" value="6.1s" />
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1.6fr 1fr", gap: 16, marginTop: 20 }}>
        <div className="card" style={{ padding: 20 }}>
          <div className="flex items-center justify-between" style={{ marginBottom: 16 }}>
            <span className="eyebrow">Active runs</span>
            <span className="mono" style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}>
              live
            </span>
          </div>
          {[0, 1, 2, 3].map(i => (
            <div
              key={i}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 12,
                padding: "10px 0",
                borderBottom: i < 3 ? "1px solid var(--color-divider)" : "none",
              }}
            >
              <span className="dot accent pulse" />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 13, fontWeight: 500 }}>{TASKS[i].name}</div>
                <div
                  className="mono"
                  style={{ fontSize: 10, color: "var(--color-text-tertiary)", marginTop: 2 }}
                >
                  {TASKS[i].agent} · run-{(8400 + i).toString(16)}
                </div>
              </div>
              <div
                style={{
                  width: 160,
                  height: 4,
                  background: "var(--color-surface-elevated)",
                  borderRadius: 2,
                  overflow: "hidden",
                }}
              >
                <div
                  style={{
                    width: `${30 + i * 20}%`,
                    height: "100%",
                    background: "var(--color-accent)",
                  }}
                />
              </div>
            </div>
          ))}
        </div>
        <div className="card" style={{ padding: 20 }}>
          <span className="eyebrow">Status breakdown</span>
          <div style={{ marginTop: 16, display: "flex", flexDirection: "column", gap: 10 }}>
            {[
              { l: "Running", v: 14, c: "accent" },
              { l: "Pending", v: 38, c: "" },
              { l: "Done", v: 76, c: "success" },
              { l: "Failed", v: 14, c: "danger" },
            ].map(r => (
              <div key={r.l} className="flex items-center gap-3">
                <span
                  className="mono"
                  style={{ fontSize: 11, color: "var(--color-text-secondary)", width: 60 }}
                >
                  {r.l}
                </span>
                <div
                  style={{
                    flex: 1,
                    height: 6,
                    background: "var(--color-surface-elevated)",
                    borderRadius: 3,
                    overflow: "hidden",
                  }}
                >
                  <div
                    style={{
                      width: `${r.v}%`,
                      height: "100%",
                      background:
                        r.c === "accent"
                          ? "var(--color-accent)"
                          : r.c === "success"
                            ? "var(--color-success)"
                            : r.c === "danger"
                              ? "var(--color-danger)"
                              : "var(--color-text-tertiary)",
                    }}
                  />
                </div>
                <span
                  className="mono"
                  style={{
                    fontSize: 11,
                    color: "var(--color-text-primary)",
                    width: 30,
                    textAlign: "right",
                  }}
                >
                  {r.v}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="card" style={{ padding: 20, marginTop: 16 }}>
        <span className="eyebrow">Queue health</span>
        <div
          style={{
            marginTop: 14,
            display: "grid",
            gridTemplateColumns: "repeat(24, 1fr)",
            gap: 2,
            height: 56,
          }}
        >
          {Array.from({ length: 24 }).map((_, i) => {
            const h = 20 + Math.sin(i / 2) * 20 + Math.random() * 16;
            return (
              <div
                key={i}
                style={{
                  background: i > 18 ? "var(--color-accent)" : "var(--color-surface-elevated)",
                  height: `${h}%`,
                  alignSelf: "end",
                  borderRadius: 2,
                }}
              />
            );
          })}
        </div>
        <div className="flex" style={{ justifyContent: "space-between", marginTop: 8 }}>
          <span className="mono" style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}>
            24h ago
          </span>
          <span className="mono" style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}>
            now
          </span>
        </div>
      </div>
    </div>
  );
}

function TasksInbox() {
  const items = [
    { title: "Approve: edit /etc/hosts", agent: "claude", unread: true, time: "3m" },
    { title: "Bridge delivery failed — slack#alerts", agent: "codex", unread: true, time: "18m" },
    { title: "Skill installed: git-flow-guard", agent: "claude", unread: true, time: "1h" },
    { title: "Session ended: docs rewrite", agent: "claude", unread: false, time: "3h" },
    { title: "Retry succeeded: memory GC policy", agent: "claude", unread: false, time: "5h" },
  ];
  return (
    <div className="scroll-y" style={{ flex: 1, padding: "16px 24px" }}>
      <div className="flex items-center gap-2" style={{ marginBottom: 14 }}>
        <Pills
          value="all"
          onChange={() => {}}
          items={[
            { value: "all", label: "All" },
            { value: "unread", label: "Unread" },
            { value: "approvals", label: "Approvals" },
          ]}
        />
        <span
          className="mono"
          style={{ fontSize: 11, color: "var(--color-text-tertiary)", marginLeft: "auto" }}
        >
          3 unread
        </span>
      </div>
      <div className="card" style={{ padding: 0 }}>
        {items.map((it, i) => (
          <div
            key={i}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 14,
              padding: "14px 18px",
              borderBottom: i < items.length - 1 ? "1px solid var(--color-divider)" : "none",
            }}
          >
            <span className={`dot ${it.unread ? "accent" : ""}`} />
            <div style={{ flex: 1 }}>
              <div
                style={{
                  fontSize: 13,
                  fontWeight: it.unread ? 500 : 400,
                  color: it.unread ? "var(--color-text-primary)" : "var(--color-text-secondary)",
                }}
              >
                {it.title}
              </div>
              <div className="flex items-center gap-2" style={{ marginTop: 4 }}>
                <span className="mono-chip">{it.agent}</span>
                <span
                  className="mono"
                  style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                >
                  {it.time} ago
                </span>
              </div>
            </div>
            <button className="btn btn-ghost" style={{ height: 28, padding: "0 10px" }}>
              Open
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

// ====================== NETWORK ======================
function NetworkPage() {
  const [tab, setTab] = React.useState("channels");
  const [selCh, setSelCh] = React.useState("coord.core");
  const [selPeer, setSelPeer] = React.useState("p1");

  return (
    <>
      <PageHeader
        title="Network"
        icon={Icon.Network}
        count={tab === "channels" ? 6 : 12}
        controls={
          <Pills
            value={tab}
            onChange={setTab}
            items={[
              { value: "channels", label: "Channels" },
              { value: "peers", label: "Peers" },
            ]}
          />
        }
        meta={
          tab === "channels" && (
            <button className="btn btn-accent-outline">
              <Icon.Plus size={13} />
              Channel
            </button>
          )
        }
      />
      <div
        style={{
          padding: 16,
          borderBottom: "1px solid var(--color-divider)",
          display: "grid",
          gridTemplateColumns: "repeat(4, 1fr)",
          gap: 12,
        }}
      >
        <Metric label="Channels" value="6" detail="active" />
        <Metric label="Peers" value="12" tone="accent" />
        <Metric label="Messages" value="1,284" detail="24h" />
        <Metric label="Protocol" value="v0" detail="agh-network" />
      </div>
      {tab === "channels" ? (
        <div className="split">
          <div className="split-list">
            <SearchInput placeholder="Filter channels…" onChange={() => {}} />
            <div className="scroll-y" style={{ flex: 1 }}>
              {[
                { id: "coord.core", peers: 4, last: "2m" },
                { id: "agh.ops.alerts", peers: 3, last: "7m" },
                { id: "research.swarm", peers: 6, last: "12m" },
                { id: "automation.triggers", peers: 2, last: "1h" },
                { id: "bridge.observers", peers: 5, last: "3h" },
                { id: "compozy.dev", peers: 8, last: "5h" },
              ].map(ch => (
                <div
                  key={ch.id}
                  className={`list-row ${selCh === ch.id ? "active" : ""}`}
                  onClick={() => setSelCh(ch.id)}
                >
                  <Icon.Hash
                    size={14}
                    style={{ color: "var(--color-text-tertiary)", marginTop: 1 }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div
                      className="mono truncate"
                      style={{ fontSize: 13, color: "var(--color-text-primary)" }}
                    >
                      {ch.id}
                    </div>
                    <div className="flex items-center gap-3" style={{ marginTop: 4 }}>
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        {ch.peers} peers
                      </span>
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        · {ch.last}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="split-detail">
            <ChannelDetail name={selCh} />
          </div>
        </div>
      ) : (
        <div className="split">
          <div className="split-list">
            <SearchInput placeholder="Filter peers…" onChange={() => {}} />
            <div className="scroll-y" style={{ flex: 1 }}>
              {[
                { id: "p1", name: "claude@laptop", status: "online", kind: "local" },
                { id: "p2", name: "codex@laptop", status: "online", kind: "local" },
                { id: "p3", name: "research-box.local", status: "online", kind: "peer" },
                { id: "p4", name: "ci-runner-02", status: "idle", kind: "peer" },
                { id: "p5", name: "ops.agh.sh", status: "offline", kind: "peer" },
              ].map(p => (
                <div
                  key={p.id}
                  className={`list-row ${selPeer === p.id ? "active" : ""}`}
                  onClick={() => setSelPeer(p.id)}
                >
                  <div style={{ flex: 1 }}>
                    <div className="flex items-center gap-2" style={{ marginBottom: 4 }}>
                      <span
                        className={`dot ${p.status === "online" ? "success" : p.status === "idle" ? "warning" : ""}`}
                      />
                      <span
                        className="mono"
                        style={{ fontSize: 12, color: "var(--color-text-primary)" }}
                      >
                        {p.name}
                      </span>
                    </div>
                    <span className="mono-chip">{p.kind}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="split-detail">
            <PeerDetail />
          </div>
        </div>
      )}
    </>
  );
}

function ChannelDetail({ name }) {
  const msgs = [
    {
      kind: "greet",
      from: "claude@laptop",
      body: "wake; capabilities: code, shell",
      time: "12:04:31",
    },
    {
      kind: "say",
      from: "codex@laptop",
      body: "echo ready; peering on coord.core",
      time: "12:04:33",
    },
    {
      kind: "direct",
      from: "claude@laptop",
      body: "→ codex: delegate(compile, workspace=agh-core)",
      time: "12:05:11",
    },
    { kind: "receipt", from: "codex@laptop", body: "ack direct#8471 ok 212ms", time: "12:05:14" },
    {
      kind: "recipe",
      from: "research-box",
      body: "advertise recipe: rag.embed.bulk v2",
      time: "12:06:40",
    },
    {
      kind: "trace",
      from: "claude@laptop",
      body: "span(delegate)→span(recipe.run)→span(receipt)",
      time: "12:07:02",
    },
  ];
  const kindTone = {
    greet: "",
    say: "info",
    direct: "accent",
    receipt: "success",
    recipe: "warning",
    trace: "info",
    whois: "info",
  };
  return (
    <div className="scroll-y" style={{ padding: 24 }}>
      <div className="flex items-center gap-3">
        <Icon.Hash size={18} style={{ color: "var(--color-text-secondary)" }} />
        <h1 style={{ margin: 0, fontSize: 20, fontWeight: 500 }} className="mono">
          {name}
        </h1>
        <span className="mono-chip success" style={{ marginLeft: "auto" }}>
          active
        </span>
      </div>
      <p style={{ marginTop: 8, fontSize: 13, color: "var(--color-text-secondary)" }}>
        4 peers · 312 messages · agh-network/v0
      </p>
      <div className="flex gap-2" style={{ marginTop: 14 }}>
        <button className="btn btn-ghost">
          <Icon.Users size={12} />
          Members
        </button>
        <button className="btn btn-ghost">
          <Icon.Copy size={12} />
          Copy name
        </button>
      </div>
      <Section
        label="Wire trace"
        right={
          <span className="mono" style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}>
            last 6
          </span>
        }
      >
        <div
          style={{
            margin: "0 16px 24px",
            background: "var(--color-canvas-deep)",
            border: "1px solid var(--color-divider)",
            borderRadius: 10,
            padding: 4,
          }}
        >
          {msgs.map((m, i) => (
            <div
              key={i}
              style={{
                display: "grid",
                gridTemplateColumns: "78px 72px 160px 1fr",
                gap: 12,
                padding: "10px 14px",
                borderBottom: i < msgs.length - 1 ? "1px solid var(--color-divider)" : "none",
                alignItems: "center",
              }}
            >
              <span className="mono" style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}>
                {m.time}
              </span>
              <span className={`mono-chip ${kindTone[m.kind]}`}>{m.kind}</span>
              <span className="mono" style={{ fontSize: 11, color: "var(--color-text-secondary)" }}>
                {m.from}
              </span>
              <span className="mono" style={{ fontSize: 11, color: "var(--color-text-primary)" }}>
                {m.body}
              </span>
            </div>
          ))}
        </div>
      </Section>
    </div>
  );
}

function PeerDetail() {
  return (
    <div className="scroll-y" style={{ padding: 24 }}>
      <div className="flex items-center gap-3">
        <span className="dot success" />
        <h1 className="mono" style={{ margin: 0, fontSize: 20, fontWeight: 500 }}>
          claude@laptop
        </h1>
        <span className="mono-chip accent" style={{ marginLeft: "auto" }}>
          local
        </span>
      </div>
      <p style={{ marginTop: 8, fontSize: 13, color: "var(--color-text-secondary)" }}>
        Joined 2h ago · 4 channels · 128 messages produced
      </p>
      <div
        style={{ marginTop: 20, display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 12 }}
      >
        <Metric label="Sent" value="128" />
        <Metric label="Received" value="184" tone="accent" />
        <Metric label="p95 rtt" value="24ms" />
      </div>
      <Section label="Capabilities">
        <div className="flex gap-2" style={{ padding: "0 20px 16px", flexWrap: "wrap" }}>
          {["code", "shell", "file.read", "file.write", "search", "plan.delegate"].map(c => (
            <span key={c} className="mono-chip">
              {c}
            </span>
          ))}
        </div>
      </Section>
      <Section label="Channels">
        <div style={{ padding: "0 16px 16px" }}>
          {["coord.core", "agh.ops.alerts", "research.swarm", "compozy.dev"].map((c, i) => (
            <div
              key={c}
              className="flex items-center gap-3"
              style={{
                padding: "10px 4px",
                borderBottom: i < 3 ? "1px solid var(--color-divider)" : "none",
              }}
            >
              <Icon.Hash size={12} style={{ color: "var(--color-text-tertiary)" }} />
              <span className="mono" style={{ fontSize: 12, color: "var(--color-text-primary)" }}>
                {c}
              </span>
              <span
                className="mono"
                style={{ fontSize: 10, color: "var(--color-text-tertiary)", marginLeft: "auto" }}
              >
                {40 - i * 8} msgs
              </span>
            </div>
          ))}
        </div>
      </Section>
    </div>
  );
}

// ====================== AUTOMATION ======================
function AutomationPage() {
  const [tab, setTab] = React.useState("jobs");
  const [scope, setScope] = React.useState("all");
  const [sel, setSel] = React.useState(0);

  const jobs = [
    { name: "nightly.digest", scope: "global", schedule: "0 2 * * *", last: "8h ago", state: "ok" },
    {
      name: "memory.gc",
      scope: "workspace",
      schedule: "*/30 * * * *",
      last: "14m ago",
      state: "ok",
    },
    {
      name: "bridge.healthcheck",
      scope: "global",
      schedule: "*/5 * * * *",
      last: "2m ago",
      state: "failed",
    },
    {
      name: "docs.reindex",
      scope: "workspace",
      schedule: "0 */4 * * *",
      last: "1h ago",
      state: "ok",
    },
  ];
  const triggers = [
    { name: "on.session.approved", scope: "workspace", hook: "session.*.approved", last: "6m ago" },
    { name: "on.bridge.error", scope: "global", hook: "bridge.*.error", last: "32m ago" },
  ];
  const items = tab === "jobs" ? jobs : triggers;
  const s = items[sel] || items[0];

  return (
    <>
      <PageHeader
        title="Automation"
        icon={Icon.Zap}
        count={items.length}
        controls={
          <div className="flex items-center gap-2">
            <Pills
              value={tab}
              onChange={setTab}
              items={[
                { value: "jobs", label: "Jobs" },
                { value: "triggers", label: "Triggers" },
              ]}
            />
            <Pills
              value={scope}
              onChange={setScope}
              items={[
                { value: "all", label: "All" },
                { value: "global", label: "Global" },
                { value: "workspace", label: "Workspace" },
              ]}
            />
          </div>
        }
        meta={
          <button className="btn btn-ghost">
            <Icon.Plus size={13} />
            {tab === "jobs" ? "Job" : "Trigger"}
          </button>
        }
      />
      <div className="split">
        <div className="split-list">
          <SearchInput placeholder={`Filter ${tab}…`} onChange={() => {}} />
          <div className="scroll-y" style={{ flex: 1 }}>
            {items.map((it, i) => (
              <div
                key={it.name}
                className={`list-row ${sel === i ? "active" : ""}`}
                onClick={() => setSel(i)}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div className="flex items-center gap-2" style={{ marginBottom: 6 }}>
                    {it.state === "failed" ? (
                      <span className="dot danger" />
                    ) : (
                      <span className="dot success" />
                    )}
                    <span
                      className="mono truncate"
                      style={{ fontSize: 13, color: "var(--color-text-primary)" }}
                    >
                      {it.name}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="mono-chip">{it.scope}</span>
                    {it.schedule && (
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        {it.schedule}
                      </span>
                    )}
                    {it.hook && (
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        {it.hook}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="split-detail">
          <div className="scroll-y" style={{ padding: 24 }}>
            <div className="flex items-center gap-3">
              <Icon.Zap size={18} style={{ color: "var(--color-text-secondary)" }} />
              <h1 className="mono" style={{ margin: 0, fontSize: 20, fontWeight: 500 }}>
                {s.name}
              </h1>
              <span
                className={`mono-chip ${s.state === "failed" ? "danger" : "success"}`}
                style={{ marginLeft: "auto" }}
              >
                {s.state || "active"}
              </span>
            </div>
            <div className="flex gap-2" style={{ marginTop: 10 }}>
              <span className="mono-chip">{s.scope}</span>
              {s.schedule && <span className="mono-chip">cron · {s.schedule}</span>}
              {s.hook && <span className="mono-chip">hook · {s.hook}</span>}
              <span className="mono-chip">last · {s.last}</span>
            </div>
            <div className="flex gap-2" style={{ marginTop: 18 }}>
              <button className="btn btn-primary">
                <Icon.Play size={12} />
                Run now
              </button>
              <button className="btn btn-ghost">
                <Icon.Pause size={12} />
                Pause
              </button>
              <button className="btn btn-ghost">
                <Icon.FileCode size={12} />
                Edit
              </button>
            </div>
            <div
              style={{
                marginTop: 24,
                display: "grid",
                gridTemplateColumns: "repeat(3, 1fr)",
                gap: 12,
              }}
            >
              <Metric label="Runs 24h" value="48" />
              <Metric label="Success" value="96%" tone="accent" />
              <Metric label="Avg duration" value="1.8s" />
            </div>
            <Section label="Run history">
              <div style={{ padding: "0 16px 20px" }}>
                {[0, 1, 2, 3, 4].map(i => (
                  <div
                    key={i}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "14px 120px 1fr 80px 60px",
                      gap: 14,
                      padding: "10px 4px",
                      borderBottom: i < 4 ? "1px solid var(--color-divider)" : "none",
                      alignItems: "center",
                    }}
                  >
                    <span className={`dot ${i === 2 ? "danger" : "success"}`} />
                    <span
                      className="mono"
                      style={{ fontSize: 11, color: "var(--color-text-primary)" }}
                    >
                      run-{1200 - i}
                    </span>
                    <span
                      className="mono"
                      style={{ fontSize: 11, color: "var(--color-text-secondary)" }}
                    >
                      {i === 2 ? "timeout waiting upstream" : "completed"}
                    </span>
                    <span
                      className="mono"
                      style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}
                    >
                      {(1.2 + i * 0.3).toFixed(1)}s
                    </span>
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      {i * 14 + 3}m
                    </span>
                  </div>
                ))}
              </div>
            </Section>
          </div>
        </div>
      </div>
    </>
  );
}

// ====================== BRIDGES ======================
function BridgesPage() {
  const [scope, setScope] = React.useState("all");
  const [sel, setSel] = React.useState(0);
  const bridges = [
    {
      name: "slack-prod",
      provider: "slack",
      scope: "global",
      status: "healthy",
      events: 1284,
      icon: Icon.Slack,
    },
    {
      name: "ops-email",
      provider: "email",
      scope: "global",
      status: "healthy",
      events: 212,
      icon: Icon.Mail,
    },
    {
      name: "linear-ops",
      provider: "linear",
      scope: "workspace",
      status: "degraded",
      events: 48,
      icon: Icon.ExternalLink,
    },
    {
      name: "gh-repo-sync",
      provider: "github",
      scope: "workspace",
      status: "healthy",
      events: 612,
      icon: Icon.GitBranch,
    },
  ];
  const s = bridges[sel];

  return (
    <>
      <PageHeader
        title="Bridges"
        icon={Icon.Waypoints}
        count={bridges.length}
        controls={
          <Pills
            value={scope}
            onChange={setScope}
            items={[
              { value: "all", label: "All" },
              { value: "global", label: "Global" },
              { value: "workspace", label: "Workspace" },
            ]}
          />
        }
        meta={
          <button className="btn btn-primary">
            <Icon.Plus size={13} />
            Bridge
          </button>
        }
      />
      <div className="split">
        <div className="split-list">
          <SearchInput placeholder="Filter bridges…" onChange={() => {}} />
          <div className="scroll-y" style={{ flex: 1 }}>
            {bridges.map((b, i) => (
              <div
                key={b.name}
                className={`list-row ${sel === i ? "active" : ""}`}
                onClick={() => setSel(i)}
              >
                <div
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 7,
                    background: "var(--color-surface-elevated)",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "var(--color-text-secondary)",
                    flexShrink: 0,
                  }}
                >
                  <b.icon size={14} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      fontSize: 13,
                      fontWeight: 500,
                      color: "var(--color-text-primary)",
                      marginBottom: 4,
                    }}
                  >
                    {b.name}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="mono-chip">{b.provider}</span>
                    <span className={`dot ${b.status === "healthy" ? "success" : "warning"}`} />
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      {b.events}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="split-detail">
          <div className="scroll-y" style={{ padding: 24 }}>
            <div className="flex items-center gap-3">
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 10,
                  background: "var(--color-surface-elevated)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  color: "var(--color-accent)",
                }}
              >
                <s.icon size={18} />
              </div>
              <div>
                <h1 style={{ margin: 0, fontSize: 20, fontWeight: 500 }}>{s.name}</h1>
                <span
                  className="mono"
                  style={{ fontSize: 12, color: "var(--color-text-tertiary)" }}
                >
                  {s.provider} · {s.scope}
                </span>
              </div>
              <span
                className={`mono-chip ${s.status === "healthy" ? "success" : "warning"}`}
                style={{ marginLeft: "auto" }}
              >
                {s.status}
              </span>
            </div>
            <div className="flex gap-2" style={{ marginTop: 18 }}>
              <button className="btn btn-primary">
                <Icon.Send size={12} />
                Test delivery
              </button>
              <button className="btn btn-ghost">
                <Icon.FileCode size={12} />
                Edit
              </button>
              <button className="btn btn-ghost">
                <Icon.Pause size={12} />
                Disable
              </button>
            </div>
            <div
              style={{
                marginTop: 22,
                display: "grid",
                gridTemplateColumns: "repeat(4, 1fr)",
                gap: 12,
              }}
            >
              <Metric label="Delivered" value={s.events.toLocaleString()} detail="24h" />
              <Metric label="Success" value="99.1%" tone="accent" />
              <Metric label="p95 latency" value="420ms" />
              <Metric label="Retries" value="8" />
            </div>
            <Section label="Event stream">
              <div style={{ padding: "0 16px 20px" }}>
                {[0, 1, 2, 3, 4].map(i => (
                  <div
                    key={i}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "14px 90px 1fr 80px",
                      gap: 14,
                      padding: "10px 4px",
                      borderBottom: i < 4 ? "1px solid var(--color-divider)" : "none",
                      alignItems: "center",
                    }}
                  >
                    <span className={`dot ${i === 2 ? "warning" : "success"}`} />
                    <span className="mono-chip" style={{ background: "transparent" }}>
                      {
                        [
                          "session.done",
                          "bridge.delivered",
                          "bridge.delivered",
                          "bridge.retry",
                          "session.approved",
                        ][i]
                      }
                    </span>
                    <span
                      className="mono truncate"
                      style={{ fontSize: 11, color: "var(--color-text-secondary)" }}
                    >
                      ws={["agh-core", "compozy", "agh-core", "agh-core", "compozy"][i]} · channel=#
                      {["ops", "ops", "alerts", "alerts", "ops"][i]}
                    </span>
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      {i * 5 + 1}m ago
                    </span>
                  </div>
                ))}
              </div>
            </Section>
          </div>
        </div>
      </div>
    </>
  );
}

// ====================== KNOWLEDGE ======================
function KnowledgePage() {
  const [tab, setTab] = React.useState("all");
  const [sel, setSel] = React.useState(0);
  const memories = [
    { name: "agh-architecture.md", scope: "workspace", size: "12.4 KB", updated: "2h" },
    { name: "wire-protocol-v0.md", scope: "global", size: "8.1 KB", updated: "1d" },
    { name: "operator-voice.md", scope: "global", size: "3.2 KB", updated: "3d" },
    { name: "sessions-model.md", scope: "workspace", size: "6.8 KB", updated: "1w" },
    { name: "bridge-conventions.md", scope: "workspace", size: "5.1 KB", updated: "1w" },
  ];
  const s = memories[sel];

  return (
    <>
      <PageHeader
        title="Knowledge"
        icon={Icon.Book}
        count={memories.length}
        controls={
          <Pills
            value={tab}
            onChange={setTab}
            items={[
              { value: "all", label: "All" },
              { value: "global", label: "Global" },
              { value: "workspace", label: "Workspace" },
            ]}
          />
        }
        meta={
          <div className="flex items-center gap-2">
            <span className="dot" />
            <span className="mono" style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}>
              dream · idle
            </span>
          </div>
        }
      />
      <div className="split">
        <div className="split-list">
          <SearchInput placeholder="Filter knowledge…" onChange={() => {}} />
          <div className="scroll-y" style={{ flex: 1 }}>
            {memories.map((m, i) => (
              <div
                key={m.name}
                className={`list-row ${sel === i ? "active" : ""}`}
                onClick={() => setSel(i)}
              >
                <Icon.FileCode
                  size={14}
                  style={{ color: "var(--color-text-tertiary)", marginTop: 1 }}
                />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    className="mono truncate"
                    style={{ fontSize: 13, color: "var(--color-text-primary)" }}
                  >
                    {m.name}
                  </div>
                  <div className="flex items-center gap-2" style={{ marginTop: 4 }}>
                    <span className="mono-chip">{m.scope}</span>
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      {m.size}
                    </span>
                    <span
                      className="mono"
                      style={{
                        fontSize: 10,
                        color: "var(--color-text-tertiary)",
                        marginLeft: "auto",
                      }}
                    >
                      {m.updated}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="split-detail">
          <div className="scroll-y" style={{ padding: 24 }}>
            <div className="flex items-center gap-3">
              <Icon.FileCode size={18} style={{ color: "var(--color-text-secondary)" }} />
              <h1 className="mono" style={{ margin: 0, fontSize: 18, fontWeight: 500 }}>
                {s.name}
              </h1>
              <span className="mono-chip" style={{ marginLeft: "auto" }}>
                {s.scope}
              </span>
            </div>
            <div className="flex gap-2" style={{ marginTop: 10 }}>
              <span className="mono-chip">{s.size}</span>
              <span className="mono-chip">updated {s.updated} ago</span>
            </div>
            <div className="flex gap-2" style={{ marginTop: 16 }}>
              <button className="btn btn-ghost">
                <Icon.Copy size={12} />
                Copy
              </button>
              <button className="btn btn-ghost">
                <Icon.FileCode size={12} />
                Open in editor
              </button>
            </div>
            <div
              style={{
                marginTop: 20,
                background: "var(--color-canvas-deep)",
                border: "1px solid var(--color-divider)",
                borderRadius: 10,
                padding: 20,
                fontFamily: "var(--font-mono)",
                fontSize: 12.5,
                lineHeight: 1.75,
                color: "var(--color-text-secondary)",
              }}
            >
              <div style={{ color: "var(--color-text-primary)", fontWeight: 500, marginBottom: 8 }}>
                # {s.name.replace(".md", "")}
              </div>
              <div>AGH is a local-first agent runtime. The operator owns the binary,</div>
              <div>the sessions, and the wire. Everything is replayable; no black boxes.</div>
              <div style={{ marginTop: 10, color: "var(--color-text-tertiary)" }}>
                ## Invariants
              </div>
              <div>- Sessions are durable across restarts.</div>
              <div>- `agh-network/v0` is the only coordination protocol.</div>
              <div>- Memory is scoped: global &gt; workspace &gt; session.</div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

// ====================== SKILLS ======================
function SkillsPage() {
  const [tab, setTab] = React.useState("installed");
  const [sel, setSel] = React.useState(0);

  const installed = [
    {
      name: "git-flow-guard",
      version: "1.2.0",
      author: "compozy",
      enabled: true,
      desc: "Block risky git operations behind approval.",
    },
    {
      name: "rag.embed.bulk",
      version: "2.0.1",
      author: "research",
      enabled: true,
      desc: "Embed a folder tree into workspace memory.",
    },
    {
      name: "shell.safe-run",
      version: "0.4.3",
      author: "compozy",
      enabled: true,
      desc: "Sandbox shell commands with allowlist policies.",
    },
    {
      name: "docs.rewrite",
      version: "0.1.0",
      author: "pedronauck",
      enabled: false,
      desc: "Rewrite markdown to operator voice.",
    },
  ];
  const market = [
    {
      name: "k8s.cluster-audit",
      author: "compozy",
      downloads: "2.1k",
      desc: "Audit a kube cluster for AGH readiness.",
    },
    {
      name: "pg.query-explain",
      author: "community",
      downloads: "840",
      desc: "Explain slow queries with plan excerpts.",
    },
    {
      name: "web.scrape.clean",
      author: "community",
      downloads: "1.2k",
      desc: "Readable text extraction from any URL.",
    },
  ];
  const s = installed[sel];

  return (
    <>
      <PageHeader
        title="Skills"
        icon={Icon.Wrench}
        count={tab === "installed" ? installed.length : market.length}
        controls={
          <Pills
            value={tab}
            onChange={setTab}
            items={[
              { value: "installed", label: "Installed" },
              { value: "marketplace", label: "Marketplace" },
            ]}
          />
        }
      />
      {tab === "installed" ? (
        <div className="split">
          <div className="split-list">
            <SearchInput placeholder="Filter skills…" onChange={() => {}} />
            <div className="scroll-y" style={{ flex: 1 }}>
              {installed.map((sk, i) => (
                <div
                  key={sk.name}
                  className={`list-row ${sel === i ? "active" : ""}`}
                  onClick={() => setSel(i)}
                >
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: 7,
                      background: "var(--color-surface-elevated)",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      color: sk.enabled ? "var(--color-accent)" : "var(--color-text-tertiary)",
                      flexShrink: 0,
                    }}
                  >
                    <Icon.Wrench size={13} />
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className="flex items-center gap-2" style={{ marginBottom: 4 }}>
                      <span
                        className="mono truncate"
                        style={{ fontSize: 13, color: "var(--color-text-primary)" }}
                      >
                        {sk.name}
                      </span>
                      {!sk.enabled && <span className="mono-chip">off</span>}
                    </div>
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      v{sk.version} · {sk.author}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="split-detail">
            <div className="scroll-y" style={{ padding: 24 }}>
              <div className="flex items-center gap-3">
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: 10,
                    background: "var(--color-surface-elevated)",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "var(--color-accent)",
                  }}
                >
                  <Icon.Wrench size={17} />
                </div>
                <div>
                  <h1 className="mono" style={{ margin: 0, fontSize: 18, fontWeight: 500 }}>
                    {s.name}
                  </h1>
                  <span
                    className="mono"
                    style={{ fontSize: 11, color: "var(--color-text-tertiary)" }}
                  >
                    v{s.version} · {s.author}
                  </span>
                </div>
                <span
                  className={`mono-chip ${s.enabled ? "success" : ""}`}
                  style={{ marginLeft: "auto" }}
                >
                  {s.enabled ? "enabled" : "disabled"}
                </span>
              </div>
              <p
                style={{
                  marginTop: 16,
                  color: "var(--color-text-secondary)",
                  fontSize: 14,
                  lineHeight: 1.7,
                  maxWidth: 560,
                }}
              >
                {s.desc}
              </p>
              <div className="flex gap-2" style={{ marginTop: 16 }}>
                <button className="btn btn-primary">{s.enabled ? "Disable" : "Enable"}</button>
                <button className="btn btn-ghost">
                  <Icon.FileCode size={12} />
                  View SKILL.md
                </button>
                <button className="btn btn-ghost">Uninstall</button>
              </div>
              <Section label="Capabilities">
                <div className="flex gap-2" style={{ padding: "0 20px 16px", flexWrap: "wrap" }}>
                  {["shell.run", "git.stage", "git.commit", "policy.deny"].map(c => (
                    <span key={c} className="mono-chip">
                      {c}
                    </span>
                  ))}
                </div>
              </Section>
              <Section label="Recent calls">
                <div style={{ padding: "0 16px 20px" }}>
                  {[0, 1, 2].map(i => (
                    <div
                      key={i}
                      style={{
                        display: "grid",
                        gridTemplateColumns: "14px 1fr 80px",
                        gap: 14,
                        padding: "10px 4px",
                        borderBottom: i < 2 ? "1px solid var(--color-divider)" : "none",
                        alignItems: "center",
                      }}
                    >
                      <span className="dot success" />
                      <span
                        className="mono"
                        style={{ fontSize: 11, color: "var(--color-text-secondary)" }}
                      >
                        {
                          [
                            "git-flow-guard.check(branch=main)",
                            "git-flow-guard.approve",
                            "git-flow-guard.check(branch=feat/x)",
                          ][i]
                        }
                      </span>
                      <span
                        className="mono"
                        style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                      >
                        {i * 12 + 3}m ago
                      </span>
                    </div>
                  ))}
                </div>
              </Section>
            </div>
          </div>
        </div>
      ) : (
        <div className="scroll-y" style={{ flex: 1, padding: 24 }}>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(320px, 1fr))",
              gap: 14,
            }}
          >
            {market.map(m => (
              <div key={m.name} className="card">
                <div className="flex items-center gap-3" style={{ marginBottom: 12 }}>
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      borderRadius: 9,
                      background: "var(--color-surface-elevated)",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      color: "var(--color-accent)",
                    }}
                  >
                    <Icon.Wrench size={15} />
                  </div>
                  <div style={{ flex: 1 }}>
                    <div
                      className="mono"
                      style={{ fontSize: 13, color: "var(--color-text-primary)" }}
                    >
                      {m.name}
                    </div>
                    <span
                      className="mono"
                      style={{ fontSize: 10, color: "var(--color-text-tertiary)" }}
                    >
                      {m.author} · {m.downloads} installs
                    </span>
                  </div>
                </div>
                <p
                  style={{
                    margin: 0,
                    fontSize: 13,
                    color: "var(--color-text-secondary)",
                    lineHeight: 1.55,
                    minHeight: 44,
                  }}
                >
                  {m.desc}
                </p>
                <div className="flex gap-2" style={{ marginTop: 14 }}>
                  <button
                    className="btn btn-accent-outline"
                    style={{ height: 28, padding: "0 10px", fontSize: 12 }}
                  >
                    Install
                  </button>
                  <button
                    className="btn btn-ghost"
                    style={{ height: 28, padding: "0 10px", fontSize: 12 }}
                  >
                    Details
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  );
}

Object.assign(window, {
  TasksPage,
  NetworkPage,
  AutomationPage,
  BridgesPage,
  KnowledgePage,
  SkillsPage,
});
