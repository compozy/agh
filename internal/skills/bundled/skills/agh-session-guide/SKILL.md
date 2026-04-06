---
name: agh-session-guide
description: Operate AGH sessions from the CLI, including creation, inspection, prompting, and shutdown.
version: "1.0.0"
---

# AGH Session Guide

Use this guide when you need to create, inspect, or continue an AGH session from the terminal.

## What AGH sessions are

An AGH session is a managed agent runtime owned by the daemon. The daemon starts the ACP-compatible agent process, records events, and exposes session control through the `agh session` command group.

AGH tracks multiple session lifecycle states:

- `starting`: the daemon has accepted the session and is booting the agent runtime
- `active`: the agent process is connected and ready to receive prompts
- `stopping`: shutdown is in progress
- `stopped`: the runtime has exited and the session can be inspected or resumed

AGH also tracks internal session types. Daily CLI work creates `user` sessions, while AGH may also create `dream` or `system` sessions for daemon-managed workflows.

## Create a session

Create a new session in the current repository:

```bash
agh session new --agent general --name review-run
```

Create a session for a specific workspace:

```bash
agh session new --agent codex --cwd /absolute/path/to/worktree --name fix-task
```

Use `--agent` to choose an `AGENT.md` definition. If you omit it, AGH uses the configured default agent.

## Inspect and monitor sessions

List active sessions:

```bash
agh session list
```

Include stopped sessions too:

```bash
agh session list --all
```

Inspect a single session:

```bash
agh session status <session-id>
```

Watch the event stream for a live session:

```bash
agh session events <session-id> --follow
```

Review the turn-grouped history for a session:

```bash
agh session history <session-id>
```

Block until the session stops:

```bash
agh session wait <session-id>
```

## Prompt, stop, and resume

Send one prompt turn to a running session:

```bash
agh session prompt <session-id> "Summarize the last three tool results."
```

Stop a running session:

```bash
agh session stop <session-id>
```

Resume a stopped session:

```bash
agh session resume <session-id>
```

## Practical workflow

When using AGH from the CLI:

1. Start with `agh session new`.
2. Use `agh session list` or `agh session status` to confirm the session reached `active`.
3. Send work with `agh session prompt`.
4. Use `agh session events --follow` or `agh session history` when you need observability.
5. Stop or resume the session explicitly instead of assuming the runtime state.

If a session does not behave as expected, inspect `status`, then `events`, then `history`. That sequence gives the fastest signal on whether the issue is startup, runtime execution, or persisted output.
