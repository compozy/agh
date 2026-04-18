// Sidebar — icon rail + panel with agents and nav
const { Icon } = window;

function Sidebar({ route, setRoute, sessionId, setSessionId }) {
  const [ws, setWs] = React.useState("A");
  const [expanded, setExpanded] = React.useState({ claude: true, codex: true, gemini: false });

  const workspaces = [
    { id: "A", name: "agh-core" },
    { id: "C", name: "compozy" },
    { id: "R", name: "research" },
  ];

  const agents = [
    {
      key: "claude",
      name: "claude",
      provider: "Anthropic",
      sessions: [
        { id: "c-1", name: "refactor tokens", state: "active" },
        { id: "c-2", name: "investigate streaming", state: "starting" },
        { id: "c-3", name: "docs rewrite", state: "stopped" },
      ],
    },
    {
      key: "codex",
      name: "codex",
      provider: "OpenAI",
      sessions: [{ id: "x-1", name: "perf audit", state: "active" }],
    },
    { key: "gemini", name: "gemini", provider: "Google", sessions: [] },
    { key: "opencode", name: "opencode", provider: "OpenCode", sessions: [] },
    { key: "cursor", name: "cursor", provider: "Cursor", sessions: [] },
  ];

  const NAV_ITEMS = [
    { to: "tasks", icon: Icon.ListChecks, label: "Tasks" },
    { to: "automation", icon: Icon.Zap, label: "Automation" },
    { to: "bridges", icon: Icon.Waypoints, label: "Bridges" },
    { to: "network", icon: Icon.Network, label: "Network" },
    { to: "knowledge", icon: Icon.Book, label: "Knowledge" },
    { to: "skills", icon: Icon.Wrench, label: "Skills" },
  ];

  return (
    <aside className="sidebar">
      <div className="icon-rail">
        <div className="rail-logo" title="AGH">
          <svg
            width="18"
            height="18"
            viewBox="0 0 167 167"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <path
              d="M84.1904 68.3481H53.7129C44.2578 68.3481 36.1606 71.7178 29.4214 78.457C22.6821 85.1963 19.3125 93.2935 19.3125 102.749C19.3125 109.287 19.3125 112.556 19.3125 112.556C19.3125 121.709 22.6821 129.806 29.4214 136.847C36.4624 143.586 44.5596 146.956 53.7129 146.956H112.405C121.659 146.956 129.806 143.586 136.847 136.847C143.486 129.806 146.805 121.709 146.805 112.556C146.805 112.556 146.805 92.9917 146.805 53.8638C146.805 44.6099 143.486 36.4624 136.847 29.4214C129.706 22.7827 121.558 19.4634 112.405 19.4634H19.4634C16.7476 19.4634 14.4341 18.5078 12.5229 16.5967C10.7124 14.6855 9.80713 12.3721 9.80713 9.65625C9.80713 7.04102 10.7124 4.77783 12.5229 2.8667C14.4341 0.955566 16.7476 0 19.4634 0H112.405C126.688 0 139.412 5.23047 150.577 15.6914C161.038 26.8564 166.269 39.5806 166.269 53.8638C166.269 53.8638 166.269 73.4277 166.269 112.556C166.269 126.738 161.038 139.412 150.577 150.577C139.412 161.038 126.688 166.269 112.405 166.269H53.7129C39.5303 166.269 26.8564 161.038 15.6914 150.577C5.23047 139.412 0 126.738 0 112.556C0 112.556 0 109.287 0 102.749C0 92.8911 2.41406 83.8887 7.24219 75.7412C12.0703 67.5938 18.5581 61.106 26.7056 56.2778C34.853 51.4497 43.8555 49.0356 53.7129 49.0356H84.0396C85.3472 46.0181 87.208 43.3022 89.6221 40.8882C94.5508 35.9595 100.485 33.4951 107.426 33.4951C114.366 33.4951 120.301 35.9595 125.229 40.8882C130.158 45.8169 132.623 51.7515 132.623 58.6919C132.623 65.6323 130.158 71.5669 125.229 76.4956C120.301 81.4243 114.366 83.8887 107.426 83.8887C100.485 83.8887 94.5508 81.4243 89.6221 76.4956C87.208 73.981 85.3975 71.2651 84.1904 68.3481ZM83.436 50.8462C82.6313 53.3608 82.229 55.9761 82.229 58.6919C82.229 61.5083 82.6313 64.1738 83.436 66.6885H53.7129C43.7549 66.6885 35.2554 70.209 28.2144 77.25C21.0728 84.291 17.502 92.7905 17.502 102.749V112.556C17.502 122.111 21.0728 130.611 28.2144 138.054C35.6577 145.196 44.1572 148.767 53.7129 148.767C53.7129 148.767 73.2769 148.767 112.405 148.767C122.061 148.767 130.611 145.196 138.054 138.054C145.095 130.611 148.616 122.111 148.616 112.556V53.8638C148.616 44.2075 145.095 35.6577 138.054 28.2144C130.611 21.1733 122.061 17.6528 112.405 17.6528C112.405 17.6528 81.4243 17.6528 19.4634 17.6528C17.2505 17.6528 15.3896 16.8984 13.8809 15.3896C12.3721 13.7803 11.6177 11.8691 11.6177 9.65625C11.6177 7.54395 12.3721 5.68311 13.8809 4.07373C15.3896 2.56494 17.2505 1.81055 19.4634 1.81055C19.4634 1.81055 50.4438 1.81055 112.405 1.81055C126.286 1.81055 138.557 6.83984 149.219 16.8984H149.37C159.429 27.6611 164.458 39.9829 164.458 53.8638V112.556C164.458 126.336 159.429 138.557 149.37 149.219V149.37C138.607 159.429 126.286 164.458 112.405 164.458C112.405 164.458 92.8408 164.458 53.7129 164.458C39.9326 164.458 27.7114 159.429 17.0493 149.37H16.8984C6.83984 138.607 1.81055 126.336 1.81055 112.556V102.749C1.81055 93.1929 4.12402 84.4922 8.75098 76.6465C13.4785 68.7002 19.7651 62.4136 27.6108 57.7866C35.4565 53.1597 44.1572 50.8462 53.7129 50.8462H83.436ZM86.001 68.3481C87.208 70.8628 88.8174 73.1763 90.8291 75.2886C95.4561 79.8149 100.988 82.0781 107.426 82.0781C113.863 82.0781 119.396 79.8149 124.022 75.2886C128.649 70.6616 130.963 65.1294 130.963 58.6919C130.963 52.2544 128.649 46.7222 124.022 42.0952C119.396 37.4683 113.863 35.1548 107.426 35.1548C100.988 35.1548 95.4561 37.4683 90.8291 42.0952C88.7168 44.2075 87.1074 46.521 86.001 49.0356H88.1133C90.7285 49.0356 92.9917 49.9912 94.9028 51.9023C96.814 53.7129 97.7695 55.9761 97.7695 58.6919C97.7695 61.3071 96.814 63.5703 94.9028 65.4814C92.9917 67.3926 90.7285 68.3481 88.1133 68.3481H86.001ZM85.2466 50.8462C84.4419 53.2603 84.0396 55.8755 84.0396 58.6919C84.0396 61.5083 84.4922 64.1738 85.3975 66.6885H88.1133C90.2256 66.6885 92.0361 65.8838 93.5449 64.2744C95.1543 62.7656 95.959 60.9048 95.959 58.6919C95.959 56.479 95.1543 54.6182 93.5449 53.1094C92.0361 51.6006 90.2256 50.8462 88.1133 50.8462H85.2466Z"
              fill="currentColor"
            />
          </svg>
        </div>
        {workspaces.map(w => (
          <button
            key={w.id}
            className={`rail-ws ${ws === w.id ? "active" : ""}`}
            onClick={() => setWs(w.id)}
            title={w.name}
          >
            {w.id}
          </button>
        ))}
        <button className="rail-ws" style={{ background: "transparent", borderStyle: "dashed" }}>
          <Icon.Plus size={12} />
        </button>
      </div>

      <div className="sidebar-panel">
        <div className="sidebar-head">
          <span className="name">{workspaces.find(w => w.id === ws)?.name}</span>
          <button className="btn-icon">
            <Icon.Search size={13} />
          </button>
          <button className="btn-icon">
            <Icon.PanelLeft size={13} />
          </button>
        </div>

        <div className="sidebar-search">
          <div className="search-input" style={{ padding: "5px 8px" }}>
            <Icon.Search size={12} style={{ color: "var(--color-text-tertiary)" }} />
            <input placeholder="Search…" />
            <span className="kbd">⌘K</span>
          </div>
        </div>

        <div className="scroll-y" style={{ flex: 1 }}>
          <div className="sidebar-section-label">Agents</div>
          {agents.map(a => (
            <div key={a.key}>
              <div
                className="agent-row"
                onClick={() => setExpanded(s => ({ ...s, [a.key]: !s[a.key] }))}
              >
                <span style={{ color: "var(--color-text-tertiary)", display: "inline-flex" }}>
                  {expanded[a.key] ? (
                    <Icon.ChevronDown size={12} />
                  ) : (
                    <Icon.ChevronRight size={12} />
                  )}
                </span>
                <AgentGlyph name={a.key} />
                <span className="name">{a.name}</span>
                <span className="count">{a.sessions.length}</span>
                <span className="btn-icon" style={{ width: 18, height: 18 }}>
                  <Icon.Plus size={11} />
                </span>
              </div>
              {expanded[a.key] &&
                a.sessions.map(s => (
                  <div
                    key={s.id}
                    className={`session-row ${route === "session" && sessionId === s.id ? "active" : ""}`}
                    onClick={() => {
                      setRoute("session");
                      setSessionId(s.id);
                    }}
                  >
                    <span className={`dot ${stateToDot(s.state)}`} />
                    <span className="truncate">{s.name}</span>
                  </div>
                ))}
              {expanded[a.key] && a.sessions.length === 0 && (
                <div
                  style={{
                    marginLeft: 22,
                    padding: "3px 8px",
                    fontSize: 11,
                    color: "var(--color-text-tertiary)",
                  }}
                >
                  No sessions
                </div>
              )}
            </div>
          ))}

          <div className="sidebar-section-label" style={{ marginTop: 12 }}>
            Workspace
          </div>
          {NAV_ITEMS.map(item => (
            <div
              key={item.to}
              className={`nav-item ${route === item.to ? "active" : ""}`}
              onClick={() => setRoute(item.to)}
            >
              <item.icon
                size={14}
                style={{
                  color:
                    route === item.to ? "var(--color-text-primary)" : "var(--color-text-tertiary)",
                }}
              />
              <span>{item.label}</span>
            </div>
          ))}
        </div>

        <div className="sidebar-footer">
          <div className="flex items-center gap-2" style={{ marginBottom: 6 }}>
            <span className="dot success pulse" />
            <span className="mono" style={{ fontSize: 10, color: "var(--color-text-secondary)" }}>
              connected
            </span>
            <span
              className="mono"
              style={{ fontSize: 10, color: "var(--color-text-tertiary)", marginLeft: "auto" }}
            >
              v0.4.1
            </span>
          </div>
          <div
            className={`nav-item ${route === "settings" ? "active" : ""}`}
            style={{ margin: 0 }}
            onClick={() => setRoute("settings")}
          >
            <Icon.Settings size={13} />
            <span>Settings</span>
          </div>
        </div>
      </div>
    </aside>
  );
}

function stateToDot(state) {
  if (state === "active") return "success";
  if (state === "starting") return "warning pulse";
  if (state === "stopping") return "warning pulse";
  return "";
}

function AgentGlyph({ name }) {
  const colors = {
    claude: "#E8572A",
    codex: "#8E8E93",
    gemini: "#BF5AF2",
    opencode: "#30D158",
    cursor: "#E5E5E7",
  };
  return (
    <span
      style={{
        width: 14,
        height: 14,
        borderRadius: 4,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        background: "color-mix(in srgb, " + (colors[name] || "#8E8E93") + " 18%, transparent)",
        color: colors[name] || "#8E8E93",
        fontFamily: "var(--font-mono)",
        fontSize: 9,
        fontWeight: 700,
      }}
    >
      {name[0].toUpperCase()}
    </span>
  );
}

window.Sidebar = Sidebar;
