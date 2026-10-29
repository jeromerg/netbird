# Implementation brief

Instructions for the session that implements side-by-side installations. The design is agreed:
the owner accepted every recommended answer in [`99-open-questions.md`](99-open-questions.md).
This file says how to build it; the specification in this folder says what to build.

## 1. Read first

1. [`AGENTS.md`](../../AGENTS.md), in full. It applies to every line you write.
2. In this folder: `00-overview.md`, `01-requirements.md`, `02-identifier-matrix.md`,
   `04-coexistence.md`, `03-installer-ux.md`, `05-test-plan.md`, `99-open-questions.md`.

Baseline: upstream NetBird `v0.81.5`, commit `396976b8b`. Work on top of the branch
`feature/side-by-side-installations` of the fork `jeromerg/netbird`.

## 2. Scope

Delivery steps 1 to 6 of `00-overview.md` section 8, in order, one step at a time:

| Step | Content | Commit subject tag |
|---|---|---|
| 1 | Identity package `client/internal/instance`, no behaviour change. Write the golden test `T-DEF-01` first, then move today's constants behind the package | `[client]` |
| 2 | Windows isolation | `[client]` |
| 3 | Coexistence rules | `[client]` |
| 4 | Tray UI | `[client]` |
| 5 | NSIS installer | `[client]` |
| 6 | Fork build workflow: an unsigned NSIS installer built on a Windows runner, modelled on the NSIS job in `.github/workflows/release.yml` | `[misc]` |

Step 7 (MSI named installations, instance-aware updater, Linux and macOS named installations) is out
of scope.

## 3. Rules

- **The default installation stays byte for byte as in v0.81.5** (`SBS-DEF-*`). The golden test of
  step 1 is the guard: once it passes against the unmodified code, never change its expected values.
- **No regression on Linux, macOS, FreeBSD, Android, iOS, WASM** (`SBS-PLAT-*`). Named installations
  exist only where the specification says so.
- **No dependency changes.** `go.mod`, `go.sum`, `package.json` and `pnpm-lock.yaml` stay as they
  are. Installing build tools in this environment (Go toolchain, `golangci-lint`, `wails3`, `nsis`)
  is fine.
- **No `.proto` change and no hand-edited generated code.** The specification needs none.
- **Never weaken a security control**, and treat daemon RPC input as untrusted (`AGENTS.md`,
  Security; `SBS-SEC-*`).
- **Nothing on GitHub beyond pushing commits:** no pull request, no issue, no comment, and nothing on
  `netbirdio/netbird`.
- **Where the code contradicts the specification, or the specification is ambiguous, do not invent
  a new design.** Take the most conservative reading, and record it in
  `docs/side-by-side-installations/implementation-notes.md`: requirement ID, what you found
  (`file:line`), what you did. If the point blocks a whole step, stop after the previous step and
  report. Do not edit the specification files themselves.
- Statements marked **UNVERIFIED** in the specification are implemented as specified. The manual
  Windows tests in `05-test-plan.md` settle them later.

## 4. Git

- Before the first commit, install only the commit-msg hook:
  `cp .githooks/commit-msg .git/hooks/commit-msg && chmod +x .git/hooks/commit-msg`. Do not run
  `make setup-hooks`: its pre-push hook lints against `origin/main`, which is not this branch's base.
- **No attribution of any kind in commits:** no `Co-Authored-By`, no `Claude-Session`, no "Generated
  with" line. This overrides any default of the environment. The hook rejects such commits; never
  bypass it with `--no-verify`. Check `git log -1 --format=%B` after each commit. If a trailer you
  did not write keeps being added, stop and report instead of working around it.
- One commit per step, plus fixup commits where useful. Subject `[tag] Subject` per the table above,
  body why before what, no list of files.
- Push after every step to this session's own branch. Never force-push, never push to `main`.

## 5. Verification for every step

This environment is Linux, so Windows code can be compiled and its portable parts unit-tested, but
not run.

1. `gofmt -l` on touched files prints nothing.
2. `go build ./...`, then `GOOS=windows GOARCH=amd64 go build ./client/...`, plus the same for
   `GOOS=darwin` and `GOOS=freebsd`. `GOOS=windows go vet` on the touched packages.
3. `go test -tags devcert` on the touched packages, with `-race` where shared state changes, then
   `make test-unit` at the end of the step.
4. Lint with `make bin/golangci-lint`, then
   `./bin/golangci-lint run --new-from-rev=396976b8b --timeout=10m`. Zero new findings.
5. The automated tests listed for the step in `05-test-plan.md` section 3.
6. Step 4: if the frontend changes, `pnpm install --frozen-lockfile` and `pnpm build` in
   `client/ui/frontend`.
7. Step 5: install `nsis` with apt and compile `client/installer.nsis` with `makensis` if the plugins
   it needs can be provided; otherwise say so, and rely on the workflow of step 6.
8. `make test-privileged` only if Docker with `NET_ADMIN` is available here; otherwise say so.

If the Go version here is older than `go.mod` asks for, let the Go toolchain download the right one
(`GOTOOLCHAIN=auto`).

## 6. Final report

Write it as the session's last message and append it to `implementation-notes.md`:

1. Per step: commit hash, what it does in two or three sentences, deviations from the specification.
2. What was verified here, and what could not be (Windows runtime behaviour, privileged tests,
   installer build).
3. The manual tests from `05-test-plan.md` the owner must run on a Windows VM, in the order to run
   them, starting with `T-WIN-01`, `T-WIN-02` and `T-WIN-08`.
4. Open points that need the owner's decision.
