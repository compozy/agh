/* V2 — linear-panel
 * Workspace inside a lifted product-frame. Sub-panels carry the hierarchy
 * via header gradient overlays and inset-top highlights. Selected rows
 * tint accent (4%) with a 2px rail. Detail panel is a full Linear-grade
 * inspector: properties → sub-issues with progress → activity feed →
 * comment composer → actions. */

const { useState, Fragment } = React;
const I = window.AGH_ICONS;
const D = window.AGH_DATA;

/* ── Avatar ──────────────────────────────────────────────────────────── */

function Avatar({ owner, size = 22 }) {
  if (!owner) return null;
  const isAgent = owner.kind === "agent";
  const isSystem = owner.kind === "system";
  const colors = D.avatarColors(owner);
  const initials = D.avatarInitials(owner.label);
  const cls = isSystem
    ? "lp-avatar lp-avatar--system"
    : isAgent
    ? "lp-avatar lp-avatar--agent"
    : "lp-avatar";
  return (
    <span
      className={cls}
      style={{
        width: size,
        height: size,
        background: isSystem ? undefined : colors.bg,
        color: isSystem ? undefined : colors.fg,
        fontSize: Math.max(9, Math.round(size * 0.42)),
      }}
      title={`${owner.kind} · ${owner.label}`}
      aria-label={`${owner.kind} ${owner.label}`}
    >
      {isSystem ? <I.Workspace size={Math.round(size * 0.6)} stroke={1.5} /> : initials}
    </span>
  );
}

function StatusGlyph({ status, size = 14 }) {
  return (
    <span className="lp-row__statusicon">
      <I.Status status={status} size={size} />
    </span>
  );
}

function PriorityGlyph({ level }) {
  if (!level) return null;
  return (
    <span className={`lp-row__priority lp-row__priority--${level}`} title={`${level} priority`}>
      <I.Priority level={level} size={12} />
    </span>
  );
}

function PropertyPill({ icon, leading, tone = "neutral", mono = false, onClick, children }) {
  const Tag = onClick ? "button" : "span";
  const cls = [
    "lp-prop",
    tone !== "neutral" ? `lp-prop--${tone}` : null,
    mono ? "lp-prop--mono" : null,
    onClick ? "lp-prop--button" : null,
  ]
    .filter(Boolean)
    .join(" ");
  return (
    <Tag type={onClick ? "button" : undefined} onClick={onClick} className={cls}>
      {leading ? <span className="lp-prop__icon">{leading}</span> : null}
      {icon ? <span className="lp-prop__icon">{icon}</span> : null}
      <span>{children}</span>
    </Tag>
  );
}

/* ── Sidebar ─────────────────────────────────────────────────────────── */

function WorkspaceRail() {
  return (
    <aside className="lp-rail" aria-label="Workspaces">
      <div className="lp-rail__brand" title="AGH">
        a
      </div>
      {D.WORKSPACES.filter((w) => !w.active).map((ws) => (
        <button key={ws.id} type="button" className="lp-rail__ws" title={ws.label}>
          {ws.initial}
        </button>
      ))}
      <button type="button" className="lp-rail__plus" aria-label="Add workspace">
        <I.Plus size={12} />
      </button>
    </aside>
  );
}

function NavRow({ item, active, onClick, disabled }) {
  const Icon = I[item.icon];
  return (
    <button
      type="button"
      className={`lp-nav__row${active ? " lp-nav__row--active" : ""}`}
      onClick={disabled ? undefined : onClick}
      aria-disabled={disabled || undefined}
      aria-current={active ? "page" : undefined}
    >
      <span className="lp-nav__icon">
        <Icon size={14} />
      </span>
      <span className="lp-nav__label">{item.label}</span>
      {item.count !== null && item.count !== undefined ? (
        <span className="lp-nav__count">{item.count}</span>
      ) : null}
    </button>
  );
}

