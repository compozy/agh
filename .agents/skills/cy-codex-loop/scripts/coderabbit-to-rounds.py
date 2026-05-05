#!/usr/bin/env python3
"""
coderabbit-to-rounds.py -- mutating.

Converts ``coderabbit review --agent`` JSON output into the directory
layout expected by ``cy-fix-reviews``: ``reviews-NNN/issue_NNN.md`` with
the canonical YAML frontmatter (see ``references/coderabbit-conversion.md``).

Usage:
    coderabbit-to-rounds.py <input.json> <reviews-NNN-output-dir>
                            [--round N] [--repo-root .]

Behavior:
- ``--round`` overrides the round number derived from the output dir name.
- The output dir must NOT exist or must be empty (refuses to overwrite).
- If the input has zero findings, creates the output dir with a ``.empty``
  marker, prints ``EMPTY``, and exits 0.

Exits:
    0 success (or EMPTY)
    1 generic error
    2 input file missing
    3 input not parseable JSON
    4 output dir exists and is non-empty
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path


_SEVERITY_MAP = {
    "critical": "critical",
    "blocker": "critical",
    "severe": "critical",
    "high": "high",
    "major": "high",
    "medium": "medium",
    "minor": "medium",
    "moderate": "medium",
    "low": "low",
    "info": "low",
    "nit": "low",
    "suggestion": "low",
}


def _now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _normalize_severity(raw: object) -> tuple[str, bool]:
    """Return (severity, was_normalized_with_note)."""
    if raw is None:
        return "medium", True
    if isinstance(raw, (int, float)):
        n = float(raw)
        if n >= 9:
            return "critical", False
        if n >= 7:
            return "high", False
        if n >= 4:
            return "medium", False
        return "low", False
    if isinstance(raw, str):
        key = raw.strip().lower()
        if key in _SEVERITY_MAP:
            return _SEVERITY_MAP[key], False
    return "medium", True


def _find_findings(payload: object, *, _depth: int = 0) -> list[dict]:
    """Defensively pull a flat list of findings from a coderabbit payload."""
    if _depth > 8:
        return []
    if isinstance(payload, list):
        return [x for x in payload if isinstance(x, dict)]
    if not isinstance(payload, dict):
        return []
    for key in ("findings", "comments", "issues", "results", "items"):
        val = payload.get(key)
        if isinstance(val, list):
            return [x for x in val if isinstance(x, dict)]
    # Sometimes the payload nests under {"review": {...}}:
    review = payload.get("review")
    if isinstance(review, dict):
        return _find_findings(review, _depth=_depth + 1)
    return []


def _first(d: dict, *keys: str) -> object:
    for k in keys:
        if k in d and d[k] not in (None, ""):
            return d[k]
    return None


def _norm_path(value: object, repo_root: Path) -> str:
    if not isinstance(value, str):
        return ""
    p = value.strip()
    if not p:
        return ""
    if p.startswith("./"):
        p = p[2:]
    candidate = Path(p)
    if candidate.is_absolute():
        try:
            resolved_candidate = candidate.resolve(strict=False)
            resolved_root = repo_root.resolve(strict=False)
            return resolved_candidate.relative_to(resolved_root).as_posix()
        except (OSError, ValueError):
            pass
    if p.startswith(str(repo_root)):
        p = p[len(str(repo_root)):].lstrip("/")
    return p


def _norm_line(value: object) -> int:
    if isinstance(value, bool):  # bool is a subclass of int
        return 0
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        match = re.search(r"\d+", value)
        if match:
            return int(match.group(0))
    return 0


def _truncate_title(text: str, limit: int = 72) -> str:
    text = " ".join(text.split())
    if not text:
        return "Untitled coderabbit finding"
    return text if len(text) <= limit else text[: limit - 1].rstrip() + "…"


def _yaml_str(value: str) -> str:
    """YAML-friendly string: quote if it contains anything tricky."""
    if value == "":
        return ""
    if re.search(r'[:#\[\]\{\}",&*!|>%@`]', value) or value != value.strip():
        escaped = value.replace("\\", "\\\\").replace('"', '\\"')
        return f'"{escaped}"'
    return value


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("input")
    ap.add_argument("output_dir")
    ap.add_argument("--round", type=int, default=None)
    ap.add_argument("--repo-root", default=".")
    args = ap.parse_args()

    input_path = Path(args.input)
    if not input_path.exists():
        print(
            f"coderabbit-to-rounds: input file {input_path} not found",
            file=sys.stderr,
        )
        return 2
    try:
        payload = json.loads(input_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        print(f"coderabbit-to-rounds: invalid JSON: {exc}", file=sys.stderr)
        return 3

    findings = _find_findings(payload)
    out_dir = Path(args.output_dir)

    if out_dir.exists() and any(out_dir.iterdir()):
        print(
            f"coderabbit-to-rounds: output dir {out_dir} is non-empty; "
            "pass a fresh reviews-NNN/ path",
            file=sys.stderr,
        )
        return 4
    out_dir.mkdir(parents=True, exist_ok=True)

    if not findings:
        marker = out_dir / ".empty"
        marker.write_text(
            f"empty_round_created_at: {_now_iso()}\n",
            encoding="utf-8",
        )
        print("EMPTY")
        return 0

    if args.round is not None:
        round_num = args.round
    else:
        match = re.match(r"reviews-0*(\d+)$", out_dir.name)
        if not match:
            print(
                f"coderabbit-to-rounds: cannot infer round number from "
                f"{out_dir.name}; pass --round N",
                file=sys.stderr,
            )
            return 1
        round_num = int(match.group(1))

    repo_root = Path(args.repo_root).resolve()
    timestamp = _now_iso()

    for idx, finding in enumerate(findings, start=1):
        sev, sev_note = _normalize_severity(_first(finding, "severity", "priority", "level"))
        file_path = _norm_path(_first(finding, "file", "path"), repo_root)
        line = _norm_line(_first(finding, "line", "line_number", "start_line"))
        comment = _first(finding, "comment", "description", "body") or ""
        if not isinstance(comment, str):
            comment = json.dumps(comment, indent=2)
        suggestion = _first(finding, "suggestion", "fix", "recommendation") or ""
        if not isinstance(suggestion, str):
            suggestion = json.dumps(suggestion, indent=2)
        raw_title = _first(finding, "title", "summary")
        if isinstance(raw_title, str) and raw_title.strip():
            title = _truncate_title(raw_title)
        else:
            title = _truncate_title(comment.split(".")[0] if comment else "")
        provider_ref = _first(finding, "id", "comment_id", "finding_id") or ""
        if not isinstance(provider_ref, (str, int)):
            provider_ref = ""

        body_parts = [comment.strip()] if comment.strip() else []
        if sev_note:
            body_parts.append(
                "_NOTE: cy-codex-loop normalized severity to `medium` because the "
                "input value was unrecognized._"
            )
        if not file_path:
            body_parts.append(
                "_NOTE: coderabbit did not provide a usable file path for this finding; "
                "triage manually._"
            )
        if suggestion.strip():
            body_parts.append("### Suggested Fix\n\n" + suggestion.strip())
        body = "\n\n".join(body_parts) if body_parts else "(no body)"

        frontmatter = (
            "---\n"
            "provider: coderabbit\n"
            "pr:\n"
            f"round: {round_num}\n"
            f"round_created_at: {timestamp}\n"
            "status: pending\n"
            f"file: {_yaml_str(file_path)}\n"
            f"line: {line}\n"
            f"severity: {sev}\n"
            "author: cy-codex-loop\n"
            f"provider_ref: {_yaml_str(str(provider_ref))}\n"
            "---\n\n"
        )
        content = (
            frontmatter
            + f"# Issue {idx:03d}: {title}\n\n"
            + "## Review Comment\n\n"
            + body
            + "\n\n## Triage\n\n"
            + "- Decision: `UNREVIEWED`\n"
            + "- Notes:\n"
        )
        out_file = out_dir / f"issue_{idx:03d}.md"
        out_file.write_text(content, encoding="utf-8")

    print(f"coderabbit-to-rounds: wrote {len(findings)} issues to {out_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
