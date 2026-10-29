# Side-by-side installations: overview

| | |
|---|---|
| **Status** | Owner answered the open questions on 2026-10-29: every recommendation accepted, Q11 clarified ([`99-open-questions.md`](99-open-questions.md)). Nothing is implemented. |
| **Baseline** | NetBird `v0.81.5` (commit `396976b8b`), branch `feature/side-by-side-installations` of `jeromerg/netbird` |
| **Input** | [`requirement-seed.md`](requirement-seed.md) |
| **Date** | 2026-10-28 |
| **Upstream ticket** | [netbirdio/netbird#446](https://github.com/netbirdio/netbird/issues/446) (open since 2022). No discussion opened from this work. |

This feature touches several areas that [`AGENTS.md`](../../AGENTS.md) calls high-risk: the
interface, DNS, firewall and routing integration, CLI and service flags, the daemon IPC address,
the config layout, and the installers. The design in this folder has to be agreed before any code
is written.

## 1. Problem

The owner's laptop must be connected to two NetBird networks at the same time. Each network is run
by a different organisation with its own self-hosted management server:

| Network | Management URL | Overlay range | Role |
|---|---|---|---|
| AIRSECOMA | `https://vpn.airsecoma.net:443` | `10.73.0.0/16` | primary |
| LFBB | `https://vpn.lfb-booking.net:443` | `100.125.0.0/16` | secondary |

The two networks are equivalent split tunnels: each carries only the traffic of its own range, and
neither uses an exit node. The roles only decide conflicts (section 5.6), and with this setup none
arises. AIRSECOMA configures no DNS nameserver group (`netbird status -d`, checked on 2026-10-29).

One NetBird daemon serves one active profile at a time, so profiles do not solve this. A throw-away
proof of concept (seed, section 2.1) ran a second, hand-patched copy of the client as a second
Windows service, and both networks worked at the same time. It showed that the set of machine-wide
names two clients fight over is finite. This specification turns that finding into a supported
feature.

## 2. Goals

- **G1.** Install the NetBird client more than once on the same Windows machine. Each installation
  is a complete client with its own service, daemon, state, peer identity, adapter, tray UI and
  settings, and all of them can be connected at the same time.
- **G2.** Choose a name and a colour for an installation in an advanced mode of the installer, or
  through unattended installer parameters.
- **G3.** Derive every machine-wide identifier of an installation from its name, in one place in
  the code.
- **G4.** Keep the installation without a name (the *default installation*) identical to today: the
  same identifiers, the same behaviour, no migration and no re-login.
- **G5.** Tell the tray icons apart by colour and name.
- **G6.** Make two connected installations behave predictably together: routes, DNS, exit nodes,
  firewall, SSH, and which one wins a conflict.
- **G7.** Keep the patch small and reviewable, so it can be rebased onto new upstream releases and
  proposed upstream later.

## 3. Non-goals

- **One daemon connected to several networks.** That is the full form of upstream #446 and needs a
  rework of the engine, routing and DNS. This feature runs several independent daemons instead.
- **Coordination between installations at runtime.** Installations do not talk to each other. Every
  conflict rule in [`04-coexistence.md`](04-coexistence.md) is decided by each installation's own
  configuration plus what it can see in the operating system.
- **Named installations on Linux, macOS and FreeBSD** in the first delivery. They are deferred, and
  those platforms must not regress (section 7).
- **Mobile, WASM and the embedded client** (`client/embed/`). Not applicable.
- **Moving a profile from one installation to another** in the first delivery (see
  [`99-open-questions.md`](99-open-questions.md), Q9).
- **Named installations from the MSI package** in the first delivery. The MSI keeps building the
  default installation only. A design is sketched in [`03-installer-ux.md`](03-installer-ux.md).
- **Auto-update of named installations built from the fork.** See `SBS-UPD` in
  [`01-requirements.md`](01-requirements.md).

## 4. Glossary

| Term | Meaning |
|---|---|
| **Installation** | One independent NetBird client on a machine: service, daemon, state directory, peer identities (profiles), overlay adapter, tray UI, installer registration. Called `instance` in code. |
| **Installation name** | The name chosen at install time, for example `lfbb`. Empty for the default installation. Case-insensitive; the canonical form is lowercase. |
| **Default installation** | The installation with an empty name. All its identifiers are today's identifiers. |
| **Named installation** | An installation with a non-empty name. Its identifiers are derived from the name. |
| **Display label** | The name as the user typed it (for example `LFBB`), used only in human-facing text. |
| **Installation colour** | A display attribute of a named installation. Drawn on its tray icon and window. Never affects networking. |
| **Installation marker** | The string `_<name>` that every derived Windows identifier carries (section 5.5). |
| **Identity package** | The single Go package that derives every machine-wide identifier from the installation name. Proposed path: `client/internal/instance`. |
| **Primary / secondary** | The role of an installation when two installations want the same machine-wide resource (default route, catch-all DNS, an identical route). See [`04-coexistence.md`](04-coexistence.md). |
| **Machine-wide identifier** | Any name, path, key, GUID, port or kernel object that is unique per machine or per user, so two installations would collide on it. Listed in [`02-identifier-matrix.md`](02-identifier-matrix.md). |
| **Unmodified upstream build** | An official NetBird release that has no knowledge of this feature. |

## 5. Decisions

The owner asked for four decisions to be made and challenged (D1 to D4). Three more came out of
the code review (D5 to D7).

### 5.1 D1: packaging model. Recommendation: separate copies (model A)

The seed's author leaned towards model B (one program, several instances, like WireGuard for
Windows). This specification recommends **model A: each installation is a separate copy** with its
own program folder, its own installer registration and its own update cycle, built on top of the
same runtime instance concept that B would need. A later move to B would then be a packaging change
only.

