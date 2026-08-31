"""Run the Semgrep rules against the fixture corpus and score the result.

Usage: python3 scripts/check_semgrep.py [--pro]
"""

import json
import subprocess
import sys

import fixtures


def main():
    pro = "--pro" in sys.argv
    cmd = ["semgrep", "--metrics=off", "--config", "rules/semgrep",
           "--json", "--quiet", fixtures.CORPUS]
    if pro:
        cmd.insert(1, "--pro")

    proc = subprocess.run(cmd, capture_output=True, text=True)
    if not proc.stdout.strip():
        print(proc.stderr[-2000:], file=sys.stderr)
        return 1
    data = json.loads(proc.stdout)
    for err in data.get("errors", []):
        print("semgrep error:", str(err)[:300], file=sys.stderr)

    found = {(r["check_id"].split(".")[-1], r["path"], r["start"]["line"])
             for r in data["results"]}

    markers = fixtures.load_markers()
    expected, forbidden = fixtures.expectations(markers)
    result = fixtures.score(found, expected, forbidden)
    fixtures.report("semgrep" + (" --pro" if pro else " (oss)"), result, expected)
    return 0 if not result["missed"] and not result["on_ok"] else 2


if __name__ == "__main__":
    sys.exit(main())
