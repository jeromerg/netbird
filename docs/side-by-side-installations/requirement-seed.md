# Requirement seed: side-by-side installations of the NetBird client

> **What this file is.** A *seed*: the input brief for an agent whose job is to turn it into a full
> requirements specification. It is not the specification itself, and it is not a design to
> implement as-is. It records the owner's request, the evidence gathered so far, and the questions
> the specification must answer.
>
> - **Baseline:** NetBird `v0.81.5`, branch `feature/side-by-side-installations` of the fork
>   `jeromerg/netbird`. The remote `upstream` is `netbirdio/netbird`.
> - **Date:** 2026-10-28.

---

## 1. Your task (instructions to the specifying agent)

Write the requirements specification for the feature described in section 3, in this folder
(`docs/side-by-side-installations/`).

**Deliverables:**

1. **`00-overview.md`**: the problem, goals, non-goals, glossary, the chosen *installation model*
   (section 6.1) with its rationale, and a link map to the other files.
2. **`01-requirements.md`**: numbered requirements (`SBS-<AREA>-<NN>`), each with:
   - the requirement, using MUST / SHOULD / MAY
   - its rationale
   - testable acceptance criteria

   Group them by the areas in section 6.
3. **`02-identifier-matrix.md`**: for every machine-global resource in section 5, a table with:
   - the identifier *today*
   - the identifier for the **default** installation, which must stay identical to today
   - the identifier for a **named** installation
   - the derivation rule
   - the code location
4. **`03-installer-ux.md`**: the installer flow, including the advanced page, silent and unattended
   parameters (NSIS and MSI), upgrade, repair and uninstall of a single installation, and mock-ups
   (ASCII is fine).
5. **`04-coexistence.md`**: how two connected installations behave together: routes, DNS, exit
   nodes, overlapping CIDRs, firewall, SSH, and priority between installations.
6. **`05-test-plan.md`**: manual and automated acceptance scenarios (section 7 has a starting list).
7. **`99-open-questions.md`**: every decision you could not make from this seed, each with your
   recommended answer, so the owner can reply quickly.

**Rules:**

- **Do not write production code.** Reading code, prototyping in a scratch branch to verify a claim,
  and quoting `file:line` are all encouraged. Every identifier you cite must be checked against the
  baseline tag; the inventory in section 5 was produced by an agent and may contain mistakes.
- Follow [`AGENTS.md`](../../AGENTS.md). This feature touches several areas it lists as
  **high-risk**: client system integration (interface, DNS, firewall, routing), CLI and service
  flags, config file format, daemon IPC, and installers. That is why the design comes first. Do not
  open pull requests or upstream discussions; the owner decides that.
