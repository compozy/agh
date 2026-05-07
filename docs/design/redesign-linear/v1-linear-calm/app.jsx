/* V1 — linear-calm
 * Direct port of Linear's grammar to AGH's warm palette. Operator density
 * stays. Orange becomes scarce. Status moves to iconographic glyphs.
 * Priority becomes a 3-bar signal. Every row carries an avatar. */

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
    ? "lc-avatar lc-avatar--system"
    : isAgent
    ? "lc-avatar lc-avatar--agent"
    : "lc-avatar";
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

/* ── Status & Priority wrappers (color comes from data) ──────────────── */

function StatusGlyph({ status, size = 14 }) {
  return (
    <span className="lc-row__statusicon">
      <I.Status status={status} size={size} />
    </span>
  );
}

function PriorityGlyph({ level }) {
  if (!level) return null;
  return (
    <span className={`lc-row__priority lc-row__priority--${level}`} title={`${level} priority`}>
      <I.Priority level={level} size={12} />
    </span>
  );
}

/* ── PropertyPill ─────────────────────────────────────────────────────── */

function PropertyPill({ icon, leading, tone = "neutral", mono = false, onClick, children }) {
  const Tag = onClick ? "button" : "span";
  const cls = [
    "lc-prop",
    tone !== "neutral" ? `lc-prop--${tone}` : null,
    mono ? "lc-prop--mono" : null,
    onClick ? "lc-prop--button" : null,
  ]
    .filter(Boolean)
    .join(" ");
  return (
    <Tag type={onClick ? "button" : undefined} onClick={onClick} className={cls}>
      {leading ? <span className="lc-prop__icon">{leading}</span> : null}
      {icon ? <span className="lc-prop__icon">{icon}</span> : null}
      <span>{children}</span>
    </Tag>
  );
}

/* ── Workspace rail + Sidebar ────────────────────────────────────────── */

function WorkspaceRail() {
  return (
    <aside className="lc-rail" aria-label="Workspaces">
      {D.WORKSPACES.map((ws) => (
        <button
          key={ws.id}
          type="button"
          className={`lc-rail__ws${ws.active ? " lc-rail__ws--active" : ""}`}
          style={ws.active ? { background: ws.color } : undefined}
          title={ws.label}
        >
          {ws.initial}
        </button>
      ))}
      <button type="button" className="lc-rail__plus" aria-label="Add workspace">
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
      className={`lc-nav__row${active ? " lc-nav__row--active" : ""}`}
      onClick={disabled ? undefined : onClick}
      aria-disabled={disabled || undefined}
      aria-current={active ? "page" : undefined}
    >
      <span className="lc-nav__icon">
        <Icon size={14} />
      </span>
      <span className="lc-nav__label">{item.label}</span>
      {item.kbd ? <span className="lc-nav__kbd">{item.kbd}</span> : null}
      {item.count !== null && item.count !== undefined ? (
        <span className="lc-nav__count">{item.count}</span>
      ) : null}
    </button>
  );
}