| Criterion | A: separate copies | B: one program, several instances |
|---|---|---|
| Matches the owner's request ("each installation is a complete, independent client") | Yes | Partly: shared binaries and a shared update cycle |
| Update policy | Each installation follows its own management server | One binary set for all networks. The updater takes its target from each management server's `AutoUpdateSettings` (`client/internal/engine.go:982-998`) and only ever upgrades (`client/internal/updater/manager.go:542-562`), so whichever organisation asks for the newest version upgrades the other organisation's client too, and a forced update restarts both networks |
| Fault isolation | A bad upgrade or a broken install affects one network | A bad upgrade takes every network down |
| Default installation can stay the official signed upstream package while named installations come from the fork | Yes, if named identifiers never match the default's cleanup patterns (D3) | No: there is one binary set |
| Windows lifecycle | One "Apps" entry, Start-menu entry and uninstaller per installation. Standard upgrade, repair and uninstall | One "Apps" entry for all. Adding or removing one network needs a custom maintenance mode or a CLI |
| Per-installation autostart | Works with the Wails fork as it is: it identifies its Run value by executable path and deletes other values pointing at the same executable (fork `autostart_windows.go:58-60,105-136`), so distinct folders keep the entries apart | Needs a change in the Wails fork: with one executable, enabling autostart for one instance deletes the other's entry |
| Windows 11 tray "show icon" choice | Kept per installation (keyed by executable path) | Shared between instances |
| Installer change | Parameterise existing names by installation (NSIS) | Instance management in the installer, uninstall of every instance, updater restarting every service and UI |
| MSI | Needs one MSI per name, or instance transforms. Deferred | Simpler: one product |
| Disk | One more copy of the binaries per installation, about 67 MB for v0.81.3 (measured) | One copy |
| CLI on `PATH` | Only the default installation's `netbird.exe` is on `PATH` | One `netbird.exe` |