function Sidebar({ view, setView }) {
  return (
    <aside className="lp-panel lp-side" aria-label="Primary navigation">
      <div className="lp-side__head">
        <span className="lp-side__brand">agh</span>
        <span className="lp-side__alpha">Alpha</span>
      </div>
      <button type="button" className="lp-side__search" aria-label="Open command palette">
        <I.Search size={13} />
        <span>Search</span>
        <span className="lp-side__search-kbd">
          <span className="lp-kbd">⌘</span>
          <span className="lp-kbd">K</span>
        </span>
      </button>
      <div className="lp-side__group">
        <div className="lp-side__label">Workspace</div>
        {D.NAV_PRIMARY.map((item) => (
          <NavRow
            key={item.id}
            item={item}
            active={item.id === view}
            disabled={item.id !== "tasks" && item.id !== "jobs"}
            onClick={() => setView(item.id)}
          />
        ))}
      </div>
      <div className="lp-side__group">
        <div className="lp-side__label">Operations</div>
        {D.NAV_SECONDARY.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="lp-side__group">
        {D.NAV_FOOTER.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="lp-side__footer">
        <span className="lp-side__footer-status">
          <span className="lp-status-dot" />
          <span>Connected</span>
        </span>
        <span>v0.41.0</span>
      </div>
    </aside>
  );
}

/* ── PageHead + filters ──────────────────────────────────────────────── */

function PageHead({ title, count, summary, primary }) {
  return (
    <header className="lp-page-head">
      <div>
        <div className="lp-page-head__title">
          <h1>{title}</h1>
          <span className="lp-page-head__count">{count}</span>
        </div>
        {summary ? <p className="lp-page-head__sub">{summary}</p> : null}
      </div>
      <div className="lp-page-head__primary">{primary}</div>
    </header>
  );
}

function PillGroup({ items, value, onChange }) {
  return (
    <div className="lp-pillgroup" role="tablist">
      {items.map((item) => (
        <button
          key={item.value}
          type="button"
          role="tab"
          aria-selected={item.value === value}
          className={`lp-pillgroup__seg${item.value === value ? " lp-pillgroup__seg--active" : ""}`}
          onClick={() => onChange(item.value)}
        >
          <span>{item.label}</span>
          {item.badge != null ? <span className="lp-pillgroup__badge">{item.badge}</span> : null}
        </button>
      ))}
    </div>
  );
}

/* ── Tasks ───────────────────────────────────────────────────────────── */

function TaskRow({ task, selected, onSelect }) {
  return (
    <div
      className={`lp-row${selected ? " lp-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(task.id)}
    >
      <div className="lp-row__line">
        <StatusGlyph status={task.status} />
        <Avatar owner={task.owner} size={20} />
        <span className="lp-row__name">{task.title}</span>
        <span className="lp-row__cluster">
          <PriorityGlyph level={task.priority} />
          {task.parentTaskId ? (
            <span className="lp-row__sub" title={`parent ${task.parentTaskId}`}>
              <I.Corner size={11} />
              <span>{task.parentTaskId.slice(-6)}</span>
            </span>
          ) : null}
          {task.activeRun ? (
            <span className="lp-row__attempts" title="attempts">
              {task.activeRun.attempt}
              <span className="lp-row__attempts-sep">/</span>
              {task.activeRun.maxAttempts}
            </span>
          ) : null}
          {task.childCount > 0 ? (
            <span className="lp-row__sub" title={`${task.childCount} children`}>
              <I.Boxes size={11} />
              <span>{task.childCount}</span>
            </span>
          ) : null}
          <span className="lp-row__id">{task.id}</span>
          <span className="lp-row__time" title={task.timestampIso}>
            {task.timestamp}
          </span>
        </span>
      </div>

      {task.status === "failed" && task.activeRun?.error ? (
        <p className="lp-row__error">
          <I.Alert size={11} stroke={2} />
          <span>{task.activeRun.error}</span>
        </p>
      ) : null}

      {task.isBlocked || task.approvalState === "pending" || task.isDraft || task.status === "failed" ? (
        <div className="lp-row__sublane">
          {task.isBlocked ? (
            <PropertyPill leading={<I.Alert size={11} />} tone="warning">
              {task.blockedReason ?? "blocked"}
            </PropertyPill>
          ) : null}
          {task.approvalState === "pending" ? (
            <PropertyPill leading={<I.Clock size={11} />} tone="accent">
              Approval pending
            </PropertyPill>
          ) : null}
          {task.isDraft ? (
            <button
              type="button"
              className="lp-row__action"
              onClick={(e) => e.stopPropagation()}
            >
              <I.Plus size={11} /> Publish
            </button>
          ) : null}
          {task.status === "failed" ? (
            <button
              type="button"
              className="lp-row__action"
              onClick={(e) => e.stopPropagation()}
            >
              <I.Refresh size={11} /> Retry
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function TasksView({ selectedId, setSelectedId }) {
  const [lane, setLane] = useState("all");
  const [search, setSearch] = useState("");

  const lanes = [
    { value: "all", label: "All", badge: D.TASK_SUMMARY.open },
    {
      value: "mine",
      label: "Mine",
      badge: D.TASKS.filter(
        (t) => t.owner.kind === "human" && !["completed", "canceled"].includes(t.status)
      ).length,
    },
    { value: "watched", label: "Watched", badge: 2 },
  ];

  const tasks = D.TASKS.filter((t) =>
    lane === "mine" ? t.owner.kind === "human" : lane === "watched" ? false : true
  ).filter((t) => (search ? t.title.toLowerCase().includes(search.toLowerCase()) : true));

  const summary = `${D.TASK_SUMMARY.open} open · ${D.TASK_SUMMARY.running} running · ${D.TASK_SUMMARY.blocked} blocked · ${D.TASK_SUMMARY.failed} failed`;

  return (
    <section className="lp-panel lp-list">
      <PageHead
        title="Tasks"
        count={D.TASK_SUMMARY.total}
        summary={summary}
        primary={
          <>
            <button type="button" className="lp-secondary">
              <I.Filter size={13} />
              Filters
            </button>
            <button type="button" className="lp-cta">
              <I.Plus size={13} />
              New task
              <span className="lp-cta__kbd">N</span>
            </button>
          </>
        }
      />
      <div className="lp-filters">
        <div className="lp-search">
          <I.Search size={13} />
          <input
            placeholder="Filter tasks…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <span className="lp-kbd">/</span>
        </div>
        <PillGroup items={lanes} value={lane} onChange={setLane} />
      </div>
      <div className="lp-list__headline">
        <span>
          All tasks
          <span className="lp-list__headline-num">{tasks.length}</span>
        </span>
        <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
          <span className="lp-kbd">J</span>
          <span className="lp-kbd">K</span>
          <span style={{ marginLeft: 4 }}>navigate</span>
        </span>
      </div>
      <div className="lp-list__rows">
        {tasks.map((task) => (
          <TaskRow
            key={task.id}
            task={task}
            selected={task.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </section>
  );
}

/* ── Jobs ────────────────────────────────────────────────────────────── */

function JobRow({ job, selected, onSelect }) {
  return (
    <div
      className={`lp-row${selected ? " lp-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(job.id)}
    >
      <div className="lp-row__line">
        <span
          className="lp-row__statusicon"
          style={{ color: job.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
        >
          <I.Status status={job.enabled ? "ready" : "canceled"} size={14} />
        </span>
        <span className="lp-row__statusicon">
          <I.Clock size={14} stroke={1.6} />
        </span>
        <span className="lp-row__name">{job.name}</span>
        <span className="lp-row__cluster">
          <span style={{ color: job.source === "dynamic" ? "var(--color-text-secondary)" : "var(--color-text-tertiary)" }}>
            {job.source}
          </span>
          <span style={{ color: job.scope === "workspace" ? "var(--color-text-secondary)" : "var(--color-text-tertiary)" }}>
            {job.scope}
          </span>
          <span className="lp-row__id">{job.id}</span>
          <span className="lp-row__time">next · {job.nextRun}</span>
        </span>
      </div>
      <div className="lp-row__sublane">
        <span style={{ color: "var(--color-text-tertiary)" }}>{job.schedule}</span>
        {job.lastFailure ? (
          <PropertyPill leading={<I.Alert size={11} />} tone="danger">
            {job.lastFailure}
          </PropertyPill>
        ) : null}
      </div>
    </div>
  );
}

function JobsView({ selectedId, setSelectedId }) {
  const [scope, setScope] = useState("all");
  const [search, setSearch] = useState("");
  const scopes = [
    { value: "all", label: "All", badge: D.JOBS.length },
    { value: "global", label: "Global", badge: D.JOBS.filter((j) => j.scope === "global").length },
    {
      value: "workspace",
      label: "Workspace",
      badge: D.JOBS.filter((j) => j.scope === "workspace").length,
    },
  ];
  const jobs = D.JOBS.filter((j) => (scope === "all" ? true : j.scope === scope)).filter((j) =>
    search ? j.name.toLowerCase().includes(search.toLowerCase()) : true
  );

  const summary = `${D.JOB_SUMMARY.active} active · ${D.JOB_SUMMARY.paused} paused · workspace agh-runtime`;

  return (
    <section className="lp-panel lp-list">
      <PageHead
        title="Jobs"
        count={D.JOB_SUMMARY.total}
        summary={summary}
        primary={
          <>
            <button type="button" className="lp-secondary">
              <I.Activity size={13} />
              History
            </button>
            <button type="button" className="lp-cta">
              <I.Plus size={13} />
              New job
              <span className="lp-cta__kbd">N</span>
            </button>
          </>
        }
      />
      <div className="lp-filters">
        <div className="lp-search">
          <I.Search size={13} />
          <input
            placeholder="Search jobs…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <span className="lp-kbd">/</span>
        </div>
        <PillGroup items={scopes} value={scope} onChange={setScope} />
      </div>
      <div className="lp-list__headline">
        <span>
          {jobs.length} visible
          <span className="lp-list__headline-num">/ {D.JOBS.length} total</span>
        </span>
        <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
          <span className="lp-kbd">J</span>
          <span className="lp-kbd">K</span>
          <span style={{ marginLeft: 4 }}>navigate</span>
        </span>
      </div>
      <div className="lp-list__rows">
        {jobs.map((job) => (
          <JobRow
            key={job.id}
            job={job}
            selected={job.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </section>
  );
}

/* ── Detail ──────────────────────────────────────────────────────────── */

function TaskDetail({ task }) {
  if (!task) return <DetailEmpty />;
  const subissues = D.getSubissuesFor(task);
  const completedKids = subissues.filter((s) => s.status === "completed").length;
  const activity = D.getActivityFor(task);

  return (
    <article className="lp-panel lp-detail">
      <div className="lp-detail__head">
        <div className="lp-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>tasks</span>
          <I.ChevronRight size={11} />
          <span>{task.id}</span>
          <span className="lp-detail__crumb-actions">
            <button type="button" className="lp-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
            <button type="button" className="lp-iconbtn" title="Open in new pane">
              <I.ArrowUpRight size={12} />
            </button>
          </span>
        </div>
        <h2 className="lp-detail__title">{task.title}</h2>
        <div className="lp-props">
          <PropertyPill leading={<I.Status status={task.status} size={12} />}>
            {D.STATUS_LABEL[task.status]}
          </PropertyPill>
          {task.priority ? (
            <PropertyPill leading={<I.Priority level={task.priority} size={11} />}>
              {D.PRIORITY_LABEL[task.priority]}
            </PropertyPill>
          ) : null}
          <PropertyPill leading={<Avatar owner={task.owner} size={14} />}>
            {task.owner.label}
          </PropertyPill>
          {task.parentTaskId ? (
            <PropertyPill icon={<I.Corner size={11} />} mono onClick={() => {}}>
              {task.parentTaskId}
            </PropertyPill>
          ) : null}
          {task.approvalState === "pending" ? (
            <PropertyPill leading={<I.Clock size={11} />} tone="accent">
              Approval pending
            </PropertyPill>
          ) : null}
          {task.activeRun ? (
            <PropertyPill mono>
              attempt {task.activeRun.attempt}/{task.activeRun.maxAttempts}
            </PropertyPill>
          ) : null}
          {task.childCount > 0 ? (
            <PropertyPill leading={<I.Boxes size={11} />}>
              {task.childCount} {task.childCount === 1 ? "child" : "children"}
            </PropertyPill>
          ) : null}
        </div>
      </div>

      <div className="lp-detail__body">
        {subissues.length > 0 ? (
          <section className="lp-section">
            <div className="lp-section__head">
              <span className="lp-section__label">Sub-issues</span>
              <span className="lp-section__count">
                {completedKids} / {subissues.length}
              </span>
            </div>
            <div className="lp-subissues">
              <div className="lp-subissues__bar">
                <span>{completedKids} done</span>
                <span className="lp-subissues__progress">
                  <i style={{ width: `${(completedKids / subissues.length) * 100}%` }} />
                </span>
                <span>{subissues.length - completedKids} open</span>
              </div>
              {subissues.map((sub) => (
                <a
                  key={sub.id}
                  href="#"
                  className="lp-subissue"
                  onClick={(e) => e.preventDefault()}
                >
                  <I.Status status={sub.status} size={13} />
                  <span className="lp-subissue__name">{sub.title}</span>
                  <span className="lp-subissue__id">{sub.id.slice(-6)}</span>
                  <Avatar owner={sub.assignee} size={16} />
                </a>
              ))}
            </div>
          </section>
        ) : null}

        <section className="lp-section">
          <div className="lp-section__head">
            <span className="lp-section__label">Activity</span>
            <span className="lp-section__count">{activity.length}</span>
          </div>
          <div className="lp-activity">
            {activity.map((event, idx) => (
              <div key={idx} className="lp-activity__item">
                <span className="lp-activity__avatar">
                  <Avatar owner={event.author} size={22} />
                </span>
                <div>
                  <div className="lp-activity__msg">
                    <strong>{event.author.label}</strong>{" "}
                    {event.msg.charAt(0).toLowerCase() + event.msg.slice(1)}
                  </div>
                  {event.detail ? <div className="lp-activity__detail">{event.detail}</div> : null}
                  <div className="lp-activity__time">{event.time}</div>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="lp-section">
          <div className="lp-section__head">
            <span className="lp-section__label">Comment</span>
          </div>
          <div className="lp-composer">
            <textarea
              className="lp-composer__input"
              placeholder="Add a comment, paste an exception, or @-mention a peer…"
              rows={3}
            />
            <div className="lp-composer__bar">
              <span className="lp-composer__hint">
                <span className="lp-kbd">⌘</span> <span className="lp-kbd">⏎</span> to send
              </span>
              <button type="button" className="lp-cta">
                Comment
              </button>
            </div>
          </div>
        </section>

        <section className="lp-section">
          <div className="lp-section__head">
            <span className="lp-section__label">Actions</span>
          </div>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {task.status === "failed" ? (
              <button className="lp-cta">
                <I.Refresh size={13} />
                Retry run
              </button>
            ) : null}
            {task.isDraft ? (
              <button className="lp-cta">
                <I.Play size={13} />
                Publish task
              </button>
            ) : null}
            <button className="lp-secondary">Open timeline</button>
            <button className="lp-secondary">
              <I.Copy size={12} />
              Copy ID
            </button>
          </div>
        </section>
      </div>
    </article>
  );
}

function JobDetail({ job }) {
  if (!job) return <DetailEmpty />;
  return (
    <article className="lp-panel lp-detail">
      <div className="lp-detail__head">
        <div className="lp-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>jobs</span>
          <I.ChevronRight size={11} />
          <span>{job.id}</span>
          <span className="lp-detail__crumb-actions">
            <button type="button" className="lp-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
          </span>
        </div>
        <h2 className="lp-detail__title">{job.name}</h2>
        <div className="lp-props">
          <PropertyPill leading={<I.Status status={job.enabled ? "ready" : "canceled"} size={12} />}>
            {job.enabled ? "Enabled" : "Disabled"}
          </PropertyPill>
          <PropertyPill leading={<I.Clock size={11} />}>{job.schedule}</PropertyPill>
          <PropertyPill mono>next {job.nextRun}</PropertyPill>
          <PropertyPill mono>avg {job.avgDuration}</PropertyPill>
          <PropertyPill>{job.source}</PropertyPill>
          <PropertyPill>{job.scope}</PropertyPill>
        </div>
      </div>
      <div className="lp-detail__body">
        <section className="lp-section">
          <div className="lp-section__head">
            <span className="lp-section__label">Recent runs</span>
            <span className="lp-section__count">3</span>
          </div>
          <div className="lp-activity">
            {[1, 2, 3].map((n) => (
              <div key={n} className="lp-activity__item">
                <span className="lp-activity__avatar">
                  <Avatar owner={{ kind: "system", label: "agh" }} size={22} />
                </span>
                <div>
                  <div className="lp-activity__msg">
                    <strong>{job.id}</strong> ran successfully
                  </div>
                  <div className="lp-activity__detail">
                    duration {job.avgDuration} · 0 retries · workspace agh-runtime
                  </div>
                  <div className="lp-activity__time">{n === 1 ? job.lastRun : `${n - 1}h before that`}</div>
                </div>
              </div>
            ))}
          </div>
        </section>
        <section className="lp-section">
          <div className="lp-section__head">
            <span className="lp-section__label">Actions</span>
          </div>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <button className="lp-cta">
              <I.Play size={13} />
              {job.enabled ? "Pause" : "Enable"}
            </button>
            <button className="lp-secondary">Run now</button>
            <button className="lp-secondary">Edit schedule</button>
          </div>
        </section>
      </div>
    </article>
  );
}

function DetailEmpty() {
  return (
    <article className="lp-panel">
      <div className="lp-empty">
        <svg width="32" height="32" viewBox="0 0 32 32" aria-hidden="true" className="lp-empty__icon">
          <rect x="4" y="6" width="24" height="20" rx="3" fill="none" stroke="currentColor" strokeWidth="1.5" />
          <line x1="8" y1="12" x2="20" y2="12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.5" />
          <line x1="8" y1="16" x2="16" y2="16" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.4" />
          <line x1="8" y1="20" x2="14" y2="20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" />
        </svg>
        <h3 className="lp-empty__title">No row selected</h3>
        <p className="lp-empty__hint">
          Pick a row on the left to inspect its run history, properties, sub-issues, and actions.
        </p>
        <div className="lp-empty__kbd">
          <span className="lp-kbd">↑</span>
          <span className="lp-kbd">↓</span>
          <span style={{ marginRight: 4 }}>navigate</span>
          <span className="lp-kbd">⏎</span>
          <span style={{ marginRight: 4 }}>open</span>
          <span className="lp-kbd">⌘</span>
          <span className="lp-kbd">K</span>
          <span>jump</span>
        </div>
      </div>
    </article>
  );
}

function VariantPin() {
  return (
    <div className="lp-variant-pin" aria-label="Design variant">
      <span className="lp-variant-pin__label">V2 Panel</span>
      <a href="../v1-linear-calm/">Calm</a>
      <button type="button" className="is-active" aria-current="page">
        Panel
      </button>
      <a href="../v3-linear-editorial/">Editorial</a>
    </div>
  );
}

function App() {
  const [view, setView] = useState("tasks");
  const [taskId, setTaskId] = useState(D.TASKS[1].id); // default = in-progress with sub-issues
  const [jobId, setJobId] = useState(D.JOBS[0].id);

  const selectedTask = D.TASKS.find((t) => t.id === taskId);
  const selectedJob = D.JOBS.find((j) => j.id === jobId);

  return (
    <Fragment>
      <VariantPin />
      <div className="lp-stage">
        <WorkspaceRail />
        <div className="lp-frame-wrap">
          <div className="lp-frame">
            <Sidebar view={view} setView={setView} />
            {view === "tasks" ? (
              <TasksView selectedId={taskId} setSelectedId={setTaskId} />
            ) : (
              <JobsView selectedId={jobId} setSelectedId={setJobId} />
            )}
            {view === "tasks" ? <TaskDetail task={selectedTask} /> : <JobDetail job={selectedJob} />}
          </div>
        </div>
      </div>
    </Fragment>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
