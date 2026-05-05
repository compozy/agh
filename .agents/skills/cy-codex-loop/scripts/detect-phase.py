#!/usr/bin/env python3
"""
detect-phase.py -- read-only.

Prints the next phase + action for ``cy-codex-loop`` to take. The agent
calls this at the start of every iteration. The output drives the rest
of the iteration deterministically. Filesystem (not state.yaml) is the
ultimate source of truth for task and review-round status; state.yaml
only mirrors what is fast to compute.

Usage:
    detect-phase.py <slug> [--tasks-root .compozy/tasks]

Output (single line, key=value space-separated):
    phase=0 action=bootstrap
    phase=B action=execute_task task=<stem>            # mode=tasks
    phase=B action=execute_free_slice                   # mode=free
    phase=C action=qa_report
    phase=C action=qa_execution
    phase=D action=coderabbit_round round=<NNN>
    phase=D action=coderabbit_fix round=<NNN>
    phase=E action=done

Exits:
    0 always (output describes the situation; missing slug => bootstrap)
    1 unrecoverable error reading state.yaml or filesystem
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from _state_io import load  # noqa: E402


_FRONTMATTER = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)
_CLOSED_STATUSES = {"resolved", "invalid"}


def _read_frontmatter(md_path: Path) -> dict[str, str]:
    if not md_path.exists():
        return {}
    text = md_path.read_text(encoding="utf-8", errors="replace")
    m = _FRONTMATTER.match(text)
    if not m:
        return {}
    fm: dict[str, str] = {}
    for line in m.group(1).splitlines():
        if ":" in line:
            k, _, v = line.partition(":")
            fm[k.strip()] = v.strip().strip("'\"")
    return fm


def _is_qa_task(slug_dir: Path, stem: str) -> bool:
    fm = _read_frontmatter(slug_dir / f"{stem}.md")
    type_field = fm.get("type", "").lower()
    title = fm.get("title", "").lower()
    return (
        "qa-report" in type_field
        or "qa-execution" in type_field
        or "qa report" in title
        or "qa execution" in title
    )


def _qa_kind(slug_dir: Path, stem: str) -> str:
    fm = _read_frontmatter(slug_dir / f"{stem}.md")
    type_field = fm.get("type", "").lower()
    title = fm.get("title", "").lower()
    if "qa-execution" in type_field or "qa execution" in title:
        return "qa_execution"
    return "qa_report"


def _next_round_number(slug_dir: Path) -> int:
    rounds = []
    for p in slug_dir.glob("reviews-*"):
        if p.is_dir():
            try:
                rounds.append(int(p.name.split("-", 1)[1]))
            except (IndexError, ValueError):
                continue
    return (max(rounds) + 1) if rounds else 1


def _round_has_unresolved(round_dir: Path) -> tuple[int, int]:
    crit = 0
    high = 0
    for issue_file in round_dir.glob("issue_*.md"):
        fm = _read_frontmatter(issue_file)
        status = fm.get("status", "").lower()
        sev = fm.get("severity", "").lower()
        if status in _CLOSED_STATUSES:
            continue
        if sev == "critical":
            crit += 1
        elif sev == "high":
            high += 1
    return crit, high


def emit(line: str) -> None:
    print(line)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("slug")
    ap.add_argument("--tasks-root", default=".compozy/tasks")
    args = ap.parse_args()

    slug_dir = Path(args.tasks_root) / args.slug
    state_path = slug_dir / "state.yaml"

    # Phase 0: bootstrap if state.yaml is missing.
    if not state_path.exists():
        emit("phase=0 action=bootstrap")
        return 0

    try:
        state = load(state_path)
    except Exception as exc:  # noqa: BLE001
        print(f"detect-phase: failed to parse {state_path}: {exc}", file=sys.stderr)
        return 1

    mode = state.get("mode")
    rounds_required = state.get("coderabbit", {}).get("rounds_required", 3)
    streak = state.get("coderabbit", {}).get("rounds_clean_streak", 0)
    verify_status = state.get("verify", {}).get("last_status")
    qa = state.get("qa", {})

    # Phase E: done.
    if streak >= rounds_required and verify_status == "PASS":
        emit("phase=E action=done")
        return 0

    # Phase B and C ordering depends on mode.
    if mode == "tasks":
        pending = list(state.get("tasks", {}).get("pending") or [])
        if pending:
            head = pending[0]
            if _is_qa_task(slug_dir, head):
                kind = _qa_kind(slug_dir, head)
                emit(f"phase=C action={kind}")
                return 0
            emit(f"phase=B action=execute_task task={head}")
            return 0
        # No pending: fall through to QA / D
    elif mode == "free":
        progress = state.get("progress", {}) or {}
        if not progress.get("deliverables_complete", False):
            emit("phase=B action=execute_free_slice")
            return 0
    else:
        print(
            f"detect-phase: unknown mode {mode!r} in {state_path}",
            file=sys.stderr,
        )
        return 1

    # Phase C: QA artifacts not yet produced.
    if not qa.get("report_done", False):
        emit("phase=C action=qa_report")
        return 0
    if not qa.get("execution_done", False):
        emit("phase=C action=qa_execution")
        return 0

    # Phase D: CodeRabbit loop until streak is satisfied.
    cr = state.get("coderabbit", {}) or {}
    current = cr.get("current_round_dir")
    if current:
        round_dir = slug_dir / current
        if round_dir.is_dir():
            num = current.split("-", 1)[1]
            emit(f"phase=D action=coderabbit_fix round={num}")
            return 0
        # Stale pointer; act as if no round in progress.
    next_num = _next_round_number(slug_dir)
    emit(f"phase=D action=coderabbit_round round={next_num:03d}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