The two strongest arguments for B (one copy, one update) are weak here: 67 MB per installation is
negligible, and a single update cycle is the coupling the owner does not want between two
organisations. WireGuard for Windows is a good precedent for one *manager* driving many tunnel
*configurations*; NetBird installations are complete clients, each with its own management server,
UI, updater and profiles.

### 5.2 D2: the installation name is a runtime concept. Agreed, with one channel per process

The name reaches every process through exactly one documented channel, and every identifier is
derived from it in the identity package:

| Process | Channel | Persisted in |
|---|---|---|
| Service (`netbird service run`) | `--instance <name>` on its command line. `NB_INSTANCE` is ignored | The service's registered command line, written by `netbird service install --instance <name>` |
| CLI (`netbird status`, `up`, ...) | `--instance <name>`, else `NB_INSTANCE`, else default | Not persisted |
| Tray UI (`netbird-ui.exe`) | `--instance <name>` | The Start-menu shortcut and the autostart entry |
| Installer | Advanced page, or `/INSTANCE=<name>` | The installation record (`SBS-ID-07`) and the above |

The service ignores `NB_INSTANCE` on purpose: `SetFlagsFromEnvVars` (`client/cmd/root.go:247-269`)
lets every `NB_*` variable override a flag *after* parsing, so a machine-wide `NB_INSTANCE` would
otherwise turn the default service into a second copy of a named one.

### 5.3 D3: the default installation keeps today's identifiers. Agreed, and strengthened

Every identifier of the default installation stays byte-for-byte as in v0.81.5, enforced by a
golden test. This specification adds a stronger rule: **no named identifier may match any pattern
that v0.81.5 uses to find or clean up its own resources.** Some cleanup in today's code matches by
prefix (NRPT rules: case-insensitive `NetBird-`, `client/internal/dns/host_windows.go:749`). With
this rule, an unmodified upstream build can run as the default installation next to named
installations from the fork, and older releases cannot delete a named installation's resources.

### 5.4 D4: one tray process per installation. Agreed

One UI process per installation, each started with `--instance <name>`, each with its own
single-instance lock, autostart entry, notification identity and colour. A single UI that shows
every installation would be a much larger change to the UI and its daemon connection, and the
owner's request is one icon per installation.

### 5.5 D5: the name qualifies the product token; it is not a literal prefix

The owner asked for the name to be "a prefix for everything". This specification recommends
attaching it **right after the product token, with an underscore**: `Netbird_lfbb` rather than
`lfbb-Netbird`. Reasons:

