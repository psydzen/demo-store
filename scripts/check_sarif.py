"""Score a SARIF file against the fixture corpus.

Usage: python3 scripts/check_sarif.py <name> <file.sarif>
"""

import json
import sys

import fixtures


def sarif_findings(path, restrict_to=None):
    data = json.load(open(path, encoding="utf-8"))
    out = []
    for run in data.get("runs", []):
        for res in run.get("results", []):
            rule = fixtures.canonical(res.get("ruleId", "?"))
            for loc in res.get("locations", []):
                phys = loc.get("physicalLocation", {})
                uri = phys.get("artifactLocation", {}).get("uri", "")
                uri = uri.replace("file://", "").lstrip("/")
                line = phys.get("region", {}).get("startLine", 0)
                if restrict_to and not uri.startswith(restrict_to):
                    continue
                out.append((rule, uri, line))
    return out


def main():
    name, path = sys.argv[1], sys.argv[2]
    # A blind run reads a marker-free copy of the corpus; map it back.
    prefix = ".analysis/blind/"
    found = [(r, u[len(prefix):] if u.startswith(prefix) else u, l)
             for r, u, l in sarif_findings(path)]
    found = [f for f in found if f[1].startswith(fixtures.CORPUS)]
    markers = fixtures.load_markers()
    expected, forbidden = fixtures.expectations(markers)
    result = fixtures.score(found, expected, forbidden)
    fixtures.report(name, result, expected)
    return 0 if not result["missed"] and not result["on_ok"] else 2


if __name__ == "__main__":
    sys.exit(main())