function Sidebar({ view, setView }) {
  return (
    <aside className="lc-side" aria-label="Primary navigation">
      <div className="lc-side__head">
        <span className="lc-side__brand">agh</span>
        <span className="lc-side__alpha">Alpha</span>
      </div>
      <button type="button" className="lc-side__search" aria-label="Open command palette">
        <span className="lc-side__search-icon">
          <I.Search size={12} />
        </span>
        <span>Search</span>
        <span className="lc-side__search-kbd">
          <span className="lc-kbd">⌘</span>
          <span className="lc-kbd">K</span>
        </span>
      </button>
      <div className="lc-side__group">
        <div className="lc-side__label">Workspace</div>
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
      <div className="lc-side__group">
        <div className="lc-side__label">Operations</div>
        {D.NAV_SECONDARY.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="lc-side__group">
        {D.NAV_FOOTER.map((item) => (
          <NavRow key={item.id} item={item} active={false} disabled onClick={() => {}} />
        ))}
      </div>
      <div className="lc-side__footer">
        <span className="lc-status-dot" />
        <span>Connected · v0.41.0</span>
      </div>
    </aside>
  );
}

/* ── PageHead ────────────────────────────────────────────────────────── */

function PageHead({ title, count, summary, primary }) {
  return (
    <header className="lc-page-head">
      <div>
        <div className="lc-page-head__title">
          <h1>{title}</h1>
          <span className="lc-page-head__count">{count}</span>
        </div>
        {summary ? <p className="lc-page-head__sub">{summary}</p> : null}
      </div>
      <div className="lc-page-head__primary">{primary}</div>
    </header>
  );
}

/* ── PillGroup ───────────────────────────────────────────────────────── */

function PillGroup({ items, value, onChange }) {
  return (
    <div className="lc-pillgroup" role="tablist">
      {items.map((item) => (
        <button
          key={item.value}
          type="button"
          role="tab"
          aria-selected={item.value === value}
          className={`lc-pillgroup__seg${item.value === value ? " lc-pillgroup__seg--active" : ""}`}
          onClick={() => onChange(item.value)}
        >
          <span>{item.label}</span>
          {item.badge != null ? <span className="lc-pillgroup__badge">{item.badge}</span> : null}
        </button>
      ))}
    </div>
  );
}

/* ── Tasks list ──────────────────────────────────────────────────────── */

function TaskRow({ task, selected, onSelect }) {
  return (
    <div
      className={`lc-row${selected ? " lc-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(task.id)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onSelect(task.id);
        }
      }}
    >
      <div className="lc-row__line">
        <StatusGlyph status={task.status} />
        <Avatar owner={task.owner} size={20} />
        <span className="lc-row__name">{task.title}</span>
        <span className="lc-row__cluster">
          <PriorityGlyph level={task.priority} />
          {task.parentTaskId ? (
            <span className="lc-row__sub" title={`parent ${task.parentTaskId}`}>
              <I.Corner size={11} />
              <span>{task.parentTaskId.slice(-6)}</span>
            </span>
          ) : null}
          {task.activeRun ? (
            <span className="lc-row__attempts" title="attempts">
              {task.activeRun.attempt}
              <span className="lc-row__attempts-sep">/</span>
              {task.activeRun.maxAttempts}
            </span>
          ) : null}
          {task.childCount > 0 ? (
            <span className="lc-row__sub" title={`${task.childCount} children`}>
              <I.Boxes size={11} />
              <span>{task.childCount}</span>
            </span>
          ) : null}
          <span className="lc-row__id">{task.id}</span>
          <span className="lc-row__time" title={task.timestampIso}>
            {task.timestamp}
          </span>
        </span>
      </div>

      {task.status === "failed" && task.activeRun?.error ? (
        <p className="lc-row__error">
          <I.Alert size={11} stroke={2} />
          <span>{task.activeRun.error}</span>
        </p>
      ) : null}

      {task.isBlocked || task.approvalState === "pending" || task.isDraft ? (
        <div className="lc-row__sublane">
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
              className="lc-row__action"
              onClick={(e) => e.stopPropagation()}
            >
              <I.Plus size={11} />
              Publish
            </button>
          ) : null}
          {task.status === "failed" ? (
            <button
              type="button"
              className="lc-row__action"
              onClick={(e) => e.stopPropagation()}
            >
              <I.Refresh size={11} />
              Retry
            </button>
          ) : null}
        </div>
      ) : task.status === "failed" ? (
        <div className="lc-row__sublane">
          <button type="button" className="lc-row__action" onClick={(e) => e.stopPropagation()}>
            <I.Refresh size={11} />
            Retry
          </button>
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
      badge: D.TASKS.filter((t) => t.owner.kind === "human" && !["completed", "canceled"].includes(t.status)).length,
    },
    { value: "watched", label: "Watched", badge: 2 },
  ];

  const tasks = D.TASKS.filter((t) =>
    lane === "mine" ? t.owner.kind === "human" : lane === "watched" ? false : true
  ).filter((t) => (search ? t.title.toLowerCase().includes(search.toLowerCase()) : true));

  const summary = `${D.TASK_SUMMARY.open} open · ${D.TASK_SUMMARY.running} running · ${D.TASK_SUMMARY.blocked} blocked · ${D.TASK_SUMMARY.failed} failed`;

  return (
    <>
      <PageHead
        title="Tasks"
        count={D.TASK_SUMMARY.total}
        summary={summary}
        primary={
          <>
            <button type="button" className="lc-secondary">
              <I.Filter size={13} />
              Filters
            </button>
            <button type="button" className="lc-cta">
              <I.Plus size={13} />
              New task
              <span className="lc-cta__kbd">N</span>
            </button>
          </>
        }
      />
      <div className="lc-filters">
        <div className="lc-search">
          <span className="lc-search__icon">
            <I.Search size={12} />
          </span>
          <input
            className="lc-search__input"
            placeholder="Filter tasks…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <span className="lc-kbd">/</span>
        </div>
        <PillGroup items={lanes} value={lane} onChange={setLane} />
      </div>
      <div className="lc-list__headline">
        <span>
          All tasks
          <span className="lc-list__headline-num">{tasks.length}</span>
        </span>
        <span className="lc-list__headline-kbd">
          <span className="lc-kbd">J</span>
          <span className="lc-kbd">K</span>
          <span style={{ marginLeft: 4 }}>navigate</span>
        </span>
      </div>
      <div className="lc-list__rows">
        {tasks.map((task) => (
          <TaskRow
            key={task.id}
            task={task}
            selected={task.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </>
  );
}

/* ── Jobs list ───────────────────────────────────────────────────────── */

function JobRow({ job, selected, onSelect }) {
  const enabledColor = job.enabled ? "var(--color-success)" : "var(--color-text-tertiary)";
  return (
    <div
      className={`lc-row${selected ? " lc-row--selected" : ""}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(job.id)}
    >
      <div className="lc-row__line">
        <span className="lc-row__statusicon" style={{ color: enabledColor }}>
          {job.enabled ? <I.Status status="ready" size={14} /> : <I.Status status="canceled" size={14} />}
        </span>
        <span className="lc-row__statusicon">
          <I.Clock size={14} stroke={1.6} />
        </span>
        <span className="lc-row__name">{job.name}</span>
        <span className="lc-row__cluster">
          <span style={{ color: job.source === "dynamic" ? "var(--color-text-secondary)" : "var(--color-text-tertiary)" }}>
            {job.source}
          </span>
          <span style={{ color: job.scope === "workspace" ? "var(--color-text-secondary)" : "var(--color-text-tertiary)" }}>
            {job.scope}
          </span>
          <span className="lc-row__id">{job.id}</span>
          <span className="lc-row__time" title={`avg ${job.avgDuration}`}>
            next · {job.nextRun}
          </span>
        </span>
      </div>
      <div className="lc-row__sublane">
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
    <>
      <PageHead
        title="Jobs"
        count={D.JOB_SUMMARY.total}
        summary={summary}
        primary={
          <>
            <button type="button" className="lc-secondary">
              <I.Activity size={13} />
              History
            </button>
            <button type="button" className="lc-cta">
              <I.Plus size={13} />
              New job
              <span className="lc-cta__kbd">N</span>
            </button>
          </>
        }
      />
      <div className="lc-filters">
        <div className="lc-search">
          <span className="lc-search__icon">
            <I.Search size={12} />
          </span>
          <input
            className="lc-search__input"
            placeholder="Search jobs…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <span className="lc-kbd">/</span>
        </div>
        <PillGroup items={scopes} value={scope} onChange={setScope} />
      </div>
      <div className="lc-list__headline">
        <span>
          {jobs.length} visible
          <span className="lc-list__headline-num">/ {D.JOBS.length} total</span>
        </span>
        <span className="lc-list__headline-kbd">
          <span className="lc-kbd">J</span>
          <span className="lc-kbd">K</span>
          <span style={{ marginLeft: 4 }}>navigate</span>
        </span>
      </div>
      <div className="lc-list__rows">
        {jobs.map((job) => (
          <JobRow
            key={job.id}
            job={job}
            selected={job.id === selectedId}
            onSelect={setSelectedId}
          />
        ))}
      </div>
    </>
  );
}

/* ── Detail panels ───────────────────────────────────────────────────── */

function TaskDetail({ task }) {
  if (!task) return <DetailEmpty />;
  const subissues = D.getSubissuesFor(task);
  const completedKids = subissues.filter((s) => s.status === "completed").length;
  const activity = D.getActivityFor(task);

  return (
    <article className="lc-detail">
      <div className="lc-detail__head">
        <div className="lc-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>tasks</span>
          <I.ChevronRight size={11} />
          <span>{task.id}</span>
          <span className="lc-detail__crumb-actions">
            <button type="button" className="lc-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
            <button type="button" className="lc-iconbtn" title="Open in new pane">
              <I.ArrowUpRight size={12} />
            </button>
          </span>
        </div>
        <h2 className="lc-detail__title">{task.title}</h2>
        <div className="lc-props">
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
      <div className="lc-detail__body">
        {subissues.length > 0 ? (
          <section className="lc-section">
            <span className="lc-section__label">Sub-issues</span>
            <div className="lc-subissues">
              <div className="lc-subissues__bar">
                <span>
                  {completedKids} / {subissues.length} done
                </span>
                <span className="lc-subissues__progress">
                  <i style={{ width: `${(completedKids / subissues.length) * 100}%` }} />
                </span>
              </div>
              {subissues.map((sub) => (
                <a key={sub.id} href="#" className="lc-subissue" onClick={(e) => e.preventDefault()}>
                  <I.Status status={sub.status} size={13} />
                  <span className="lc-subissue__name">{sub.title}</span>
                  <span className="lc-subissue__id">{sub.id.slice(-6)}</span>
                  <Avatar owner={sub.assignee} size={16} />
                </a>
              ))}
            </div>
          </section>
        ) : null}

        <section className="lc-section">
          <span className="lc-section__label">Activity</span>
          <div className="lc-activity">
            {activity.map((event, idx) => (
              <div key={idx} className="lc-activity__item">
                <span className="lc-activity__avatar">
                  <Avatar owner={event.author} size={20} />
                </span>
                <div>
                  <div className="lc-activity__msg">
                    <strong>{event.author.label}</strong> {event.msg.toLowerCase()}
                  </div>
                  {event.detail ? <div className="lc-activity__detail">{event.detail}</div> : null}
                  <div className="lc-activity__time">{event.time}</div>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="lc-section">
          <span className="lc-section__label">Actions</span>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {task.status === "failed" ? (
              <button type="button" className="lc-cta">
                <I.Refresh size={13} />
                Retry run
              </button>
            ) : null}
            {task.isDraft ? (
              <button type="button" className="lc-cta">
                <I.Play size={13} />
                Publish task
              </button>
            ) : null}
            <button type="button" className="lc-secondary">
              Open timeline
            </button>
            <button type="button" className="lc-secondary">
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
    <article className="lc-detail">
      <div className="lc-detail__head">
        <div className="lc-detail__crumb">
          <span>agh-runtime</span>
          <I.ChevronRight size={11} />
          <span>jobs</span>
          <I.ChevronRight size={11} />
          <span>{job.id}</span>
          <span className="lc-detail__crumb-actions">
            <button type="button" className="lc-iconbtn" title="Copy ID">
              <I.Copy size={12} />
            </button>
          </span>
        </div>
        <h2 className="lc-detail__title">{job.name}</h2>
        <div className="lc-props">
          <PropertyPill
            leading={<I.Status status={job.enabled ? "ready" : "canceled"} size={12} />}
            tone={job.enabled ? "neutral" : "neutral"}
          >
            {job.enabled ? "Enabled" : "Disabled"}
          </PropertyPill>
          <PropertyPill leading={<I.Clock size={11} />}>{job.schedule}</PropertyPill>
          <PropertyPill mono>next {job.nextRun}</PropertyPill>
          <PropertyPill>{job.source}</PropertyPill>
          <PropertyPill>{job.scope}</PropertyPill>
        </div>
      </div>
      <div className="lc-detail__body">
        <section className="lc-section">
          <span className="lc-section__label">Recent runs</span>
          <div className="lc-activity">
            <div className="lc-activity__item">
              <span className="lc-activity__avatar">
                <Avatar owner={{ kind: "system", label: "agh" }} size={20} />
              </span>
              <div>
                <div className="lc-activity__msg">
                  <strong>{job.id}</strong> ran successfully
                </div>
                <div className="lc-activity__detail">duration {job.avgDuration} · 0 retries</div>
                <div className="lc-activity__time">{job.lastRun}</div>
              </div>
            </div>
            <div className="lc-activity__item">
              <span className="lc-activity__avatar">
                <Avatar owner={{ kind: "system", label: "agh" }} size={20} />
              </span>
              <div>
                <div className="lc-activity__msg">
                  <strong>{job.id}</strong> ran successfully
                </div>
                <div className="lc-activity__detail">
                  duration {job.avgDuration} · 0 retries
                </div>
                <div className="lc-activity__time">prior</div>
              </div>
            </div>
            <div className="lc-activity__item">
              <span className="lc-activity__avatar">
                <Avatar owner={{ kind: "system", label: "agh" }} size={20} />
              </span>
              <div>
                <div className="lc-activity__msg">
                  <strong>{job.id}</strong> ran successfully
                </div>
                <div className="lc-activity__detail">
                  duration {job.avgDuration} · 0 retries
                </div>
                <div className="lc-activity__time">prior</div>
              </div>
            </div>
          </div>
        </section>
        <section className="lc-section">
          <span className="lc-section__label">Actions</span>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <button type="button" className="lc-cta">
              <I.Play size={13} />
              {job.enabled ? "Pause" : "Enable"}
            </button>
            <button type="button" className="lc-secondary">
              Run now
            </button>
            <button type="button" className="lc-secondary">
              Edit schedule
            </button>
          </div>
        </section>
      </div>
    </article>
  );
}

function DetailEmpty() {
  return (
    <div className="lc-detail lc-detail--empty">
      <div className="lc-empty">
        <svg
          width="32"
          height="32"
          viewBox="0 0 32 32"
          aria-hidden="true"
          className="lc-empty__icon"
        >
          <rect x="4" y="6" width="24" height="20" rx="3" fill="none" stroke="currentColor" strokeWidth="1.5" />
          <line x1="8" y1="12" x2="20" y2="12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.5" />
          <line x1="8" y1="16" x2="16" y2="16" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.4" />
          <line x1="8" y1="20" x2="14" y2="20" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" />
        </svg>
        <h3 className="lc-empty__title">No row selected</h3>
        <p className="lc-empty__hint">
          Pick a row on the left to inspect its run history, properties, and actions.
        </p>
        <div className="lc-empty__kbd">
          <span className="lc-kbd">↑</span>
          <span className="lc-kbd">↓</span>
          <span style={{ marginRight: 4 }}>navigate</span>
          <span className="lc-kbd">⏎</span>
          <span style={{ marginRight: 4 }}>open</span>
          <span className="lc-kbd">⌘</span>
          <span className="lc-kbd">K</span>
          <span>jump</span>
        </div>
      </div>
    </div>
  );
}

/* ── Variant pin ─────────────────────────────────────────────────────── */

function VariantPin() {
  return (
    <div className="lc-variant-pin" aria-label="Design variant">
      <span className="lc-variant-pin__label">V1 Calm</span>
      <button type="button" className="is-active" aria-current="page">
        Calm
      </button>
      <a href="../v2-linear-panel/">Panel</a>
      <a href="../v3-linear-editorial/">Editorial</a>
    </div>
  );
}

/* ── App ─────────────────────────────────────────────────────────────── */

function App() {
  const [view, setView] = useState("tasks");
  const [taskId, setTaskId] = useState(D.TASKS[0].id);
  const [jobId, setJobId] = useState(D.JOBS[0].id);

  const selectedTask = D.TASKS.find((t) => t.id === taskId);
  const selectedJob = D.JOBS.find((j) => j.id === jobId);

  return (
    <Fragment>
      <VariantPin />
      <div className="lc-shell">
        <WorkspaceRail />
        <Sidebar view={view} setView={setView} />
        <section className="lc-list">
          {view === "tasks" ? (
            <TasksView selectedId={taskId} setSelectedId={setTaskId} />
          ) : (
            <JobsView selectedId={jobId} setSelectedId={setJobId} />
          )}
        </section>
        {view === "tasks" ? <TaskDetail task={selectedTask} /> : <JobDetail job={selectedJob} />}
      </div>
    </Fragment>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