1. Several identifiers have a fixed leading part that must stay first: `99-` orders the SSH config
   include, `Global\` is the kernel namespace, `ProtectedPrefix\Administrators\` protects the daemon
   pipe, `wt` keeps the adapter out of every installation's ICE candidates
   (`client/internal/profilemanager/config.go:55-58`), and `io.netbird.` is a reverse-DNS
   identifier. A literal prefix cannot be applied to these, a qualifier can.
2. NetBird entries stay grouped in the services list, the registry, `ProgramData` and Program Files.
3. It follows common practice for multi-instance Windows software (`MSSQL$INSTANCE`,
   `WireGuardTunnel$name`).
4. **No machine-wide identifier that v0.81.5 creates contains `_`** after the product token
   (repository-wide search; the underscore occurs only in build artifact names such as
   `dist/netbird_windows_amd64` and in server-side `NETBIRD_*` environment variables, which nothing
   on the client matches against). The default's only prefix match (`netbird-`, case-insensitive)
   cannot match `netbird_`. That is what makes D3 provable. `$` was rejected because it breaks
   PowerShell paths; `-` was rejected because the default's NRPT cleanup would delete
   `NetBird-lfbb-*` rules.

Names are `^[a-z][a-z0-9]{0,11}$` after lowercasing: no separators, so the name `a` can never match
the name `ab`, and 12 characters keep `wt_<name>` within the 15-character Linux interface limit for
later. Full rules: `SBS-ID` in [`01-requirements.md`](01-requirements.md); every identifier:
[`02-identifier-matrix.md`](02-identifier-matrix.md).

### 5.6 D6: the default installation is primary, named installations are secondary

Installations do not coordinate, so every conflict rule must be static. A **secondary**
installation never installs a default route or exit node, never takes over system DNS (no NRPT
catch-all rule, no adapter `NameServer`), and adds its routes with a higher metric, so the primary
wins an identical route. The default installation is primary and behaves exactly as today. A named
installation is secondary unless it is explicitly made primary. For the owner's two networks the
role changes nothing that can be observed: their ranges differ and neither uses an exit node, so
there is no conflict to decide. The rule exists for setups where there is one, or for when one of
the networks adds an exit node later. Details: [`04-coexistence.md`](04-coexistence.md).

### 5.7 D7: colour as an overlay dot

The colour is drawn as a filled dot with a contrasting ring in the **top-left** corner of the
existing 256x256 tray PNGs, composited at runtime with the Go standard library. Tinting is rejected
because the state icons already use colour: an orange bird, and status badges in green, red, blue
and black. Those badges sit bottom-right and the update arrow top-right, so top-left is the only
free corner in every state. The palette is the colour-blind-safe Okabe-Ito set, ordered so that the
pre-selected colours cannot be mistaken for a state. Details: `SBS-COL` in
[`01-requirements.md`](01-requirements.md) and [`03-installer-ux.md`](03-installer-ux.md#44-colour-palette).

## 6. How the name flows

```text
  Installer (NSIS)                 advanced page or /INSTANCE=lfbb /COLOR=...
      |
      |-- writes  installation record   HKLM\SOFTWARE\NetBird\Installations\lfbb
      |-- runs    netbird.exe service install --instance lfbb
      |             -> service Netbird_lfbb, command line "... service run --instance lfbb ..."
      |-- creates Start-menu shortcut   netbird-ui_lfbb.exe --instance lfbb
      v
  Service Netbird_lfbb  --instance lfbb
      |  identity.For("lfbb") -> pipe, state dir, logs, adapter GUID and name, NRPT prefix,
      |                          firewall rule, SSH config file, role, ...
      v
  Daemon pipe \\.\pipe\ProtectedPrefix\Administrators\netbird_lfbb
      ^                                   ^
      |                                   |
  netbird --instance lfbb status      netbird-ui_lfbb.exe --instance lfbb
  (or NB_INSTANCE=lfbb)               lock io.netbird.ui.lfbb, Run value netbird_lfbb,
                                      AUMID NetBird_lfbb, colour from the installation record