- Prefer requirements that keep the patch **small and upstreamable**. The upstream request is
  [netbirdio/netbird#446](https://github.com/netbirdio/netbird/issues/446) ("support connecting to
  multiple networks/management instances simultaneously", open since 2022). Upstream described the
  rework as large and pointed to firewall and routing-table races. A design that keeps the default
  installation byte-for-byte unchanged has the best chance of being accepted.
- **Windows is the primary target.** Linux and macOS must not regress. For each area, state whether
  the platform is in scope, deferred, or not applicable (section 6.10).

---

## 2. Background and motivation

The owner's laptop must be connected to **two NetBird networks at the same time**. The networks are
run by different organisations, each with its own self-hosted management server:

| Network   | Management URL                   | Overlay range    | Rank      |
|-----------|----------------------------------|------------------|-----------|
| AIRSECOMA | `https://vpn.airsecoma.net:443`  | `10.73.0.0/16`   | primary   |
| LFBB      | `https://vpn.lfb-booking.net:443`| `100.125.0.0/16` | secondary |

Since v0.52 the client has *profiles*, which store several accounts. **Only one profile can be
active at a time.** Switching profiles disconnects the other network, so profiles do not solve this.

### 2.1 Evidence from a throw-away proof of concept (2026-10-28)

A second copy of the v0.81.3 client was built with five constants changed and installed as a second
Windows service (`Netbird2`, `npipe://netbird2`, `NB_STATE_DIR=C:\ProgramData\Netbird2`, interface
`wt1`, port 51821). **Both networks were connected at the same time and worked:**

- each adapter had its own GUID
- per-network DNS split through NRPT worked (`*.airsecoma.internal` and `*.netbird.selfhosted` were
  each resolved by their own network)
- stopping or restarting the second service left the first service's DNS rules, firewall rule and
  connectivity intact

The patch:

1. a different wintun adapter GUID (ldflag `device.CustomWindowsGUIDString`)
2. `NRPTKeyPrefix` changed from `NetBird-` to `NetBird2-`
3. `firewallRuleName` changed from `Netbird` to `Netbird2`
4. `NetBirdSSHConfigFile` changed from `99-netbird.conf` to `99-netbird2.conf`
5. the auto-updater disabled, plus service `DisplayName` set from the service name

The PoC was reverted. It showed that the isolation surface is **finite and enumerable**. It also
exposed what a proper feature must solve:

- **No second tray UI.** `client/ui/main.go:300-305` uses a fixed single-instance ID,
  `UniqueID: "io.netbird.ui"`, so a second UI started with `-daemon-addr` exits immediately.
- **Manual install.** Everything had to be installed by hand, and nothing survives an upgrade of
  the official package.
- **Profiles cannot be moved between installations.** `netbird profile remove` deregisters the peer
  on its management server, even when another daemon is using a copy of that profile. Moving a
  profile between installations must therefore be a supported operation, or be explicitly
  forbidden.

---

## 3. The feature, as requested by the owner

> - Support **side-by-side installations**.
> - Extend the **installer** to ask, in an **advanced mode**, for an **installation name** and a
>   **color**.
>   - The **name** is used as a **prefix for everything**: network adapter, Windows service,
>     firewall rule, and so on.
>   - The **color** is applied to the **taskbar (tray) icon**, either by tinting the existing icon or
>     by drawing an overlay such as a coloured circle, so the user can see which tray icon controls
>     which installation.

The owner sees this as the simplest way to get consistent, parallel access to several networks: each
installation is a complete, independent NetBird client that does not know about the others.

---

## 4. Terminology (proposed; refine in `00-overview.md`)

- **Installation** (in code, probably `instance`): one independent client with its own daemon,
  state, identity, adapter, UI and settings.
- **Default installation**: the one with no name. Its identifiers MUST stay exactly as they are
  today, so existing users, scripts, MDM policies and docs keep working without migration.
- **Named installation**: one created with a name, for example `lfbb`. Every machine-global
  identifier is derived from that name.
- **Installation color**: a display attribute only. It never affects networking.

---

## 5. Inventory of machine-global resources (baseline v0.81.5, Windows)

These collide when two installations exist. The spec MUST cover each row, or justify leaving it
shared. Paths are relative to the repo root.

### 5.1 Network, DNS, firewall

| Resource | Today | Where | What happens with two installations |
|---|---|---|---|
| Wintun adapter GUID | constant `{f2f29e61-d91f-4d76-8151-119b20c4bdeb}`; test-only override `CustomWindowsGUIDString` | `client/iface/device/device_windows.go:20,48-54` | The second adapter cannot be created. NLA network profile and per-interface registry keys are keyed by this GUID. |
| Interface name | `wt0` (`--interface-name`) | `client/iface/configurer/name.go:6` | Duplicate alias. The ICE blacklist already skips `wt*` (`internal/profilemanager/config.go:55-58`). |
| WireGuard port | 51820 | `client/iface/iface.go:32` | Falls back to a free port (`internal/connect.go:828-851`). Low risk. |
| NRPT rule names | prefix `NetBird-` → `NetBird-Match-<n>`, `NetBird-ExemptLocal` | `client/internal/dns/host_windows.go:43-68` | **Destructive.** `removeDNSMatchPolicies` deletes *every* key with that prefix, in both the local and the GPO store, on every DNS apply, on restore and on unclean-shutdown cleanup (`:313,603,628-651,727-755`). |
| GPO NRPT store handling | `SOFTWARE\Policies\Microsoft\Windows NT\DNSClient\DnsPolicyConfig` | `host_windows.go:55,665-722` | One installation may delete the store while it is "empty" for itself. |
| "Original" nameserver capture | reads every interface except its own | `host_windows.go:153-215` | Installation B records installation A's overlay DNS IP as an upstream fallback. |
| Catch-all DNS (`.` rule) | per installation | `host_windows.go:386-393` | Two route-all installations compete; only one can win. |
| Windows Firewall rule | `Netbird`, `Netbird-v6` | `client/firewall/uspfilter/interface_allower_windows.go:20,41-86` | B skips adding its rule because a rule with that name exists. Whichever stops first deletes the rule for both. |
| SSH client config | `%ProgramData%\ssh\ssh_config.d\99-netbird.conf` | `client/ssh/common.go:20-23`, `client/ssh/config/manager.go` | Last writer wins, either cleanup deletes it, and `ProxyCommand` / `Match exec` do **not** pass `--daemon-addr`, so they always reach the default daemon. |
| Fixed local ports (opt-in) | metrics `127.0.0.1:9191`; netstack SOCKS5 `1080`; pprof `6969` | `internal/localmetrics/localmetrics.go:23`, `client/iface/netstack/proxy.go:12`, `client/cmd/pprof.go` | The second bind fails. |

### 5.2 Daemon, service, files

| Resource | Today | Where | What happens with two installations |
|---|---|---|---|
| Daemon pipe | `npipe://netbird` → `\\.\pipe\ProtectedPrefix\Administrators\netbird`, falls back to `\\.\pipe\netbird` | `internal/daemonaddr/pipe.go`, `client/cmd/service_pipe_windows.go:26-41` | **Silent misrouting.** B falls back to the unprotected name, but clients try the protected name first and reach A. `resolve_pipe_windows.go:39` hard-codes `"netbird"`. |
| Windows service | name `Netbird` (`-s`), DisplayName hard-coded `"Netbird"` | `client/cmd/service.go:76-79,110-111` | Install fails or reuses the existing service. The Event Log source follows the service name. |
| State and profiles | `%ProgramData%\Netbird` (`NB_STATE_DIR` overrides) | `client/configs/configs.go:16-29`, `internal/profilemanager/service.go:64-87` | Shared keys and identity: both would be the same peer. |
| Service parameters | `<StateDir>\service.json` | `client/cmd/service_params.go` | B's `service install` inherits A's saved arguments if the state dir is shared. |
| Logs | `%ProgramData%\Netbird\client.log`, `netbird.err` | `client/cmd/root.go:136-162`, `client/server/panic_windows.go:31` | Hard-coded; ignores `NB_STATE_DIR`. |
| Per-user files | `%APPDATA%\netbird\` (`active_profile.txt`, `ui-preferences.json`, `gui-client.log`) | `internal/profilemanager/config.go:247-268`, `client/ui/preferences/store.go:25` | Shared between both installations' CLI and UI. |
| Debug named event | `Global\NetbirdDebugTriggerEvent` | `client/cmd/debug_windows.go:19` | Shared (opt-in). |

### 5.3 UI (Wails v3)

| Resource | Today | Where | What happens with two installations |
|---|---|---|---|
| Single-instance lock | `UniqueID: "io.netbird.ui"` | `client/ui/main.go:300-305` | The second UI exits. **This blocked the PoC.** |
| Daemon address | `-daemon-addr` flag, default `npipe://netbird` | `client/ui/main.go:244-260`, `client/ui/grpc.go:66-71` | Must be per installation. |
| Autostart | `HKCU\...\Run` value `netbird` | Wails fork `autostart_windows.go`; `client/ui/services/autostart.go:41-52` | Overwritten, so only one UI autostarts. The fork already supports `AutostartOptions.Identifier`. |
| First-run autostart gate | checks for an existing footprint | `client/ui/autostart_default.go:55-67` | B sees A's footprint and never auto-enables. |
| Toast identity | AUMID `NetBird`, COM CLSID `{0E1B4DE7-...}` | `client/ui/main.go:274-282`, `client/netbird.wxs:23-27,66` | Clicking a toast opens the wrong installation's UI. |
| Quick-actions event | `Global\NetBirdQuickActionsTriggerEvent` | `client/ui/signal_windows.go:15` | One signal triggers every UI. |
| Tray tooltip and title | `"NetBird"` | `client/ui/tray.go:183,301`, i18n `tray.tooltip` | Two identical icons and tooltips. |
| **Tray icons** (colour feature) | PNGs in `client/ui/assets/netbird-systemtray-*.png` (+`-dark`, `-macos`, `-mono`) | chosen in `client/ui/tray_icon.go:54-136` (`iconForState`), applied in `applyIcon` (`:14-43`); `client/ui/icons.go` | **Hook point**: post-process the selected PNG bytes (tint or composite an overlay) per state and per theme. Windows needs single-frame PNG, not `.ico` (`icons.go:7`). macOS uses monochrome *template* icons, which lose any tint. |

### 5.4 Installers and updater

| Resource | Today | Where | What happens with two installations |
|---|---|---|---|
| NSIS install dir | forced `$PROGRAMFILES64\Netbird` in `.onInit` | `client/installer.nsis:171` | `/D=` and the registry value are ignored. |
| NSIS existing-install detection | `HKLM\...\Uninstall\Netbird` → uninstalls the existing one | `installer.nsis:170-193` | **Installing B removes A.** |
| NSIS pages | MUI2: Welcome, License, Directory, InstFiles, Finish. No install-time custom page (only the uninstall "delete data" page, built with nsDialogs) | `installer.nsis:52-54,86,103-124` | The advanced page is new work (nsDialogs is already used). |
| NSIS registry, shortcuts, PATH | `App Paths\Netbird`, `Uninstall\Netbird`, `Netbird.lnk`, `EnVar` PATH | `installer.nsis:16-23,218-234` | Overwritten; both directories end up on PATH. |
| NSIS service setup | `netbird service install` / `start` with defaults | `installer.nsis:274-279` | This is where per-installation flags go. |
| NSIS uninstall | `taskkill /im Netbird-ui.exe /f`; optional `RMDir /r %ProgramData%\Netbird` | `installer.nsis:282-336` | Kills every installation's UI; may delete other data. |
| MSI identity | fixed `UpgradeCode 6456ec4e-...`, `MajorUpgrade` | `client/netbird.wxs:4,14,18` | MSI B upgrades (removes) A. |
| MSI service | `ServiceInstall Name="NetBird"`, args without `--` | `netbird.wxs:33-45` | Same service; the argument format is suspicious (verify). |
| Auto-updater | detects the installer type from `Uninstall\Netbird`; runs the standard package; kills **all** `netbird-ui.exe`; restarts the default service; temp dir `%ProgramData%\Netbird\tmp-install` | `client/internal/updater/installer/*_windows.go` | Updating one installation breaks or overwrites the others. |
| Release build | test build only in `.github/workflows/release.yml:722-869`; signed installers are built in the external `netbirdio/sign-pipelines` repo | — | The fork needs its own build and signing story (section 6.9). |

### 5.5 Policy and other platforms

- **GPO/MDM:** `HKLM\Software\Policies\NetBird` (`client/mdm/policy_windows.go:18`, `docs/netbird.admx`)
  applies to all installations.
- **Linux:** a `release_files/systemd/netbird@.service` template exists but is not packaged. It still
  shares `/var/lib/netbird` and `wt0`. Linux already has environment overrides
  (`NB_NFTABLES_TABLE`, `NB_FWMARK_BASE`) that Windows lacks.
- **macOS:** utun default `utun100`, scutil keys `State:/Network/Service/NetBird-%s/DNS`.

---

## 6. Areas the specification must cover

### 6.1 Installation model: decide and justify

Compare at least these two and recommend one:

- **(A) Full side-by-side copies.** Each named installation has its own program directory and
  binaries, its own installer registration (Uninstall key, MSI UpgradeCode or instance transform)
  and its own updater cycle. This is closest to the owner's wording ("side-by-side installations").
  It is simple to reason about, but duplicates binaries, and the updater and installer must become
  installation-aware.
- **(B) One program, several instances.** One copy of the binaries and one installer registration
  register several daemon instances (service, state, adapter, UI autostart entry). The installer
  "advanced mode" creates or removes instances, and a CLI (`netbird instance add|remove|list`) does
  the same. Precedent: WireGuard for Windows, which runs one service per tunnel
  (`WireGuardTunnel$<name>`) from one install. One update covers all instances.

Whichever is chosen, the **instance name must be a runtime concept** in the daemon, CLI and UI (for
example a `--instance` flag or `NB_INSTANCE` env, persisted in the service arguments). Every
identifier in section 5 must be derived from it in **one place**, a single "instance identity"
package. The installer only *sets* the name. Hard-coded strings must not be re-patched at build
time.

### 6.2 Installation name

- **Charset and length.** Must be valid at once for: service name, adapter alias, NRPT key, firewall
  rule, file and directory names, a named-pipe segment, registry keys, a systemd instance and a
  launchd label. A suggested starting point is `^[a-z][a-z0-9-]{0,15}$`. Decide how to handle case
  (Windows is case-insensitive) and reserved words (`default`, `netbird`, ...).
- **Prefix or suffix, and separators.** The owner said "prefix". The spec must check that no
  installation's cleanup can ever match another's resources. Example: the default prefix
  `NetBird-` must not match `NetBird-<name>-...`, and name `a` must not match name `a-b`. Put the
  matching rule in writing, with tests.
- **Deterministic adapter GUID**, e.g. UUIDv5 over a fixed namespace and the name, so it is stable
  across reinstalls. The default installation keeps the current GUID.
- **Renaming:** supported or not?

### 6.3 Color

- Picked in the installer's advanced page: a palette or free choice; the default for named
  installations; none for the default installation.
- Stored per installation, and changeable afterwards (UI settings? CLI?).
- Rendering:
  - tint versus overlay badge (dot or ring), across **all** tray states (connected, connecting,
    disconnected, error, needs-login, update-*) and light and dark taskbars
  - contrast and accessibility rules (the state must stay readable; colour-blind safe palette)
  - the same colour in the UI window header and in toasts
- The tooltip and window title include the installation name, e.g. `NetBird — LFBB`.
- Platforms: Windows first. macOS template icons are monochrome, so define a fallback such as a
  letter badge or no colour.

### 6.4 Installer (NSIS and MSI)

- **Advanced mode:** how the user gets there (button or checkbox on the welcome page, or a separate
  page), the fields (name, colour, optionally install directory and "start with Windows"), and
  validation messages.
- **Unattended mode:** e.g. NSIS `/S /INSTANCE=lfbb /COLOR=#E8A33D`, MSI properties
  `INSTANCENAME`, `INSTANCECOLOR`.
- Detecting an existing installation **with the same name** (upgrade or repair) versus a different
  name (coexist). Never uninstall another installation (today `installer.nsis:170-193` does).
- MSI multi-instance: instance transforms, or a per-instance UpgradeCode or ProductCode. Explain how
  `MajorUpgrade` still works for each installation.
- Start menu and desktop shortcuts, Uninstall entry, `App Paths`, and the PATH (which `netbird.exe`
  wins?), all per installation.
- Uninstall of one installation removes only its own service, adapter, NRPT keys, firewall rules,
  autostart value, AUMID, ProgramData folder and UI processes (not `taskkill` by image name).

### 6.5 Daemon and system integration

Define per-installation derivation, in the identifier matrix, for every row of 5.1 and 5.2. Also:

- The SSH config's `ProxyCommand` and `Match exec` must carry the installation's daemon address.
- Everything that ignores `NB_STATE_DIR` today (logs, panic log, updater temp) must follow the
  installation root.
- Unclean-shutdown cleanup must only touch the installation's own resources.

### 6.6 Coexistence semantics (`04-coexistence.md`)

- **Overlapping overlay CIDRs** (e.g. two NetBird-cloud accounts, both in `100.64.0.0/10`), and
  overlapping routed networks: detect and warn, refuse, or let the installation with priority win?
- **Exit nodes / default route** in more than one installation.
- **DNS:** multiple route-all (`.`) nameserver groups; match-domain conflicts.
- **Priority / rank:** the owner thinks in terms of primary and secondary installations. Should
  there be an explicit rank that affects route metric, NRPT ordering and which installation owns
  the catch-all?
- **Control traffic:** `GetBestInterface` (`systemops_windows.go`) excludes only its own adapter. B's
  management, signal and relay traffic must never be routed through A's tunnel unless that is
  intended.
- **Same management server in two installations:** allowed? Each needs its own peer identity.

### 6.7 UI

- One UI process per installation (single-instance key derived from the name), **or** one UI that
  shows every installation. Recommend one. The owner's request implies one tray icon per
  installation, told apart by colour.
- Autostart per installation (Run value from `AutostartOptions.Identifier`), per-installation
  preferences, log and WebView2 data directory, AUMID and toast activator.
- The quick-actions event and similar global events: per installation.
- Profiles inside a named installation keep working as today.

### 6.8 CLI

- Address an installation by name, e.g. `netbird --instance lfbb status`, without the user knowing
  pipe names. `NB_INSTANCE` env as an alternative.
- `netbird instance list` (name, colour, service state, management URL, overlay IP), if model B (or
  even with A).
- `netbird service install --instance <name>` performs the complete, consistent registration.

### 6.9 Updates, build and distribution (fork-specific)

- Auto-update per installation (model A) or once for all (model B). It must never kill another
  installation's UI or restart the wrong service.
- The fork does not have upstream's signing pipeline. Specify how the owner builds and installs:
  unsigned or self-signed, GitHub Actions in the fork, how to bundle wintun (`load_wgnt_from_rsrc`
  embeds it). Also cover what happens if the official NetBird package is installed next to it, or
  over it.
- Rebasing onto new upstream releases: keep the diff small and isolated; list the files expected to
  change.

### 6.10 GPO/MDM and other platforms

- Global policy versus per-installation keys (e.g. `HKLM\Software\Policies\NetBird\Instances\<name>`).
- Linux (systemd template units, nftables table and fwmark via the existing env overrides,
  resolved per link) and macOS (launchd label, utun, scutil keys): in scope, deferred or not
  applicable. Requirements must at least guarantee no regression there.

---

## 7. Acceptance scenarios (starting list for `05-test-plan.md`)

1. Fresh machine: install the default installation, then a named `lfbb` installation with colour
   orange. Both connect to different management servers at the same time. Both tray icons are
   visible and distinguishable. DNS for both domains resolves through the right network.
2. Stop, restart or kill -9 one installation (and reboot). The other's adapter, routes, NRPT rules,
   firewall rules, SSH config, autostart entry and UI stay intact.
3. Upgrade one installation (interactive and silent). The other is untouched and keeps running.
4. Uninstall one installation. Nothing of the other is removed, and nothing of the removed one is
   left behind (service, adapter, registry, ProgramData, Run value, shortcuts).
5. An existing default installation upgrades to the new build and is byte-for-byte identical in its
   identifiers: same GUID, service, pipe, NRPT names, firewall rule, paths. No re-login.
6. Overlapping CIDRs between two installations produce the specified behaviour (warn, refuse, or
   priority).
7. `ssh` to a peer of the named installation works through NetBird SSH.
8. Invalid names (too long, illegal characters, duplicates, reserved) are rejected in the installer
   and the CLI with clear messages.

---

## 8. Author's recommendation (non-binding; challenge it)

- **Model B (one program, several instances)**, with the installer's advanced page as the
  user-facing way to add an instance. It avoids N copies of binaries, N updaters and MSI
  multi-instance pain, and it matches the proven WireGuard-for-Windows design. If the owner insists
  on literal separate installations (model A), keep the same runtime instance concept underneath,
  so A only differs in packaging.
- **Default installation = empty instance name = today's identifiers.** No migration, and upstream
  can review the change as a pure refactor plus opt-in.
- **One `instance` package** that derives every identifier (service, pipe, state dir, log paths,
  GUID, interface name, NRPT prefix, firewall rule, SSH file, UI lock, AUMID, Run value,
  updater temp). All call sites in section 5 read from it, and nothing else hard-codes `Netbird`.
- **Colour as an overlay dot**, bottom-right, ~40 % of the icon, with a contrasting ring, composited
  at runtime on the existing PNGs. The state icons stay unchanged and recognisable.
