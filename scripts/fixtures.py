"""Shared fixture bookkeeping for the three static-analysis engines.

The corpus in internal/analysisfixtures marks every expected finding with
`// ruleid: <rule>` and every case that must stay silent with `// ok: <rule>`.
A marker applies to the first following line that is not itself a marker.
"""

import os
import re

CORPUS = "internal/analysisfixtures"
MARKER = re.compile(r"^\s*//\s*(ruleid|ok):\s*(\S+)\s*$")


def load_markers(root=CORPUS):
    """Return {(file, line): {"ruleid": {...}, "ok": {...}}} for the corpus."""
    out = {}
    for name in sorted(os.listdir(root)):
        if not name.endswith(".go"):
            continue
        path = os.path.join(root, name)
        lines = open(path, encoding="utf-8").read().split("\n")
        pending = []
        for i, line in enumerate(lines, start=1):
            m = MARKER.match(line)
            if m:
                pending.append((m.group(1), m.group(2)))
                continue
            if pending:
                slot = out.setdefault((path, i), {"ruleid": set(), "ok": set()})
                for kind, rule in pending:
                    slot[kind].add(rule)
                pending = []
    return out


def expectations(markers):
    """Return (expected, forbidden) as sets of (rule, file, line)."""
    expected, forbidden = set(), set()
    for (path, line), slot in markers.items():
        for rule in slot["ruleid"]:
            expected.add((rule, path, line))
        for rule in slot["ok"]:
            forbidden.add((rule, path, line))
    return expected, forbidden


def score(findings, expected, forbidden):
    """Compare engine findings against the corpus.

    findings: iterable of (rule, path, line) restricted to the corpus.
    """
    found = set(findings)
    hit = found & expected
    missed = expected - found
    on_ok = found & forbidden
    stray = found - expected - forbidden
    return {"hit": hit, "missed": missed, "on_ok": on_ok, "stray": stray}


def report(name, result, expected):
    print(f"{name}: {len(result['hit'])}/{len(expected)} expected findings")
    for label, key in (("MISSED", "missed"), ("FLAGGED AN OK CASE", "on_ok"),
                       ("UNMARKED", "stray")):
        for rule, path, line in sorted(result[key]):
            print(f"  {label}: {rule} at {path}:{line}")


# Each engine names the rules its own way; the corpus uses one canonical id.
ALIASES = {
    "go/spnd/stacktrace-in-response": "go-stacktrace-in-response",
    "go/spnd/raw-request-to-sql-or-api": "go-raw-request-to-sql-or-api",
    "go/spnd/logger-without-logtag-context": "go-logger-without-logtag-context",
    "go/spnd/handler-without-metrics": "go-handler-without-metrics",
    "go/spnd/handler-without-start-end-log": "go-handler-without-start-end-log",
    "go/spnd/sensitive-data-in-log-tags": "go-sensitive-data-in-log-tags",
}

RULES = [
    "go-stacktrace-in-response",
    "go-raw-request-to-sql-or-api",
    "go-logger-without-logtag-context",
    "go-handler-without-metrics",
    "go-handler-without-start-end-log",
    "go-sensitive-data-in-log-tags",
]


def canonical(rule_id):
    """Map an engine-specific rule id onto the corpus id."""
    return ALIASES.get(rule_id, rule_id.split(".")[-1])