```

## 7. Platform scope

| Area | Windows | Linux | macOS | FreeBSD, mobile, WASM, embedded |
|---|---|---|---|---|
| Identity package, `--instance` flag | In scope | Code is shared; named installations rejected at startup until supported | Same as Linux | Not applicable |
| Adapter, DNS, firewall, routing | In scope | Deferred | Deferred | Not applicable |
| Service, pipe or socket, state, logs | In scope | Deferred (`release_files/systemd/netbird@.service` exists but is not packaged) | Deferred | Not applicable |
| Tray UI, colour | In scope | Deferred | Deferred (template icons are monochrome) | Not applicable |
| Installer | NSIS in scope; MSI deferred | Not applicable | Not applicable | Not applicable |
| **No regression for the default installation** | Required | Required | Required | Required |

On every platform where named installations are not supported, `--instance` with a non-empty name
fails with a clear error, and the default installation's identifiers come from the identity package
with today's values.

## 8. Delivery and upstreamability

The change is split so that each step builds, passes tests and makes sense on its own, matching the
stacked-PR advice in `AGENTS.md`. Sizes are estimates of hand-written lines.

1. **Identity package, no behaviour change.** Introduce `client/internal/instance`, move today's
   constants behind it, add the golden test that pins every default identifier. A pure refactor
   that upstream can review in isolation (about 300 lines).
2. **Windows isolation.** `--instance` flag and its precedence, service registration, pipe, state,
   logs, per-user files, adapter GUID and name, port, NRPT prefix, firewall rules, SSH config,
   updater temp folder, suggested commands (about 400 lines).
3. **Coexistence rules.** Roles, interface metric, exit-node and catch-all suppression in a
   secondary, uplink and next-hop exclusion of NetBird adapters, overlap detection and warnings,
   policy split (about 350 lines).
4. **Tray UI.** Lock, autostart, toast identity, quick-actions event, WebView2 folder, titles,
   colour overlay (about 300 lines plus React).
5. **NSIS installer.** Advanced page, unattended parameters, per-installation registration and
   uninstall (about 250 lines of NSIS).
6. **Fork build.** A GitHub Actions workflow in the fork that builds an unsigned NSIS installer.
7. **Later.** MSI named installations, instance-aware updater, Linux and macOS.

Steps 1 to 5 would each be one upstream PR, stacked in that order, if upstream agrees on the design
in #446. Each stays under the size `AGENTS.md` asks for.

## 9. Corrections to the seed

The inventory in the seed was checked against v0.81.5. The full list is in
[`02-identifier-matrix.md`](02-identifier-matrix.md#14-corrections-to-the-seed). The ones that change
the design:

- Both installations write the **same NRPT key paths** (`NetBird-Match-0`, ...), so they overwrite
  each other's DNS rules even without the prefix cleanup.
- Each installation keeps its control traffic off **its own** tunnel only. It treats another
  NetBird adapter as a normal uplink, so one installation's exit node pins the other's management,
  signal, relay and peer traffic into its tunnel (`client/internal/routemanager/systemops/systemops_windows.go:919-928`).
  The network monitor reacts only to `/0` changes (`client/internal/networkmonitor/check_change_windows.go:29`),
  so nothing re-pins those sockets when that tunnel goes away.
- NetBird finds its own adapter **by name** in several places (routes, uplink exclusion, interface
  monitor, `netsh` shutdown). Interface names must be unique per installation, not just GUIDs.
- The firewall rule of the second installation never allows its own overlay IP: the existing rule
  is reused and carries the first installation's `localip`
  (`client/firewall/uspfilter/interface_allower_windows.go:41-49`).
- A second service cannot be registered even with a different `-s` name, because its DisplayName
  `Netbird` is hard-coded (`client/cmd/service.go:110`) and Windows requires it to be unique.
- The shared update folder `%ProgramData%\Netbird\tmp-install` is wiped by every daemon on each
  connect (`client/internal/connect.go:313-316`).
- Commands suggested in error messages (`netbird down; netbird up ...`) carry no installation, so a
  command copied from LFBB's error would disconnect AIRSECOMA.
- On macOS, every daemon deletes all NetBird-tagged routes at start, including another daemon's
  (`client/internal/routemanager/systemops/systemops_unix.go:63-120`). This is one reason macOS is
  deferred.

## 10. Documents

| File | Content |
|---|---|
| [`00-overview.md`](00-overview.md) | This file |
| [`01-requirements.md`](01-requirements.md) | Numbered requirements with acceptance criteria |
| [`02-identifier-matrix.md`](02-identifier-matrix.md) | Every machine-wide identifier: today, default, named, rule, code location; seed corrections |
| [`03-installer-ux.md`](03-installer-ux.md) | Installer flow, advanced page, unattended parameters, upgrade, repair, uninstall, MSI sketch |
| [`04-coexistence.md`](04-coexistence.md) | Two connected installations: routes, DNS, exit nodes, overlaps, firewall, SSH, roles |
| [`05-test-plan.md`](05-test-plan.md) | Automated and manual acceptance scenarios |
| [`99-open-questions.md`](99-open-questions.md) | Decisions for the owner, each with a recommended answer |
| [`requirement-seed.md`](requirement-seed.md) | The input brief |
