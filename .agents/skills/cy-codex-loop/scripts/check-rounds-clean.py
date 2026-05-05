#!/usr/bin/env python3
"""
check-rounds-clean.py -- read-only.

Reports whether a given ``reviews-NNN/`` directory is "clean" (zero
unresolved critical or high issues). Used by Phase D to decide whether
to advance the streak.

Usage:
    check-rounds-clean.py <reviews-NNN-dir>

Output (single line):
    clean=true critical=0 high=0 total=12
    clean=false critical=2 high=3 total=12

Exits:
    0 success (regardless of clean/dirty)
    1 directory missing or unreadable
    2 directory contains no issue files
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


_FRONTMATTER = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)
_CLOSED_STATUSES = {"resolved", "invalid"}


def _read_frontmatter(md_path: Path) -> dict[str, str]:
    try:
        text = md_path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return {}
    match = _FRONTMATTER.match(text)
    if not match:
        return {}
    fm: dict[str, str] = {}
    for line in match.group(1).splitlines():
        if ":" in line:
            k, _, v = line.partition(":")
            fm[k.strip()] = v.strip().strip("'\"")
    return fm


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("round_dir")
    args = ap.parse_args()

    round_dir = Path(args.round_dir)
    if not round_dir.is_dir():
        print(
            f"check-rounds-clean: directory {round_dir} not found",
            file=sys.stderr,
        )
        return 1

    issue_files = sorted(round_dir.glob("issue_*.md"))
    if not issue_files:
        print(
            f"check-rounds-clean: no issue_*.md files under {round_dir}",
            file=sys.stderr,
        )
        return 2

    critical = 0
    high = 0
    for issue_file in issue_files:
        fm = _read_frontmatter(issue_file)
        status = fm.get("status", "").lower()
        sev = fm.get("severity", "").lower()
        if status in _CLOSED_STATUSES:
            continue
        if sev == "critical":
            critical += 1
        elif sev == "high":
            high += 1

    clean = critical == 0 and high == 0
    print(
        f"clean={'true' if clean else 'false'} "
        f"critical={critical} high={high} total={len(issue_files)}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
