# Requirements

Keywords: **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT** and **MAY** as in RFC 2119. Each
requirement has a rationale and acceptance criteria; test IDs (`T-*`) refer to
[`05-test-plan.md`](05-test-plan.md). Identifier values are in
[`02-identifier-matrix.md`](02-identifier-matrix.md); behaviour between installations is explained
in [`04-coexistence.md`](04-coexistence.md). "Named" means an installation with a non-empty name,
"default" the one without. Examples use the name `acme` with label `ACME`.

Areas: [ID](#sbs-id-identity-and-naming) ·
[DEF](#sbs-def-default-installation) ·
[SVC](#sbs-svc-service-and-daemon-ipc) ·
[FS](#sbs-fs-files-and-state) ·
[CLI](#sbs-cli-command-line) ·
[NET](#sbs-net-adapter-interface-ports) ·
[DNS](#sbs-dns-dns) ·
[FW](#sbs-fw-firewall) ·
[SSH](#sbs-ssh-ssh) ·
[COEX](#sbs-coex-coexistence-and-roles) ·
[PROF](#sbs-prof-profiles) ·
[UI](#sbs-ui-tray-ui) ·
[COL](#sbs-col-colour) ·
[INS](#sbs-ins-installer) ·
[UPD](#sbs-upd-updates-build-distribution) ·
[POL](#sbs-pol-group-policy-and-mdm) ·
[PLAT](#sbs-plat-other-platforms) ·
[SEC](#sbs-sec-security)

---

## SBS-ID: identity and naming

### SBS-ID-01 The installation name is a runtime parameter (MUST)

**Requirement.** The daemon, the CLI and the tray UI MUST accept the installation name as
`--instance <name>`. An empty or absent name selects the default installation. The name MUST NOT be
compiled into a binary or patched in at build time.

**Rationale.** One binary serves every installation's role; the installer only sets the name. This
is what keeps the patch upstreamable and lets a later packaging change (model B) reuse it.

**Acceptance.** The same `netbird.exe` runs as the default service and, from another folder, as
`Netbird_acme`, with only the command line differing (`T-ID-01`). A build without `--instance`
anywhere produces today's behaviour (`T-DEF-01`).

### SBS-ID-02 Name syntax (MUST)

**Requirement.** A name MUST match `^[a-z][a-z0-9]{0,11}$` after ASCII lowercasing. Input is
case-insensitive; the lowercase form is the canonical name used in every identifier; the form as
typed is kept as the *label* for display. Anything else MUST be rejected with the messages in
[`03-installer-ux.md`](03-installer-ux.md#45-validation-messages).

**Rationale.** The name must be valid at once in a service name, adapter alias, NRPT key, firewall
rule, file name, pipe name, registry key, AUMID, and later a systemd instance and a launchd label.
Excluding `-` and `_` makes prefix matching between names impossible (`SBS-ID-05`). Twelve
characters keep `wt_<name>` within Linux's 15-character interface limit for a later port.

**Acceptance.** Table-driven test of valid and invalid names, plus fuzzing that every accepted name
yields valid identifiers on all rows of the matrix (`T-ID-02`).

### SBS-ID-03 Reserved names (MUST)

**Requirement.** The names `default` and `netbird` MUST be rejected for a named installation.
`--instance default` MUST select the default installation in the CLI.

**Rationale.** `default` gives scripts a way to address the default installation when `NB_INSTANCE`
is set in their environment. `netbird` produces confusing identifiers.

**Acceptance.** `T-ID-02`.

### SBS-ID-04 One place derives every identifier (MUST)

**Requirement.** A single package (proposed `client/internal/instance`) MUST derive every identifier
in [`02-identifier-matrix.md`](02-identifier-matrix.md) from the installation name and role. Other
code MUST NOT hard-code a value that the package derives. Values MUST be read through the package
when they are used, not cached in package-level variables computed before the name is known.

**Rationale.** Today several values are computed in `init()` before flags are parsed
(`client/cmd/root.go:130-158`, `client/configs/configs.go:16-29`); a name set later would be
ignored by them. One package makes the derivation reviewable in one file and testable as a table.

**Acceptance.** A repository search finds none of the derived literals (`"Netbird"` as a service
name, `"NetBird-"`, `"wt0"`, `"io.netbird.ui"`, `"99-netbird.conf"`, the adapter GUID, `PROGRAMDATA`
joined with `Netbird`, ...) outside the identity package and its tests (`T-ID-04`).

### SBS-ID-05 Named identifiers are disjoint (MUST)

**Requirement.** Named identifiers MUST be formed by the rules of
[`02-identifier-matrix.md`](02-identifier-matrix.md#1-derivation-rules). No named identifier may
equal, or be matched by any prefix or pattern matcher of, the default installation's v0.81.5 code
or another named installation's code. The only intended match is the `wt` interface prefix.

**Rationale.** Cleanup in today's code matches by prefix (NRPT). Disjointness lets an unmodified
upstream build run as the default installation and makes cross-installation deletion impossible by
construction.

**Acceptance.** Property test over generated names: every named identifier differs from every
default identifier and from every other name's identifiers, and none is matched by the v0.81.5
matchers listed in section 13 of the matrix (`T-ID-03`).

### SBS-ID-06 Stable adapter GUID (MUST)

**Requirement.** A named installation's adapter GUID MUST be `UUIDv5(f2f29e61-d91f-4d76-8151-119b20c4bdeb,
"adapter/" + name)`. The default installation MUST keep `{f2f29e61-d91f-4d76-8151-119b20c4bdeb}`.

**Rationale.** Windows keys per-adapter TCP/IP, DNS and NetBT settings and the network profile by
GUID. A stable GUID avoids orphaned registry keys and a new "Network 2" profile on every reinstall.

**Acceptance.** Known vector: `acme` gives `{930adb3f-6a74-541d-b49f-0166bb47c7a5}`; never equal to the
default GUID; uninstall and reinstall of `acme` produce the same GUID (`T-ID-05`, `T-WIN-05`).

### SBS-ID-07 Installation record (MUST)

**Requirement.** The installer MUST write `HKLM\SOFTWARE\NetBird\Installations\<name>` with `Label`,
`Color`, `Role`, `InstallDir` and `Version` for every named installation, and remove it on
uninstall. The key MUST be readable by users and writable only by administrators. The daemon MUST
NOT need it to connect: everything the daemon needs comes from its command line. It MAY read it
for warnings (`SBS-COEX-02`).

**Rationale.** The UI needs the colour and label, the uninstaller needs to know which installation it
belongs to, and `netbird instance list` needs an inventory. Keeping the daemon independent of the
record means a broken record cannot change networking.

**Acceptance.** After install the key exists with the values given; a standard user can read but not
write it; after uninstall it is gone; deleting it does not stop the service from connecting
(`T-INS-04`).

### SBS-ID-08 No renaming (MUST NOT)

**Requirement.** An installation MUST NOT be renamed in place. Changing the name means uninstalling
and installing again, which creates a new peer.

**Rationale.** The name is in the service name, adapter GUID, folders and registry. A rename touches
every row of the matrix for little value. See Q8.

**Acceptance.** No rename command or installer path exists; the documentation states the procedure.

### SBS-ID-09 Name precedence (MUST)

**Requirement.** `netbird service run` MUST take the name from `--instance` only and ignore
`NB_INSTANCE`. Every other command MUST use `--instance` if given, else `NB_INSTANCE`, else the
default. This is an exception to `SetFlagsFromEnvVars`, which today lets `NB_*` override flags.

**Rationale.** `SetFlagsFromEnvVars` (`client/cmd/root.go:247-269`) applies environment variables
after parsing. A machine-wide `NB_INSTANCE` would otherwise turn the default service into a second
copy of a named installation, sharing its adapter, rules and identity.

**Acceptance.** With `NB_INSTANCE=acme` set machine-wide and services restarted, the default service
still runs as the default installation (`T-ID-06`). `NB_INSTANCE=acme netbird --instance default
status` queries the default installation.

---

## SBS-DEF: default installation

### SBS-DEF-01 Identical identifiers (MUST)

**Requirement.** Every identifier of the default installation MUST be byte-for-byte equal to its
v0.81.5 value, as listed in the "Default installation" column of the matrix.

**Rationale.** Existing users, scripts, MDM policies, documentation and support procedures depend on
them. It also makes the first upstream PR a pure refactor.

**Acceptance.** A golden test pins every default value (`T-DEF-01`); a before/after snapshot of a
real upgrade shows no difference (`T-WIN-01`).

### SBS-DEF-02 Upgrade without migration (MUST)

**Requirement.** Upgrading a default installation from an official v0.81.x to a build with this
feature MUST keep the same peer, overlay IP, profiles, settings and service, with no re-login.

**Rationale.** Owner's requirement; a re-registration costs an admin action on the management side.

**Acceptance.** `T-WIN-01`.

### SBS-DEF-03 Unchanged behaviour without named installations (MUST)

**Requirement.** On a machine with no named installation, the default installation MUST write the
same files, registry values, rules and routes, and behave the same on the network, as v0.81.5.
Changes that are no-ops without another NetBird adapter (`SBS-COEX-06`, `SBS-DNS-05`) are allowed.

**Rationale.** Regression safety for every existing user, and the argument for upstream review.

**Acceptance.** `T-WIN-01` and the full existing test suite pass unchanged.

### SBS-DEF-04 Coexistence with an unmodified upstream default (MUST)

**Requirement.** A named installation MUST work next to a default installation that is an official
upstream build (v0.81.3 or later), without either deleting or changing the other's resources. The
limitations that remain MUST be documented: the upstream build still treats a named adapter as an
uplink for destinations the named installation routes, and its installer and updater end every
`netbird-ui.exe` process.

**Rationale.** During the fork period the owner can keep GLOBEX on the official, signed and
auto-updated package. This is only possible with packaging model A and disjoint identifiers.

**Acceptance.** `T-WIN-08`.

### SBS-DEF-05 Existing knobs keep their meaning (MUST)

**Requirement.** `--daemon-addr`, `-s/--service`, `NB_STATE_DIR`, `--interface-name`,
`--wireguard-port`, `--log-file` and `--config` MUST keep their current meaning for the default
installation. For a named installation an explicit value of these flags MUST override the derived
value, except `NB_STATE_DIR` (`SBS-FS-02`).

**Rationale.** Scripts and the proof of concept rely on them; explicit values are deliberate.

**Acceptance.** `T-CLI-03`.

---

## SBS-SVC: service and daemon IPC

### SBS-SVC-01 Service registration (MUST)

**Requirement.** `netbird service install --instance acme` MUST register service `Netbird_acme` with
DisplayName `NetBird (ACME)`, description `NetBird mesh network client (ACME)`, start type
automatic, restart on failure as today, and the command line
`service run --instance acme --instance-role <role> --log-level <lvl> --daemon-addr npipe://netbird_acme --log-file %ProgramData%\Netbird_acme\client.log`
plus the optional flags written today. The label comes from the installation record when it exists,
else the canonical name is used.

**Rationale.** The service's identity must be visible with standard tools (`sc qc Netbird_acme`).
DisplayName uniqueness is enforced by the Service Control Manager, so the fixed `Netbird`
(`client/cmd/service.go:110`) blocks a second service even with a different `-s`.

**Acceptance.** `sc qc` and `sc qdescription` show the values; the default service registration is
unchanged (`T-SVC-01`).

### SBS-SVC-02 Service subcommands follow the name (MUST)

**Requirement.** `service start|stop|restart|status|uninstall|reconfigure` with `--instance acme`
MUST act on `Netbird_acme` only. `-s` keeps working; `-s` together with a non-empty `--instance` and
a different derived name MUST be an error.

**Rationale.** One flag must be enough; two contradicting flags must not pick one silently.

**Acceptance.** `T-SVC-02`.

### SBS-SVC-03 Pipe with today's protections (MUST)

**Requirement.** A named daemon MUST serve `npipe://netbird_acme` with the same protections as the
default: the `ProtectedPrefix\Administrators\` name first, the same security descriptor, and caller
identity from the pipe client token (`client/internal/ipcauth`). Clients MUST apply the same server
verification to the derived name.

**Rationale.** The pipe crosses the privilege boundary (`AGENTS.md`, "Daemon RPC input is
untrusted"). `npipe://<name>` already works generically (`client/internal/daemonaddr/pipe.go:38-82`);
the requirement is that nothing is weakened for named installations.

**Acceptance.** `T-SEC-01`.

### SBS-SVC-04 No ambiguous pipe (MUST)

**Requirement.** A named daemon running as a service MUST refuse to start when its protected pipe
name is already in use, instead of falling back to the unprotected name.

**Rationale.** A taken protected name means a duplicate daemon for the same installation. Falling
back to `\\.\pipe\netbird_acme` would let clients reach either daemon depending on start order,
which is the misrouting the proof of concept hit.

**Acceptance.** Start a second daemon with `--instance acme` while the service runs: it exits with an
error naming the pipe (`T-SVC-03`).

### SBS-SVC-05 Fail before writing (MUST)

**Requirement.** `service install --instance <name>` MUST check that neither the service name nor
the DisplayName is in use before it writes anything (service `Environment` value, `service.json`,
Event Log source).

**Rationale.** The kardianos fork writes the `Environment` value before its existence check, so a
failed install can overwrite another service's environment or leave an orphan key.

**Acceptance.** Installing `acme` twice leaves the first registration byte-identical (`T-SVC-04`).

### SBS-SVC-06 Uninstall removes the service's own items (MUST)

**Requirement.** `service uninstall --instance acme` MUST remove the service, its Event Log source and
its `Environment` value, and SHOULD remove its wintun adapter if the service left it behind,
identified by the derived GUID.

**Rationale.** Leftovers block a later reinstall under the same name or accumulate as phantom
adapters.

**Acceptance.** `T-WIN-04`.

### SBS-SVC-07 No legacy TCP for named installations (MUST)

**Requirement.** A named daemon MUST NOT listen on, probe or migrate the legacy loopback TCP address
(`tcp://127.0.0.1:41731`), and `MigrateLegacy` (`client/internal/daemonaddr/pipe.go:97`) MUST apply
only to the default installation.

**Rationale.** The migration rewrites the legacy address to `npipe://netbird`, which would move a
named installation onto the default installation's pipe.

**Acceptance.** `T-SVC-02`.

---

## SBS-FS: files and state

### SBS-FS-01 State directory (MUST)

**Requirement.** A named installation MUST keep every daemon-side file in `%ProgramData%\Netbird_<name>`:
profile configs, `active_profile.json`, `state.json`, `service.json`, `client.log` and its rotations,
`netbird.err`, and `tmp-install`.

**Rationale.** Shared files mean a shared peer identity (the WireGuard key is in the profile JSON)
and cross-installation crash cleanup (`state.json`).

**Acceptance.** After connecting `acme`, no file under `%ProgramData%\Netbird` changed (`T-WIN-02`).

### SBS-FS-02 `NB_STATE_DIR` (MUST)

**Requirement.** The default installation MUST honour `NB_STATE_DIR` as today. A named installation
MUST ignore it and log a warning when it is set.

**Rationale.** `NB_STATE_DIR` is a process environment variable and may be set machine-wide; honouring
it would point every named installation at the same folder.

**Acceptance.** With `NB_STATE_DIR=C:\x` machine-wide, `acme` still uses `%ProgramData%\Netbird_acme`
and logs the warning (`T-FS-02`).

### SBS-FS-03 No hard-coded paths (MUST)

**Requirement.** The log directory (`client/cmd/root.go:131-162`), the panic log
(`client/server/panic_windows.go:31-32`), the updater temp folder
(`client/internal/updater/installer/installer_run_windows.go:37`) and the two independent readers of
`NB_STATE_DIR` (`client/configs/configs.go:17`, `client/internal/profilemanager/service.go:70`) MUST
get their values from the identity package.

**Rationale.** These are the places that ignore the state directory today.

**Acceptance.** `T-ID-04`; default values unchanged (`T-DEF-01`).

### SBS-FS-04 Per-user directory (MUST)

**Requirement.** The CLI and UI of a named installation MUST use `%APPDATA%\netbird_<name>` for
`active_profile.txt`, `<profileID>.state.json`, `ui-preferences.json` and `gui-client.log`.

**Rationale.** The CLI sends the profile named in `active_profile.txt` to whichever daemon it talks
to (`client/cmd/up.go:126-143`), so a shared file makes a profile selected for one installation
appear in the other.

**Acceptance.** `netbird --instance acme profile select x` does not change
`%APPDATA%\netbird\active_profile.txt` (`T-FS-03`).

### SBS-FS-05 Crash replay is local (MUST)

**Requirement.** On start, a daemon MUST replay only its own `state.json`.

**Rationale.** The replay removes NRPT rules, routes and the SSH config recorded there.

**Acceptance.** `T-WIN-03` (kill one service, restart it, the other keeps all its rules).

---

## SBS-CLI: command line

### SBS-CLI-01 Global `--instance` flag (MUST)

**Requirement.** Every `netbird` command MUST accept `--instance <name>` (env `NB_INSTANCE`,
precedence `SBS-ID-09`) and derive the daemon address and per-user directory from it.

**Rationale.** Users address an installation by name, not by pipe path.

**Acceptance.** `netbird --instance acme status` shows ACME's peers while the default installation is
connected to GLOBEX (`T-CLI-01`).

### SBS-CLI-02 Explicit address wins (MUST)

**Requirement.** An explicit `--daemon-addr` MUST override the address derived from `--instance`.

**Rationale.** Scripts and diagnostics need a way to point anywhere.

**Acceptance.** `T-CLI-03`.

### SBS-CLI-03 `netbird instance list` (SHOULD)

**Requirement.** `netbird instance list` SHOULD print, for the default installation (if its service
exists) and every installation record: name, label, colour, role, service state, version, program
folder, and, when the daemon answers, management URL and overlay IP. It SHOULD work without
administrator rights.

**Rationale.** One command to see what is installed and connected. Model A has no single program
that knows all installations otherwise.

**Acceptance.** `T-CLI-02`.

### SBS-CLI-04 Reaching a named daemon from any CLI (MUST)

**Requirement.** A named daemon MUST be reachable with `--daemon-addr npipe://netbird_<name>` from a
v0.81.5 CLI.

**Rationale.** When the default installation is an unmodified upstream build, its CLI has no
`--instance` flag.

**Acceptance.** Official v0.81.5 `netbird.exe --daemon-addr npipe://netbird_acme status` works
(`T-WIN-08`).

### SBS-CLI-05 Status shows the installation (MUST)

**Requirement.** For a named installation, `netbird status` MUST show the installation name and role
(in text, JSON and YAML output). For the default installation the output MUST stay unchanged.

**Rationale.** Two terminals, two networks: the user must see which one answered.

**Acceptance.** `T-CLI-01`; golden output of the default unchanged.

### SBS-CLI-06 Unsupported platforms (MUST)

**Requirement.** On every platform where named installations are not supported, a non-empty
`--instance` MUST fail with `named installations are supported on Windows only` and a non-zero exit
code.

**Rationale.** Fail early rather than half-isolate.

**Acceptance.** `T-PLAT-01`.

### SBS-CLI-07 Suggested commands carry the installation (MUST)

**Requirement.** Every command that the daemon, the CLI or the UI suggests to the user in an error or
hint (for example `netbird down; netbird up ...`, `netbird logout`, `netbird debug bundle ...`) MUST
include `--instance <name>` when it concerns a named installation.

**Rationale.** These hints are copied and pasted. Without the installation, a command copied from
ACME's error runs against GLOBEX, and its `netbird down` disconnects GLOBEX
(`client/internal/ipcauth/privileged.go:131-150`, `client/server/ssh_gate.go:95-220`,
`client/server/debug_gate.go:74,87`, `client/ui/services/guarded.go:219-226`).

**Acceptance.** `T-CLI-04`.

### SBS-CLI-08 Kubeconfig entries (SHOULD)

**Requirement.** `netbird --instance acme kubernetes ...` SHOULD name its kubeconfig user entry
`netbird_acme` instead of `netbird`.

**Rationale.** Both installations otherwise write the same user entry into `~/.kube/config`
(`client/cmd/kubernetes.go:225-270`).

**Acceptance.** Manual check of `~/.kube/config` after using both.

---

## SBS-NET: adapter, interface, ports

### SBS-NET-01 Adapter (MUST)

**Requirement.** A named installation MUST create its wintun adapter with the derived GUID
(`SBS-ID-06`) and the interface name `wt_<name>`.

**Rationale.** NetBird finds its own adapter by name in six places (see
[`04-coexistence.md`](04-coexistence.md#3-adapters-and-overlay-addresses)); GUID and name must both
be unique.

**Acceptance.** `Get-NetAdapter` shows `wt0` and `wt_acme` with their GUIDs (`T-WIN-05`).

### SBS-NET-02 Custom interface names (MUST)

**Requirement.** An explicit `--interface-name` for a named installation MUST start with `wt` and
MUST NOT equal the name of another existing adapter; otherwise `up` fails with a clear error.

**Rationale.** The `wt` prefix keeps the adapter out of every installation's ICE candidates and
marks it as a NetBird adapter for `SBS-COEX-06`.

**Acceptance.** `T-NET-02`.

### SBS-NET-03 WireGuard port (MUST)

**Requirement.** The default WireGuard port of a named installation MUST be `51820 + k` with
`k = 1 + FNV-1a-32(name) mod 100`, written into new profile configs. The existing fallback to a free
port stays.

**Rationale.** The default installation keeps 51820 whatever the start order, so port forwards and
firewall exceptions made for it keep working.

**Acceptance.** `acme` uses 51836; the default keeps 51820 after a reboot in either start order
(`T-NET-03`).

### SBS-NET-04 Interface metric of a secondary (MUST)

**Requirement.** A secondary installation MUST set the interface metric of its adapter to `1000` for
IPv4 and IPv6. A primary MUST leave it automatic.

**Rationale.** The primary wins identical routes, including the on-link overlay route
([`04-coexistence.md`](04-coexistence.md#4-routed-networks)).

**Acceptance.** `Get-NetIPInterface -InterfaceAlias wt_acme` shows `1000`; `wt0` is automatic
(`T-COEX-03`).

### SBS-NET-05 Opt-in local listeners (SHOULD)

**Requirement.** For a named installation the default addresses of local metrics, pprof and netstack
SOCKS5 SHOULD be derived with the same `k` (`9191 + k`, `6969 + k`, `1080 + k`).

**Rationale.** Enabling an opt-in feature on both installations should not fail on a port clash;
pprof even exits the process on bind failure.

**Acceptance.** Enable local metrics on both installations: both endpoints answer (`T-NET-05`).

### SBS-NET-06 Own adapter only (MUST)

**Requirement.** Interface teardown (`netsh interface set interface <name> admin=disable`, wintun
close) MUST act only on the installation's own adapter.

**Rationale.** Teardown is name-keyed (`client/iface/iface_destroy_windows.go:12-19`).

**Acceptance.** `T-WIN-03`.

### SBS-NET-07 No legacy routing in a named installation (MUST)

**Requirement.** A named installation MUST use socket pinning ("advanced routing") and ignore
`NB_USE_LEGACY_ROUTING`, logging a warning when it is set.

**Rationale.** Legacy routing adds `/32` host routes to management, signal, relay and STUN through the
physical adapter and refcounts them per process
(`client/internal/routemanager/systemops/systemops_windows.go:196-296`). Two installations reaching
the same relay or STUN address would share a route row that the first one to stop deletes.

**Acceptance.** `T-NET-06`.

---

## SBS-DNS: DNS

### SBS-DNS-01 Own NRPT keys (MUST)

**Requirement.** A named installation MUST name its NRPT keys `NetBird_<name>-Match-<n>` and
`NetBird_<name>-ExemptLocal`, and its cleanup (apply, restore, unclean shutdown) MUST match only keys
starting with `NetBird_<name>-`, case-insensitively, in both the local and the group-policy store.

**Rationale.** Today both installations write the same keys and each one deletes every key starting
with `netbird-` (`client/internal/dns/host_windows.go:43-68,313,603,628-655,727-755`).

**Acceptance.** With both connected, `Get-DnsClientNrptPolicy` lists both installations' rules;
reconnecting either leaves the other's rules untouched (`T-DNS-01`).

### SBS-DNS-02 Never touch other rules (MUST NOT)

**Requirement.** An installation MUST NOT create, modify or delete an NRPT key outside its own prefix.

**Rationale.** Restates `SBS-DNS-01` for the default installation's existing code: its prefix
`netbird-` does not match `netbird_`, so no change is needed there.

**Acceptance.** `T-DNS-01`, `T-WIN-08`.

### SBS-DNS-03 No DNS takeover by a secondary (MUST)

**Requirement.** A secondary MUST NOT write the catch-all `.` rule, the `.local` exemption or the
adapter `NameServer`. When its configuration asks for a catch-all, it MUST keep its match domains and
log once that the catch-all was dropped because the installation is secondary.

**Rationale.** Two catch-all rules tie and Windows picks one arbitrarily
([`04-coexistence.md`](04-coexistence.md#62-catch-all)).

**Acceptance.** Configure a catch-all nameserver group on ACME's management: Windows still resolves
public names through GLOBEX's or the local resolver, and ACME's match domains through ACME
(`T-DNS-02`).

### SBS-DNS-04 Duplicate match domains (SHOULD)

**Requirement.** Before applying, a secondary SHOULD read the other installations' NRPT keys (names
starting `NetBird-` or `NetBird_` but not its own prefix), skip namespaces already claimed there, and
report them as a warning in `netbird status`. It SHOULD re-check on every apply.

**Rationale.** The tie-break between identical NRPT namespaces is undocumented; a skipped namespace
with a warning is predictable.

**Acceptance.** Both networks claim `corp.example`: names under it resolve through the primary, and
`netbird --instance acme status` shows the warning (`T-DNS-03`).

### SBS-DNS-05 Original nameservers exclude NetBird adapters (SHOULD)

**Requirement.** The capture of fallback nameservers (`host_windows.go:153-215`) SHOULD skip every
adapter whose name starts with `wt`, not only the installation's own GUID.

**Rationale.** Otherwise B records A's overlay DNS address as a fallback upstream, and that entry can
outlive A's session.

**Acceptance.** With the default installation route-all, start `acme`: its debug log lists only
physical-adapter nameservers (`T-DNS-04`).

### SBS-DNS-06 Group-policy store unchanged (MUST)

**Requirement.** The group-policy store handling (`useGPOPolicyStore`, `removeEmptyGPOPolicyStore`)
MUST keep today's semantics per installation: a store is removed only when it has no subkey and no
value.

**Rationale.** With per-installation prefixes, one installation can no longer empty the store under
another; the remaining race is documented in
[`04-coexistence.md`](04-coexistence.md#66-group-policy-nrpt-store).

**Acceptance.** `T-DNS-05`.

---

## SBS-FW: firewall

### SBS-FW-01 Own rules (MUST)

**Requirement.** A named installation MUST use the Windows Firewall rules `Netbird_<name>` and
`Netbird_<name>-v6`, and create and delete only those.

**Rationale.** Today the second installation reuses the first one's rule and is never allowed inbound
on its own IP; the first to stop deletes it for both
(`client/firewall/uspfilter/interface_allower_windows.go:36-86`).

**Acceptance.** `netsh advfirewall firewall show rule name=Netbird_acme` shows ACME's overlay IP;
stopping either installation leaves the other's rule (`T-FW-01`).

### SBS-FW-02 No stale rule (MUST)

**Requirement.** When a rule with its name already exists, a named installation MUST check that its
`localip` is the current overlay address and replace it otherwise.

**Rationale.** The rule has no crash state; after a crash and an address change a reused rule
allows the old address and blocks the new one.

**Acceptance.** Change ACME's overlay IP on management after a crash: the rule follows (`T-FW-02`).

---

## SBS-SSH: SSH

### SBS-SSH-01 Own client config file (MUST)

**Requirement.** A named installation MUST write and delete only
`%ProgramData%\ssh\ssh_config.d\99-netbird_<name>.conf`.

**Rationale.** Today the file is shared, last writer wins, and either cleanup deletes it.

**Acceptance.** Both files exist while both are connected; stopping either keeps the other's file
(`T-SSH-01`).

### SBS-SSH-02 Commands carry the installation (MUST)

**Requirement.** In a named installation's file, `ProxyCommand` and `Match ... exec` MUST call the
installation's own `netbird.exe` with `--instance <name>`.

**Rationale.** `ssh proxy` talks to the daemon through the default address otherwise
(`client/cmd/ssh.go:834`).

**Acceptance.** `ssh <acme-peer-fqdn>` from a plain OpenSSH client reaches the peer through ACME's
daemon (`T-SSH-02`).

### SBS-SSH-03 NetBird SSH client (MUST)

**Requirement.** `netbird --instance acme ssh <peer>` MUST connect through ACME's daemon.

**Rationale.** Follows from `SBS-CLI-01`, listed so it is tested.

**Acceptance.** `T-SSH-03`.

---

## SBS-COEX: coexistence and roles

### SBS-COEX-01 Roles (MUST)

**Requirement.** Every installation MUST have a role, `primary` or `secondary`, given on the service
command line as `--instance-role`. The default installation MUST be primary and ignore the flag. A
named installation MUST be secondary unless installed as primary. An unknown value MUST be treated
as secondary and logged.

**Rationale.** Installations do not coordinate; a static role is the only deterministic conflict rule.
Secondary is the more restrictive role, so an unparseable value falls back to it.

**Acceptance.** `T-COEX-01`.

### SBS-COEX-02 Two primaries (MUST)

**Requirement.** Two primaries MUST be allowed. A primary named installation MUST show a warning in
`netbird status` when another primary exists (the default service exists, or another record has
`Role=primary`).

**Rationale.** A user may want two full installations; the conflicts must at least be visible.

**Acceptance.** `T-COEX-02`.

### SBS-COEX-03 Identical routes (MUST)

**Requirement.** For an identical prefix routed by a primary and a secondary, traffic MUST go to the
primary (`SBS-NET-04`). A secondary SHOULD warn when one of its routes is identical to a route on
another NetBird adapter.

**Rationale.** Today both use route metric 1 on adapters with automatic metric: a tie.

**Acceptance.** `T-COEX-03`.

### SBS-COEX-04 Overlapping overlays (MUST)

**Requirement.** A secondary MUST NOT bring its tunnel up when its overlay prefix overlaps an address
prefix on another NetBird adapter, and MUST report the conflicting adapter and prefix in
`netbird status` and the UI. It SHOULD detect an overlap that appears later.

**Rationale.** Overlapping overlays break one of the two networks completely, and two default
NetBird networks collide with a probability of about 1 in 64.

**Acceptance.** `T-COEX-04`.

### SBS-COEX-05 Partial overlaps (SHOULD)

**Requirement.** A secondary SHOULD warn when a route on another NetBird adapter is more specific than,
and inside, its overlay prefix.

**Rationale.** Longest-prefix routing then sends part of its overlay to the other network; this may
be intended, so it is a warning, not a refusal.

**Acceptance.** `T-COEX-05`.

### SBS-COEX-06 No NetBird adapter as uplink (MUST)

**Requirement.** An installation MUST NOT select any NetBird adapter (name starting with `wt`) as the
uplink for its own sockets, in `GetBestInterface`
(`client/internal/routemanager/systemops/systemops_windows.go:885-928`) and in the legacy-routing
path (`client/internal/routemanager/systemops/systemops_generic.go:141-147`). The network monitor's initial next-hop lookup
(`client/internal/networkmonitor/monitor.go:57-58`) MUST skip NetBird adapters in the same way.

**Rationale.** Otherwise one installation's exit node pins the other's management, signal, relay and
WireGuard traffic into its tunnel, and nothing re-pins it when that tunnel goes away
([`04-coexistence.md`](04-coexistence.md#5-control-traffic-and-uplink-selection)).

**Acceptance.** In the lab, default installation on an exit node that carries all internet traffic:
ACME's management connection still
leaves through the physical adapter, ACME stays connected when the default installation disconnects,
and a DHCP renewal on the physical adapter does not restart ACME (`T-COEX-06`).

### SBS-COEX-07 No default route in a secondary (MUST)

**Requirement.** A secondary MUST NOT install a default route: no `0.0.0.0/1` + `128.0.0.0/1`, no
`::/1` + `8000::/1`, no IPv6 blackhole. Exit nodes auto-applied by management MUST be ignored, and a
user selection of an exit node MUST be refused with `exit nodes are disabled in a secondary
installation`.

**Rationale.** Two default routes tie; an exit node in a secondary would capture the primary's
traffic and, in an unmodified primary, its control plane.

**Acceptance.** `T-COEX-07`.

### SBS-COEX-08 Start order (MUST)

**Requirement.** The state reached after a reboot MUST NOT depend on which service starts first,
except for the group-policy store race documented in `04-coexistence.md`, section 6.6.

**Rationale.** Services start in parallel.

**Acceptance.** `T-WIN-06`, repeated with forced start orders.

### SBS-COEX-09 Independent lifecycle (MUST)

**Requirement.** Stopping, restarting, killing, upgrading or uninstalling one installation MUST leave
the other's adapter, routes, NRPT keys, firewall rules, SSH config file, autostart entry, UI process
and connectivity unchanged.

**Rationale.** The core promise of the feature.

**Acceptance.** `T-WIN-03`, `T-WIN-04`, `T-INS-02`.

---

## SBS-PROF: profiles

### SBS-PROF-01 Profiles per installation (MUST)

**Requirement.** Profiles (`netbird profile add|list|select|rename|remove`) MUST work in a named
installation as today, scoped to that installation's state and per-user directories.

**Rationale.** Owner's requirement; a named installation is a complete client.

**Acceptance.** `T-PROF-01`.

### SBS-PROF-02 No profile transfer (MUST NOT)

**Requirement.** The first delivery MUST NOT offer moving or copying a profile between
installations. The documentation MUST describe the supported procedure: log in fresh in the target
installation, then remove the profile from the source installation.

**Rationale.** A copied profile shares the WireGuard key, so management sees one peer with a
flapping endpoint, and `profile remove` in either installation deregisters it
(`client/server/server.go:1738-1801,2548-2580`). See Q9.

**Acceptance.** Documentation review.

---

## SBS-UI: tray UI

### SBS-UI-01 One UI per installation (MUST)

**Requirement.** `netbird-ui --instance acme` MUST run next to the default UI. Its single-instance
lock MUST be `io.netbird.ui.acme`; a second start of the same installation's UI MUST behave as
today (hand over to the running one).

**Rationale.** The fixed `UniqueID: "io.netbird.ui"` (`client/ui/main.go:300-305`) made the second UI
exit in the proof of concept.

**Acceptance.** Both tray icons visible; starting either again focuses the right one (`T-UI-01`).

### SBS-UI-02 Daemon address (MUST)

**Requirement.** The UI MUST derive its daemon address from `--instance`; an explicit `--daemon-addr`
MUST win.

**Rationale.** `SBS-CLI-02` for the UI.

**Acceptance.** `T-UI-01`.

### SBS-UI-03 Names in the UI (MUST)

**Requirement.** For a named installation, the tray tooltip, window title and tray menu header MUST
read `NetBird (ACME)`. The default installation's texts MUST stay unchanged.

**Rationale.** The colour alone is not enough for users who do not see colours well.

**Acceptance.** `T-UI-02`.

### SBS-UI-04 Autostart per installation (MUST)

**Requirement.** A named UI's autostart entry MUST be a separate Run value (`netbird_<name>`) whose
command is its own executable followed by `--instance <name>` (`AutostartOptions.Identifier` and
`Arguments` of the Wails fork). The first-run autostart decision MUST look at the installation's own
preferences file and state directory, not the default installation's.

**Rationale.** Today one Run value is shared, and the first-run gate of a second installation sees
the first one's `autostartInitialized` flag and footprint and never enables autostart
(`client/ui/autostart_default.go:55-93`). The Wails fork finds "its" Run value by executable path and
deletes other values pointing at the same executable (fork `autostart_windows.go:58-60,105-136`), so
isolation also depends on each installation having its own executable path, which model A gives.

**Acceptance.** Sign out and in: both UIs start; disabling one leaves the other (`T-UI-03`).

### SBS-UI-05 Notifications per installation (MUST)

**Requirement.** A named UI MUST set Wails `Options.Name` to `NetBird_<name>`, which gives it its own
AUMID, notification-categories key and toast icon file, and MUST write its activator CLSID
`UUID(toast-activator)` into `HKCU\Software\Classes\AppUserModelId\NetBird_<name>` before Wails reads
it. Notifications MUST show `NetBird (ACME)` as the source, and a toast action MUST be handled by the
UI of the installation that sent it.

**Rationale.** Every UI start rewrites the shared CLSID's `LocalServer32` to its own path (fork
`notifications_windows.go:93-103`), so with a shared identity the last UI started receives every
activation, and a session-warning "Extend now" would extend the wrong daemon's session. NSIS
shortcuts carry no AUMID today and do not need one.

**Acceptance.** `T-UI-04`.

### SBS-UI-06 Signals per installation (MUST)

**Requirement.** The quick-actions event MUST be per installation
(`Global\NetBird_<name>QuickActionsTriggerEvent`).

**Rationale.** One shared event triggers every UI (`client/ui/signal_windows.go:15`).

**Acceptance.** `T-UI-05`.

### SBS-UI-07 Per-user data (MUST)

**Requirement.** A named UI MUST keep preferences and logs in `%APPDATA%\netbird_<name>`, and MUST set
`Options.Windows.WebviewUserDataPath` to `%APPDATA%\netbird_<name>\webview2`. The default UI keeps
the library default `%APPDATA%\netbird-ui.exe`.

**Rationale.** Two WebView2 instances cannot share one user data folder with different settings, and
shared preferences mix the two installations' choices.

**Acceptance.** `T-UI-06`.

### SBS-UI-08 Never touch another UI (MUST NOT)

**Requirement.** A UI MUST NOT end, signal or reconfigure another installation's UI.

**Acceptance.** `T-UI-05`.

### SBS-UI-09 Installation details (SHOULD)

**Requirement.** The settings window SHOULD show the installation name, label, colour and role,
read-only.

**Rationale.** Support and screenshots need it.

**Acceptance.** Visual check.

### SBS-UI-10 Elevation and one-shot paths (MUST)

**Requirement.** The UI's elevated one-shot (`--apply-privileged-settings`) and the fallback command it
shows when elevation fails MUST target the installation's own daemon.

**Rationale.** The one-shot passes `--daemon-addr` on, but the fallback command omits it
(`client/ui/services/guarded.go:133-139,219-226`).

**Acceptance.** `T-CLI-04`.

---

## SBS-COL: colour

### SBS-COL-01 Colour attribute (MUST)

**Requirement.** Every named installation MUST have a colour, stored as `#RRGGBB` in its record. The
installer offers the palette of [`03-installer-ux.md`](03-installer-ux.md#44-colour-palette) and
accepts any `#RRGGBB`. The default installation MUST have no colour.

**Rationale.** Owner's requirement. A colour-blind-safe palette is the default choice.

**Acceptance.** `T-COL-01`.

### SBS-COL-02 Overlay dot (MUST)

**Requirement.** The UI MUST draw a filled circle in the installation colour, about 40 % of the icon
width, with a contrasting ring (dark on a light taskbar, light on a dark one) about 6 % of the icon
width, in the **top-left** corner, on every tray state and theme variant, at runtime, on top of the
existing PNG bytes at their native 256x256 size. The default installation's icons MUST stay
byte-identical.

**Rationale.** The state icons already use colour; tinting would make states unreadable. The status
badge sits bottom-right and the update arrow top-right in every state, so top-left is the free
corner. Windows scales the icon down to 16 px at 100 % (fork `w32/icon.go:217-244`), so the dot must
be large and the ring is what keeps it visible on any taskbar. Compositing at runtime keeps one set
of assets.

**Acceptance.** Golden images per state and theme; pixel test that outside the dot area the image is
unchanged (`T-COL-02`).

### SBS-COL-03 No new dependency (MUST)

**Requirement.** Compositing MUST use only the Go standard library (`image`, `image/draw`,
`image/png`, `image/color`).

**Rationale.** `AGENTS.md` requires the owner's decision for any new dependency; none is needed.

**Acceptance.** `go.mod` unchanged.

### SBS-COL-04 Colour elsewhere (SHOULD)

**Requirement.** The UI SHOULD show the colour as a stripe or badge in the window header and use the
dotted icon in its notifications.

**Acceptance.** Visual check.

### SBS-COL-05 Changing the colour (SHOULD)

**Requirement.** Re-running the installer for the same name with another colour, or with
`/COLOR=`, SHOULD update the record; the UI SHOULD pick it up at its next start.

**Rationale.** Colour is cosmetic; a reinstall path is enough for the first delivery. See Q5.

**Acceptance.** `T-COL-03`.

---

## SBS-INS: installer

### SBS-INS-01 Advanced mode (MUST)

**Requirement.** The NSIS installer MUST offer, on its Welcome page, a checkbox that leads to the
named-installation page described in [`03-installer-ux.md`](03-installer-ux.md#43-named-installation-page).
With the checkbox off, the pages and their effect MUST be today's.

**Acceptance.** `T-INS-01`.

### SBS-INS-02 Unattended parameters (MUST)

**Requirement.** The NSIS installer MUST accept `/INSTANCE=`, `/COLOR=`, `/ROLE=` and `/AUTOSTART=` as
specified in [`03-installer-ux.md`](03-installer-ux.md#51-nsis-parameters).

**Acceptance.** `T-INS-01`.

### SBS-INS-03 Invalid parameters (MUST)

**Requirement.** An invalid parameter MUST abort with exit code 2 and a message in the install log,
and MUST NOT fall back to installing or upgrading the default installation.

**Rationale.** A typo in a deployment script must not overwrite the default installation.

**Acceptance.** `T-INS-03`.

### SBS-INS-04 Never act on another installation (MUST NOT)

**Requirement.** Installing, upgrading, repairing or uninstalling one installation MUST NOT run
another installation's uninstaller, stop its service, end its processes or change its registration.

**Rationale.** Today installing a second copy removes the first (`client/installer.nsis:170-193`).

**Acceptance.** `T-INS-02`.

### SBS-INS-05 Uninstaller knows its installation (MUST)

**Requirement.** The uninstaller MUST identify its installation from the record whose `InstallDir`
is its own folder, and MUST stop with an error when no record matches and its folder is not the
default one.

**Rationale.** A stray uninstaller must never act on the default installation (fail closed).

**Acceptance.** `T-INS-05`.

### SBS-INS-06 Upgrade keeps identity (MUST)

**Requirement.** Upgrading a named installation MUST keep its state folder, record and per-user
files, so it reconnects as the same peer without a new login.

**Acceptance.** `T-INS-06`.

### SBS-INS-07 End UI processes by path (MUST)

**Requirement.** Fork-built installers, uninstallers and updaters MUST end only UI processes whose
executable is inside the target installation's folder.

**Rationale.** `taskkill /im netbird-ui.exe` ends every installation's UI.

**Acceptance.** `T-INS-02`.

### SBS-INS-08 Distinct UI file name (SHOULD)

**Requirement.** A named installation SHOULD install its UI as `netbird-ui_<name>.exe`.

**Rationale.** Keeps named UIs out of an unmodified upstream installer's or updater's
`taskkill /im netbird-ui.exe`. See Q6.

**Acceptance.** `T-WIN-08`.

### SBS-INS-09 Registration (MUST)

**Requirement.** A named installation MUST register as in
[`03-installer-ux.md`](03-installer-ux.md#7-registration-per-installation): its own folder, Uninstall
key, Start-menu and desktop shortcuts that pass `--instance`; no `App Paths` key; no `PATH`
change.

**Acceptance.** `T-INS-04`.

### SBS-INS-10 Complete, contained uninstall (MUST)

**Requirement.** Uninstalling a named installation MUST remove its service, adapter, NRPT keys,
firewall rules, SSH config file, Run value and AUMID key of the uninstalling user, shortcuts,
Uninstall key, record, program folder, and (when asked) its state and per-user folders, and nothing
of any other installation.

**Acceptance.** `T-WIN-04`.

### SBS-INS-11 MSI (MUST)

**Requirement.** The MSI MUST keep building the default installation exactly as today. Named MSI
installations are deferred to the design in [`03-installer-ux.md`](03-installer-ux.md#9-msi-named-installations-deferred-design).

**Acceptance.** MSI build output unchanged except for version.

---

## SBS-UPD: updates, build, distribution

### SBS-UPD-01 No foreign installer for a named installation (MUST)

**Requirement.** A named installation MUST NOT launch an installer that does not support named
installations, and its UI MUST NOT offer a download link to one. Until an instance-aware release
channel exists, a named installation built from the fork MUST refuse `TriggerUpdate` and ignore
forced updates; it MAY show that a newer version exists.

**Rationale.** The upstream installer knows only the default installation; run by a named
installation's updater it would upgrade or replace the default one. The UI picks its download link
from `App Paths\Netbird` (`version/url_windows.go:13-28`), so a named UI would link to the installer
that replaces the default installation.

**Acceptance.** `T-UPD-01`.

### SBS-UPD-02 Updater acts on its own installation (MUST)

**Requirement.** The updater MUST end only its own installation's UI processes and restart only its
own service.

**Rationale.** Today it ends every `netbird-ui.exe` and starts the default service
(`client/internal/updater/installer/installer_run_windows.go:129-216`). Without named installations
this is the same behaviour.

**Acceptance.** `T-UPD-02`.

### SBS-UPD-03 Own temp folder (MUST)

**Requirement.** The updater's temp folder MUST be `<state directory>\tmp-install`, and the cleanup at
connect (`client/internal/connect.go:313-316`) MUST touch only that folder.

**Rationale.** Today every daemon deletes the shared folder on connect, including another
installation's update in progress.

**Acceptance.** `T-UPD-03`.

### SBS-UPD-04 Instance-aware updater (MAY, deferred)

**Requirement.** A later delivery MAY make the updater pass `/INSTANCE=<name>` to an NSIS installer
that supports it and restart `Netbird_<name>`.

### SBS-UPD-05 Fork build (MUST)

**Requirement.** The fork MUST have a GitHub Actions workflow that builds an unsigned NSIS installer
from a tag, using the steps in [`03-installer-ux.md`](03-installer-ux.md#10-building-the-forks-installer),
with version `<upstream version>+sbs.<n>`.

**Rationale.** The fork has no access to upstream's signing pipeline.

**Acceptance.** A tag in the fork produces `netbird-installer.exe` as a workflow artifact
(`T-UPD-04`).

### SBS-UPD-06 Rebase cost (SHOULD)

**Requirement.** The change SHOULD stay within the files listed in
[`00-overview.md`](00-overview.md#8-delivery-and-upstreamability) steps 1 to 5 plus their tests, so a
rebase onto a new upstream release touches a known, small set of files.

---

## SBS-POL: group policy and MDM

The Windows policy key is `HKLM\Software\Policies\NetBird`, read value by value and reloaded every
minute (`client/mdm/policy_windows.go:18,81,95`, `client/mdm/ticker.go:16`). Its values split into
identity values that belong to one network and posture values that an administrator sets for the
machine ([`02-identifier-matrix.md`](02-identifier-matrix.md#12-policy-gpo--mdm)).

### SBS-POL-01 Posture policy applies to every installation (MUST)

**Requirement.** Posture values in the global key (`allowServerSSH`, `allowRemoteJobs`,
`disableClientRoutes`, `disableServerRoutes`, `blockInbound`, `disableAutoConnect`, `rosenpass*`,
`lazyConnection`, `debugBundleUploadURL`, `disableProfiles`, `disableNetworks`,
`disableUpdateSettings`, `disableAdvancedView`, `disableAutostart`, `disableMetricsCollection`) MUST
apply to named installations as they apply to the default one, including the locking of managed
fields.

**Rationale.** An administrator's machine policy must not be bypassed by adding an installation
(`AGENTS.md`, "Never fail open").

**Acceptance.** `T-POL-01`.

### SBS-POL-02 Identity policy applies to the default installation only (MUST)

**Requirement.** `managementURL`, `preSharedKey`, `wireguardPort`, `enableLocalMetrics` and
`localMetricsAddress` from the global key MUST apply only to the default installation.

**Rationale.** Applied to every installation, `managementURL` would point ACME at GLOBEX's server,
`preSharedKey` would block every ACME peer, and the port and address would collide.

**Acceptance.** `T-POL-01`.

### SBS-POL-03 Per-installation policy (SHOULD)

**Requirement.** A named installation SHOULD read `HKLM\Software\Policies\NetBird\Installations\<name>`
with the same value names and types. A value there overrides the global value for that installation.

**Rationale.** Lets an administrator manage a named installation's network settings. The v0.81.5
loader reads values, not subkeys, so this key is invisible to it. ADMX cannot express a key path
that contains the name; administrators use registry preferences, Intune custom settings or `.reg`.

**Acceptance.** `T-POL-02`.

### SBS-POL-04 Forbidding named installations (SHOULD)

**Requirement.** A global boolean `disableNamedInstallations` SHOULD make named daemons refuse to
connect, with a clear status message, and the installer refuse to create a named installation.

**Rationale.** An organisation that manages NetBird by policy needs a way to say no; without it, the
feature is an unmanaged second client on managed machines. A new policy value is public surface
(ADMX), hence Q13.

**Acceptance.** `T-POL-03`.

---

## SBS-PLAT: other platforms

### SBS-PLAT-01 Linux, macOS, FreeBSD (MUST)

**Requirement.** On these platforms the identity package MUST return today's values for the default
installation, and named installations MUST be rejected (`SBS-CLI-06`). Packaging MUST NOT change.

**Rationale.** No regression; named installations there are deferred.

**Acceptance.** `T-PLAT-01`, `T-DEF-01` on every platform in CI.

### SBS-PLAT-02 Mobile, WASM, embedded (MUST)

**Requirement.** Android, iOS, WASM and `client/embed` MUST NOT be affected.

**Acceptance.** Their builds and tests pass unchanged.

---

## SBS-SEC: security

### SBS-SEC-01 No weaker IPC (MUST)

**Requirement.** Named installations MUST keep every IPC protection of the default one
(`SBS-SVC-03`, `SBS-SVC-04`). The first delivery MUST NOT add a transport, a gRPC method or a proto
field.

**Rationale.** `AGENTS.md`: weakening a security control needs the owner's decision; proto changes
are high-risk. None is needed: the name and colour reach the UI without the daemon, the CLI knows the
name from its own flag and the role from the record, and every warning in this specification goes
through the existing system-event stream (`SystemEvent`, `client/proto/daemon.proto:683`;
`PublishEvent`, `client/internal/peer/status.go:1289`), which the UI shows and `netbird status`
lists (`client/status/status.go:160,642-648`).

**Acceptance.** `T-SEC-01`; `client/proto` unchanged.

### SBS-SEC-02 Name validation before use (MUST)

**Requirement.** The name MUST be validated (`SBS-ID-02`) before it is used in any path, registry
key, pipe name or command line, in every process.

**Rationale.** The charset rules out path traversal, quoting problems and pipe-namespace tricks.

**Acceptance.** `T-ID-02`.

### SBS-SEC-03 The record is untrusted input for the UI (MUST)

**Requirement.** The UI MUST validate the colour and label it reads from the record and fall back to
no colour and the canonical name when they are invalid.

**Acceptance.** `T-COL-01`.

### SBS-SEC-04 Fail closed (MUST)

**Requirement.** Every check introduced by this feature that protects another installation (role
parsing, uninstaller identification, pipe ownership) MUST deny or choose the more restrictive option
on error.

**Rationale.** `AGENTS.md`, "Never fail open".

**Acceptance.** `T-COEX-01`, `T-INS-05`, `T-SVC-03`.
