"""Drop the fixture findings from a SARIF file before it is published.

The corpus in internal/analysisfixtures is deliberately wrong code whose only
job is to prove the rules still work. It must not reach the security tab.

Filtering here rather than through the CodeQL config is deliberate: for Go with
a traced build, `paths-ignore` does not suppress these results — the extractor
follows the compiler, and everything `go build ./...` compiles is analysed.

Usage: python3 scripts/strip_fixture_results.py <file.sarif> [more.sarif ...]
"""

import json
import sys

EXCLUDE_PREFIXES = (
    "internal/analysisfixtures/",
    "internal/grpcapi/paymentspb/",
)


def uri_of(result):
    for loc in result.get("locations", []):
        uri = loc.get("physicalLocation", {}).get("artifactLocation", {}).get("uri")
        if uri:
            return uri.replace("file://", "").lstrip("/")
    return ""


def strip(path):
    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)

    removed = 0
    for run in data.get("runs", []):
        keep = []
        for res in run.get("results", []):
            if uri_of(res).startswith(EXCLUDE_PREFIXES):
                removed += 1
            else:
                keep.append(res)
        run["results"] = keep

    with open(path, "w", encoding="utf-8") as fh:
        json.dump(data, fh)

    kept = sum(len(r.get("results", [])) for r in data.get("runs", []))
    print(f"{path}: dropped {removed} fixture findings, kept {kept}")


def main():
    for path in sys.argv[1:]:
        strip(path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
