// Package analysisfixtures holds the shared corpus the three static-analysis
// engines are tested against.
//
// Every rule owns a pair of cases: one marked `// ruleid: <rule>` that the rule
// must report, and one marked `// ok: <rule>` that it must not. The markers sit
// on the line directly above the line the finding is expected on.
//
// The package compiles, so CodeQL sees it in the build. It is excluded from the
// repository-wide runs by path, so the fixtures never inflate the counts.
package analysisfixtures
