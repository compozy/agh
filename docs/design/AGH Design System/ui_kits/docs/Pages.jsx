// Page bodies for the docs kit — a session page, a network overview page, a CLI reference page.

function SessionsPage() {
  const sessionCode = `# Create a session; specify ACP agent + provider
$ agh session new --agent coder --provider claude

# Stream events back over SSE
$ agh session events <session-id> --tail`;
  const resumeCode = `# Stop cleanly. State is durable.
$ agh session stop <session-id>

# Resume exactly where it left off
$ agh session resume <session-id>

# Fork from any turn — creates a new session rooted at that point
$ agh session fork <session-id> --from-turn 14`;
  return (
    <>
      <Breadcrumb trail={["Runtime", "Sessions"]} />
      <DocH1>Sessions</DocH1>
      <p
        style={{
          margin: "16px 0 0",
          fontFamily: "Inter",
          fontSize: 17,
          lineHeight: 1.6,
          color: "#8E8E93",
          maxWidth: "62ch",
        }}
      >
        A session is one durable agent run. You can stop it, resume it, inspect every event, and
        fork from any turn — because everything is persisted to SQLite as it happens.
      </p>
      <DocH2 id="what">What sessions are</DocH2>
      <DocP>
        Every time you run an agent under AGH, the daemon spawns the ACP process as a managed
        subprocess and records each event — prompts, tool calls, permission decisions, output — to a
        local event log. The <InlineCode>session</InlineCode> subcommand is how you drive this
        surface.
      </DocP>
      <Callout kind="info" title="Default storage">
        Sessions live under <InlineCode>$AGH_HOME/sessions/&lt;id&gt;/</InlineCode> with a WAL-mode
        SQLite database per session. Safe to back up, safe to delete.
      </Callout>
      <DocH2 id="create">Create a session</DocH2>
      <CodeBlock code={sessionCode} caption="agh session" shell />
      <DocH2 id="resume">Resume and fork</DocH2>
      <DocP>
        Sessions are addressable by their ID forever. A stopped session can be resumed, and any past
        turn can be used as a fork point.
      </DocP>
      <CodeBlock code={resumeCode} caption="agh session" shell />
      <DocH2 id="events">Event stream</DocH2>
      <DocP>
        Events are published as they happen. Subscribe over SSE for live monitoring, or paginate the
        log for replay.
      </DocP>
      <CommandTable
        rows={[
          { cmd: "session events --tail", desc: "Stream new events as they happen (SSE)." },
          { cmd: "session events --since", desc: "Replay from a timestamp or event id." },
          {
            cmd: "session events --kind",
            desc: "Filter by event kind: prompt, tool_call, permission, output.",
          },
        ]}
      />
      <PageNav prev="Core concepts" next="Memory" />
    </>
  );
}

Object.assign(window, { SessionsPage });
