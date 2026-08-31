"""Decide whether a SARIF report should stop the build.

Blocking rules carry `error`; advisory rules carry `warning`. Everything is
reported either way — only the exit code differs.

Usage: python3 scripts/gate_sarif.py <file.sarif>
"""

import collections
import json
import sys


def levels(path):
    data = json.load(open(path, encoding="utf-8"))
    counts = collections.Counter()
    blocking = []
    for run in data.get("runs", []):
        rules = {r.get("id"): r for r in run.get("tool", {}).get("driver", {}).get("rules", [])}
        for res in run.get("results", []):
            level = res.get("level") or _default_level(rules.get(res.get("ruleId"), {}))
            counts[level] += 1
            if level == "error":
                blocking.append(res)
    return counts, blocking


def _default_level(rule):
    return rule.get("defaultConfiguration", {}).get("level", "warning")


def where(res):
    loc = (res.get("locations") or [{}])[0].get("physicalLocation", {})
    return "%s:%s" % (
        loc.get("artifactLocation", {}).get("uri", "?"),
        loc.get("region", {}).get("startLine", "?"),
    )


def main():
    counts, blocking = levels(sys.argv[1])
    total = sum(counts.values())
    print("%d findings: %s" % (total, dict(counts) or "none"))

    if not blocking:
        print("Nothing at error level. Advisory findings do not stop the build.")
        return 0

    print("\n%d finding(s) at error level stop the build:" % len(blocking))
    for res in blocking:
        print("  %s  %s" % (res.get("ruleId", "?"), where(res)))
    return 1


if __name__ == "__main__":
    sys.exit(main())
