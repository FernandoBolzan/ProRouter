# ProRouter Development Worklog

## [2026-07-01] - Planning and Documentation Setup
*   **Activity:** Created standard project documentation and memory structures under `DEV/`.
*   **Revision Pass:** Refined plans to add active details about:
    *   OAuth 2.0 PKCE flow logic targeting local loopbacks (`http://localhost`).
    *   Assuring memory efficiency during streaming event transformation.
    *   Addressing SQLite database write lockups under highly concurrent setups using Go channels buffers.
*   **Files Created/Updated:**
    *   `DEV/INDEX.md` - Index mapping.
    *   `DEV/CONTEXT.md` - Context, vision, and core pillars.
    *   `DEV/SPECS/ACTIVE.md` - Architecture, structures, and interfaces.
    *   `DEV/SPECS/CHECKLISTS.md` - Step-by-step checklist of development phases (now with Phase 4 OAuth focus).
    *   `DEV/VERIFY.md` - Test strategies and performance targets.
    *   `DEV/HANDOFF.md` - System state summary.
*   **Status:** Plans reviewed, reinforced against common engineering bottlenecks, and approved. Ready for execution.

## [2026-07-01] - E2E Tests, Distribution Docs, and URL Fixes
*   **Activity:** Fixed all 3 failing E2E tests; removed stale `prorouter.dev` references throughout the project.
*   **Changes:**
    *   Fixed nil-slice bug in `GetAPIKeys()` and `GetAuditLogs()` (`database.go`) — returned `nil` instead of `[]`.
    *   Fixed context key type mismatch in `proxy.go` — used raw `string("api_key_id")` instead of `middleware.APIKeyIDKey` (custom `contextKey` type), causing panic in playground handler.
    *   Added recover guard in `handlePlayground()` for defensive error handling.
    *   Replaced all `github.com/prorouter/…` → `github.com/FernandoBolzan/ProRouter` in Go imports, install scripts, Scoop/Homebrew manifests, and DISTRIBUTION.md.
    *   Removed all references to `prorouter.dev` domain (README, ACTIVE.md, DISTRIBUTION.md) — install instructions now point to GitHub releases and raw.githubusercontent.com.
*   **E2E Results:** 14/14 passing (Playwright, chromium).
*   **Status:** E2E tests green, all project URLs pointing to GitHub only. No external domain dependencies.

## [2026-07-01] - Install Scripts, NPM Package, Release Workflow
*   **Activity:** Fixed install scripts with fallback to `go install`; rewrote npm download script; created GitHub Actions release workflow.
*   **Changes:**
    *   `scripts/install.sh` — added graceful fallback to `go install` when no GitHub release exists; handles missing `curl` gracefully.
    *   `scripts/install.ps1` — same fallback logic for Windows.
    *   `cli-npm/scripts/download-binary.js` — rewrote without broken external imports (`tar-stream` not imported, `adm-zip` not in deps); now uses native `tar`/`Expand-Archive` via `child_process.execSync`.
    *   `cli-npm/scripts/cleanup.js` — created (was missing, referenced by package.json).
    *   `cli-npm/package.json` — renamed from `@prorouter/cli` to `@fernandobolzan/prorouter-cli`; added `repository` field.
    *   `.github/workflows/release.yml` — GoReleaser + npm publish on tag push `v*`.
    *   `gateway-go/.goreleaser.yaml` — builds linux/darwin/windows amd64+arm64 with CGO_ENABLED=0.
*   **Status:** Ready for first release. User needs to push a tag and create npm org/package.
