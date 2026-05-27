#!/usr/bin/env python3
"""Read-only token baseline collector for the tokens-perf incident.

The script intentionally reads only:
- Claude Code JSONL transcripts from one project directory.
- AGH SQLite/crash-bundle observability files.

It does not inspect provider auth/config/secret paths.
"""

from __future__ import annotations

import argparse
import json
import sqlite3
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


TOKEN_FIELDS = (
    "input_tokens",
    "cache_creation_input_tokens",
    "cache_read_input_tokens",
    "output_tokens",
)


def parse_time(value: str | None) -> datetime | None:
    if not value:
        return None
    text = value.strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    try:
        parsed = datetime.fromisoformat(text)
    except ValueError:
        return None
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def iso_utc(value: datetime | None) -> str | None:
    if value is None:
        return None
    return value.astimezone(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def minute_bucket(value: datetime | None) -> str:
    if value is None:
        return "unknown"
    rounded = value.astimezone(timezone.utc).replace(second=0, microsecond=0)
    return iso_utc(rounded) or "unknown"


def text_from_content(value: Any, max_chars: int = 80_000) -> str:
    parts: list[str] = []

    def walk(node: Any) -> None:
        if sum(len(part) for part in parts) >= max_chars:
            return
        if isinstance(node, str):
            parts.append(node[:max_chars])
            return
        if isinstance(node, list):
            for item in node:
                walk(item)
            return
        if isinstance(node, dict):
            for key in ("text", "content", "input"):
                if key in node:
                    walk(node[key])
            return

    walk(value)
    text = "\n".join(parts)
    if len(text) > max_chars:
        return text[:max_chars]
    return text


@dataclass
class TranscriptStats:
    path: Path
    first_timestamp: datetime | None = None
    first_user_text: str = ""
    raw_assistant_usage_rows: int = 0
    deduped_assistant_calls: int = 0
    usage: Counter[str] = field(default_factory=Counter)

    def classify(self) -> str:
        lowered = self.first_user_text.lower()
        if "extractor candidate prompt v1" in lowered and "what_not_to_save v1" in lowered:
            return "memory_extractor"
        if "<network-message" in lowered or "<network-body" in lowered:
            return "network_orchestrator"
        if "memory extractor" in lowered and "jsonl" in lowered:
            return "memory_extractor_possible"
        return "other"


def collect_claude(project_dir: Path, start: datetime | None, end: datetime | None) -> dict[str, Any]:
    seen: set[tuple[str, str, str]] = set()
    totals: Counter[str] = Counter()
    by_class: dict[str, Counter[str]] = defaultdict(Counter)
    session_starts_by_class: Counter[str] = Counter()
    calls_by_minute: Counter[str] = Counter()
    cache_read_by_minute: Counter[str] = Counter()
    top_files: list[dict[str, Any]] = []
    transcript_count = 0
    raw_rows = 0
    deduped_calls = 0

    for path in sorted(project_dir.glob("*.jsonl")):
        transcript_count += 1
        stats = TranscriptStats(path=path)

        with path.open("r", encoding="utf-8", errors="replace") as handle:
            for line_no, line in enumerate(handle, 1):
                try:
                    row = json.loads(line)
                except json.JSONDecodeError:
                    continue

                timestamp = parse_time(row.get("timestamp"))
                if timestamp and (stats.first_timestamp is None or timestamp < stats.first_timestamp):
                    stats.first_timestamp = timestamp

                row_type = row.get("type")
                message = row.get("message") if isinstance(row.get("message"), dict) else {}

                if row_type == "user" and not stats.first_user_text:
                    content = message.get("content", row.get("content"))
                    stats.first_user_text = text_from_content(content)

                if row_type != "assistant":
                    continue

                usage = message.get("usage") if isinstance(message.get("usage"), dict) else None
                if not usage:
                    continue
                if not in_window(timestamp, start, end):
                    continue

                stats.raw_assistant_usage_rows += 1
                raw_rows += 1

                request_id = str(row.get("requestId") or row.get("request_id") or "")
                message_id = str(message.get("id") or row.get("uuid") or f"line-{line_no}")
                dedupe_key = (path.name, request_id, message_id)
                if dedupe_key in seen:
                    continue
                seen.add(dedupe_key)

                stats.deduped_assistant_calls += 1
                deduped_calls += 1

                for field_name in TOKEN_FIELDS:
                    value = usage.get(field_name) or 0
                    if isinstance(value, (int, float)):
                        stats.usage[field_name] += int(value)
                        totals[field_name] += int(value)

                bucket = minute_bucket(timestamp)
                calls_by_minute[bucket] += 1
                cache_read_by_minute[bucket] += int(usage.get("cache_read_input_tokens") or 0)

        cls = stats.classify()
        if stats.first_timestamp and in_window(stats.first_timestamp, start, end):
            session_starts_by_class[cls] += 1
        if stats.raw_assistant_usage_rows > 0:
            by_class[cls]["transcripts"] += 1
            by_class[cls]["raw_assistant_usage_rows"] += stats.raw_assistant_usage_rows
            by_class[cls]["deduped_assistant_calls"] += stats.deduped_assistant_calls
            for field_name in TOKEN_FIELDS:
                by_class[cls][field_name] += stats.usage[field_name]

            top_files.append(
                {
                    "file": path.name,
                    "class": cls,
                    "first_timestamp": iso_utc(stats.first_timestamp),
                    "deduped_assistant_calls": stats.deduped_assistant_calls,
                    "cache_creation_input_tokens": stats.usage["cache_creation_input_tokens"],
                    "cache_read_input_tokens": stats.usage["cache_read_input_tokens"],
                    "output_tokens": stats.usage["output_tokens"],
                }
            )

    context_total = (
        totals["input_tokens"]
        + totals["cache_creation_input_tokens"]
        + totals["cache_read_input_tokens"]
    )
    top_files.sort(key=lambda row: int(row["cache_read_input_tokens"]), reverse=True)

    return {
        "project_dir": str(project_dir),
        "transcripts": transcript_count,
        "raw_assistant_usage_rows": raw_rows,
        "deduped_assistant_calls": deduped_calls,
        "usage": dict(totals),
        "context_token_movement": context_total,
        "by_class": {key: dict(value) for key, value in sorted(by_class.items())},
        "session_starts_in_window_by_class": dict(session_starts_by_class),
        "top_files_by_cache_read": top_files[:15],
        "top_minutes_by_calls": calls_by_minute.most_common(10),
        "top_minutes_by_cache_read": cache_read_by_minute.most_common(10),
    }


def in_window(value: datetime | None, start: datetime | None, end: datetime | None) -> bool:
    if value is None:
        return False
    if start and value < start:
        return False
    if end and value > end:
        return False
    return True


def sqlite_connect_readonly(path: Path) -> sqlite3.Connection:
    return sqlite3.connect(f"file:{path}?mode=ro", uri=True)


def table_exists(conn: sqlite3.Connection, name: str) -> bool:
    row = conn.execute(
        "SELECT 1 FROM sqlite_master WHERE type='table' AND name=?",
        (name,),
    ).fetchone()
    return row is not None


def columns(conn: sqlite3.Connection, table: str) -> set[str]:
    return {str(row[1]) for row in conn.execute(f"PRAGMA table_info({table})")}


def first_existing(candidates: tuple[str, ...], available: set[str]) -> str | None:
    for candidate in candidates:
        if candidate in available:
            return candidate
    return None


def where_time(column: str | None, start_utc: str | None, end_utc: str | None) -> tuple[str, list[Any]]:
    clauses: list[str] = []
    params: list[Any] = []
    if column and start_utc:
        clauses.append(f"{column} >= ?")
        params.append(start_utc)
    if column and end_utc:
        clauses.append(f"{column} <= ?")
        params.append(end_utc)
    if not clauses:
        return "", params
    return " WHERE " + " AND ".join(clauses), params


def collect_agh(agh_home: Path, start: datetime | None, end: datetime | None) -> dict[str, Any]:
    db_path = agh_home / "agh.db"
    if not db_path.is_file():
        return {"agh_home": str(agh_home), "error": "agh.db not found"}

    start_utc = iso_utc(start)
    end_utc = iso_utc(end)
    result: dict[str, Any] = {"agh_home": str(agh_home), "db": str(db_path)}

    with sqlite_connect_readonly(db_path) as conn:
        if table_exists(conn, "sessions"):
            cols = columns(conn, "sessions")
            time_col = first_existing(("created_at", "started_at", "updated_at"), cols)
            where, params = where_time(time_col, start_utc, end_utc)

            group_cols = [
                col
                for col in ("spawn_role", "session_type", "provider", "agent_name", "state", "failure_kind")
                if col in cols
            ]
            if group_cols:
                select_cols = ", ".join(f"COALESCE({col}, '') AS {col}" for col in group_cols)
                group_by = ", ".join(group_cols)
                query = (
                    f"SELECT {select_cols}, COUNT(*) AS count FROM sessions"
                    f"{where} GROUP BY {group_by} ORDER BY count DESC"
                )
                rows = conn.execute(query, params).fetchall()
                result["sessions_by_role_type_provider_agent"] = [
                    {**{group_cols[i]: row[i] for i in range(len(group_cols))}, "count": row[-1]}
                    for row in rows
                ]

            if "spawn_role" in cols:
                query = f"SELECT COUNT(*) FROM sessions{where} AND spawn_role = ?" if where else (
                    "SELECT COUNT(*) FROM sessions WHERE spawn_role = ?"
                )
                query_params = [*params, "memory-extractor"] if where else ["memory-extractor"]
                result["memory_extractor_sessions"] = conn.execute(query, query_params).fetchone()[0]

        if table_exists(conn, "memory_events"):
            cols = columns(conn, "memory_events")
            time_col = first_existing(("created_at", "timestamp", "time"), cols)
            op_col = first_existing(("op", "operation", "event_type", "type"), cols)
            if op_col:
                where, params = where_time(time_col, start_utc, end_utc)
                rows = conn.execute(
                    f"SELECT {op_col}, COUNT(*) FROM memory_events{where} GROUP BY {op_col} ORDER BY COUNT(*) DESC",
                    params,
                ).fetchall()
                result["memory_events_by_op"] = [{"op": row[0], "count": row[1]} for row in rows]

        if table_exists(conn, "token_stats"):
            cols = columns(conn, "token_stats")
            time_col = first_existing(("created_at", "timestamp", "recorded_at", "time"), cols)
            session_col = first_existing(("session_id", "session"), cols)
            total_tokens_col = first_existing(("total_tokens", "context_used"), cols)
            total_cost_col = first_existing(("total_cost", "cost_amount"), cols)
            where_ts, params_ts = where_time(time_col, start_utc, end_utc)
            if session_col and table_exists(conn, "sessions") and total_tokens_col:
                cost_expr = f"SUM(ts.{total_cost_col})" if total_cost_col else "NULL"
                query = (
                    "SELECT COALESCE(s.spawn_role, ''), COUNT(*), "
                    f"SUM(ts.{total_tokens_col}), {cost_expr} "
                    f"FROM token_stats ts JOIN sessions s ON s.id = ts.{session_col}"
                    f"{where_ts} GROUP BY COALESCE(s.spawn_role, '') ORDER BY COUNT(*) DESC"
                )
                rows = conn.execute(query, params_ts).fetchall()
                result["token_stats_by_spawn_role"] = [
                    {
                        "spawn_role": row[0],
                        "rows": row[1],
                        "total_tokens_or_context_used": row[2],
                        "total_cost": row[3],
                    }
                    for row in rows
                ]

        if table_exists(conn, "network_audit_log"):
            cols = columns(conn, "network_audit_log")
            time_col = first_existing(("created_at", "timestamp", "time"), cols)
            verb_col = first_existing(("verb", "message_type", "kind", "op"), cols)
            direction_col = first_existing(("direction", "flow"), cols)
            where, params = where_time(time_col, start_utc, end_utc)
            if verb_col:
                select = f"{verb_col}"
                group = verb_col
                if direction_col:
                    select += f", {direction_col}"
                    group += f", {direction_col}"
                rows = conn.execute(
                    f"SELECT {select}, COUNT(*) FROM network_audit_log{where} GROUP BY {group} ORDER BY COUNT(*) DESC",
                    params,
                ).fetchall()
                result["network_audit_counts"] = [
                    (
                        {"verb": row[0], "direction": row[1], "count": row[2]}
                        if direction_col
                        else {"verb": row[0], "count": row[1]}
                    )
                    for row in rows
                ]

    result["crash_bundles"] = collect_crash_bundles(agh_home)
    return result


def collect_crash_bundles(agh_home: Path) -> dict[str, Any]:
    crash_dir = agh_home / "logs" / "crash-bundles"
    if not crash_dir.is_dir():
        return {"dir": str(crash_dir), "files": 0}

    counters: Counter[str] = Counter()
    for path in crash_dir.glob("*.json"):
        try:
            text = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        counters["files"] += 1
        lowered = text.lower()
        if "rate_limit" in lowered or "session limit" in lowered:
            counters["rate_or_session_limit"] += 1
        if "prompt_failure" in lowered:
            counters["prompt_failure"] += 1
        if "process_exit" in lowered:
            counters["process_exit"] += 1

    return {"dir": str(crash_dir), **dict(counters)}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--claude-project", default="~/.claude/projects/-Users-pedronauck-Desktop-test")
    parser.add_argument("--agh-home", default="~/.agh")
    parser.add_argument("--start-utc", default="2026-05-26T20:50:00Z")
    parser.add_argument("--end-utc", default="2026-05-26T21:45:00Z")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    start = parse_time(args.start_utc)
    end = parse_time(args.end_utc)
    claude_project = Path(args.claude_project).expanduser()
    agh_home = Path(args.agh_home).expanduser()

    payload = {
        "window_utc": {"start": iso_utc(start), "end": iso_utc(end)},
        "claude": collect_claude(claude_project, start, end) if claude_project.is_dir() else {
            "project_dir": str(claude_project),
            "error": "directory not found",
        },
        "agh": collect_agh(agh_home, start, end),
    }
    print(json.dumps(payload, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
