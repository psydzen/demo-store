"""Build the cross-engine comparison from the three result files."""

import json
import os
import subprocess
import sys
from collections import defaultdict

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import fixtures

EXCLUDE_PREFIXES = ("internal/analysisfixtures", ".analysis/", "internal/grpcapi/paymentspb")


def load_sarif(path):
    data = json.load(open(path, encoding="utf-8"))
    out = set()
    for run in data.get("runs", []):
        for res in run.get("results", []):
            rule = fixtures.canonical(res.get("ruleId", "?"))
            for loc in res.get("locations", []):
                pl = loc.get("physicalLocation", {})
                uri = pl.get("artifactLocation", {}).get("uri", "").replace("file://", "").lstrip("/")
                line = pl.get("region", {}).get("startLine", 0)
                if uri.startswith(EXCLUDE_PREFIXES):
                    continue
                out.add((rule, uri, line))
    return out


def load_semgrep(path):
    data = json.load(open(path, encoding="utf-8"))
    out = set()
    for r in data["results"]:
        uri = r["path"].lstrip("./")
        if uri.startswith(EXCLUDE_PREFIXES):
            continue
        out.add((fixtures.canonical(r["check_id"]), uri, r["start"]["line"]))
    return out


def by_site(findings):
    """Collapse to (rule, file) with a count: engines anchor lines differently."""
    d = defaultdict(int)
    for rule, uri, _ in findings:
        d[(rule, uri)] += 1
    return d


def main():
    engines = {}
    if os.path.exists(".analysis/semgrep-oss.json"):
        engines["semgrep-oss"] = load_semgrep(".analysis/semgrep-oss.json")
    if os.path.exists(".analysis/semgrep-pro.json"):
        engines["semgrep-pro"] = load_semgrep(".analysis/semgrep-pro.json")
    if os.path.exists(".analysis/codeql.sarif"):
        engines["codeql"] = load_sarif(".analysis/codeql.sarif")
    if os.path.exists(".analysis/agent-repo.sarif"):
        engines["agent"] = load_sarif(".analysis/agent-repo.sarif")

    names = list(engines)
    print("Findings per rule (repository, fixtures excluded)\n")
    header = "rule".ljust(36) + "".join(n.ljust(14) for n in names)
    print(header)
    print("-" * len(header))
    for rule in fixtures.RULES:
        row = rule.ljust(36)
        for n in names:
            row += str(sum(1 for r, _, _ in engines[n] if r == rule)).ljust(14)
        print(row)
    row = "TOTAL".ljust(36)
    for n in names:
        row += str(len(engines[n])).ljust(14)
    print(row)

    print("\n\nAgreement per rule, at (rule, file) granularity\n")
    sites = {n: by_site(engines[n]) for n in names}
    for rule in fixtures.RULES:
        keys = {n: {k for k in sites[n] if k[0] == rule} for n in names}
        common = set.intersection(*keys.values()) if keys else set()
        print(f"\n{rule}")
        print(f"  in every engine: {len(common)} file(s)")
        for n in names:
            only = keys[n] - set.union(*[keys[m] for m in names if m != n]) if len(names) > 1 else keys[n]
            if only:
                print(f"  only {n}: " + ", ".join(sorted(f for _, f in only)))


if __name__ == "__main__":
    main()
