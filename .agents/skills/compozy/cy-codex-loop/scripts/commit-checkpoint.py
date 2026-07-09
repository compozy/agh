#!/usr/bin/env python3
"""
commit-checkpoint.py -- mutating.

Per-task checkpoint commit for ``cy-codex-loop``. Invoked by the
orchestrator at the end of every Phase B iteration (after
update-state.py advances state and cy-final-verify reports PASS) so
each completed task or free-mode slice becomes one atomic, restorable
git commit.

Usage:
    commit-checkpoint.py <slug> --task <stem>        # mode=tasks
    commit-checkpoint.py <slug> --slice "<text>"     # mode=free
    commit-checkpoint.py <slug> [--tasks-root <p>]

Behavior:
    1. Verify state.yaml exists under <tasks-root>/<slug>/.
    2. ``git status --porcelain``: empty tree -> print ``SKIP: no changes`` and exit 0.
    3. Build commit header:
         --task task_NN -> ``feat: <title from task_NN.md frontmatter> #<NN>``
         --slice "<txt>" -> ``feat: <txt>`` (whitespace collapsed)
       Hard-cap full header at 72 chars.
    4. Build body lines:
         ``Checkpoint via cy-codex-loop (iteration <N>, phase B mode=<mode>).``
         (blank line)
         ``Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>``
    5. ``git add -A`` then ``git commit -m <header> -m <body>``. No --amend,
       --no-verify, or --no-gpg-sign. Hook failures surface as exit 1.
    6. On success, print new commit SHA (``git rev-parse HEAD``) and exit 0.

Exits:
    0 success (commit created OR ``SKIP: no changes``)
    1 git command failure (commit, hook, rev-parse, ...)
    2 argument or state error (missing slug dir, state.yaml, both flags)
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from _state_io import load  # noqa: E402


_FRONTMATTER = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)
_TASK_STEM = re.compile(r"^task_(\d+)$")
_HEADER_MAX = 72
_COAUTHOR = "Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"


def _read_frontmatter(md_path: Path) -> dict[str, str]:
    if not md_path.exists():
        return {}
    text = md_path.read_text(encoding="utf-8", errors="replace")
    match = _FRONTMATTER.match(text)
    if not match:
        return {}
    fm: dict[str, str] = {}
    for line in match.group(1).splitlines():
        if ":" not in line:
            continue
        key, _, value = line.partition(":")
        fm[key.strip()] = value.strip().strip("'\"")
    return fm


def _collapse_ws(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip()


def _truncate(header: str, limit: int = _HEADER_MAX) -> str:
    if len(header) <= limit:
        return header
    return header[: limit - 1].rstrip() + "…"


def _build_task_header(task_md: Path, stem: str) -> str:
    match = _TASK_STEM.match(stem)
    if not match:
        raise SystemExit(
            f"commit-checkpoint: --task value {stem!r} must look like task_NN"
        )
    nn = match.group(1)
    title = _collapse_ws(_read_frontmatter(task_md).get("title", "")) or stem
    return _truncate(f"feat: {title} #{nn}")


def _build_slice_header(text: str) -> str:
    body = _collapse_ws(text)
    if not body:
        raise SystemExit("commit-checkpoint: --slice text is empty after trim")
    return _truncate(f"feat: {body}")


def _run_git(args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        check=False,
        text=True,
        capture_output=True,
    )


def _tree_is_clean() -> bool:
    status = _run_git(["status", "--porcelain"])
    if status.returncode != 0:
        print(
            f"commit-checkpoint: git status failed: {status.stderr.strip()}",
            file=sys.stderr,
        )
        raise SystemExit(1)
    return status.stdout.strip() == ""


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("slug")
    ap.add_argument("--task", help="task stem like task_07 (mode=tasks)")
    ap.add_argument("--slice", dest="slice_text", help="slice text (mode=free)")
    ap.add_argument("--tasks-root", default=".compozy/tasks")
    args = ap.parse_args()

    if not args.task and not args.slice_text:
        print(
            "commit-checkpoint: provide --task <stem> or --slice \"<text>\"",
            file=sys.stderr,
        )
        return 2
    if args.task and args.slice_text:
        print(
            "commit-checkpoint: --task and --slice are mutually exclusive",
            file=sys.stderr,
        )
        return 2

    slug_dir = Path(args.tasks_root) / args.slug
    state_path = slug_dir / "state.yaml"
    if not state_path.exists():
        print(
            f"commit-checkpoint: {state_path} missing; run init-state.py first",
            file=sys.stderr,
        )
        return 2

    if _tree_is_clean():
        print("SKIP: no changes")
        return 0

    try:
        state = load(state_path)
    except Exception as exc:  # noqa: BLE001
        print(
            f"commit-checkpoint: failed to parse {state_path}: {exc}",
            file=sys.stderr,
        )
        return 1

    iteration = int(state.get("iteration", 0))
    mode = state.get("mode") or "?"

    if args.task:
        task_md = slug_dir / f"{args.task}.md"
        header = _build_task_header(task_md, args.task)
    else:
        header = _build_slice_header(args.slice_text or "")

    body = (
        f"Checkpoint via cy-codex-loop (iteration {iteration}, "
        f"phase B mode={mode}).\n\n{_COAUTHOR}"
    )

    add = _run_git(["add", "-A"])
    if add.returncode != 0:
        print(
            f"commit-checkpoint: git add -A failed: {add.stderr.strip()}",
            file=sys.stderr,
        )
        return 1

    commit = _run_git(["commit", "-m", header, "-m", body])
    if commit.returncode != 0:
        sys.stderr.write(commit.stderr)
        sys.stderr.write(commit.stdout)
        return 1

    sha = _run_git(["rev-parse", "HEAD"])
    if sha.returncode != 0:
        print(
            f"commit-checkpoint: git rev-parse HEAD failed: {sha.stderr.strip()}",
            file=sys.stderr,
        )
        return 1
    print(sha.stdout.strip())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
