# Test plan

Tests that prove the requirements in [`01-requirements.md`](01-requirements.md). Automated tests run
in the existing CI (`golang-test-{linux,darwin,windows,freebsd}.yml`) unless marked otherwise.
Manual tests run on a disposable Windows VM, never on a machine whose connectivity matters
(`AGENTS.md`, "Run locally").

## 1. Environments

| ID | Environment | Used for |
|---|---|---|
| E1 | CI unit tests, every OS, `make test-unit` (host-safe, no admin) | Identity package, validation, derivation, matchers, config generation, golden images |
| E2 | Windows 11 VM (Hyper-V or VirtualBox), local administrator, snapshot "clean" and snapshot "official v0.81.5 installed" | Every `T-WIN-*`, `T-INS-*`, and the manual parts of other tests |
| E3 | Two management servers: either the owner's AIRSECOMA and LFBB, or two lab instances from `infrastructure_files/` | Connected scenarios |
| E4 | Lab management instances whose account settings can be changed through the REST API: `network_range` and `dns_domain` (`shared/management/http/api/openapi.yml:341-352`) | Overlap and duplicate-domain tests (`T-COEX-04`, `T-DNS-03`) |

Lab setup for E4: two self-hosted instances, `lab-a` (primary) and `lab-b` (secondary), each with
one routing peer in a separate VM, so routed networks and exit nodes can be tested without the
production networks.

## 2. Verification snippets

Used by the manual tests. Run in an elevated PowerShell unless noted.

```powershell
# Services
Get-Service Netbird, Netbird_* | Format-Table Name, DisplayName, Status
sc.exe qc Netbird_lfbb

# Adapters, metrics, routes
Get-NetAdapter -IncludeHidden | Where-Object Name -like 'wt*' | Format-Table Name, InterfaceGuid, Status, ifIndex
Get-NetIPInterface | Where-Object InterfaceAlias -like 'wt*' | Format-Table InterfaceAlias, AddressFamily, InterfaceMetric, AutomaticMetric
Get-NetRoute | Where-Object InterfaceAlias -like 'wt*' | Format-Table DestinationPrefix, InterfaceAlias, RouteMetric

# NRPT (effective and raw keys)
Get-DnsClientNrptPolicy -Effective | Format-Table Namespace, NameServers
Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Services\Dnscache\Parameters\DnsPolicyConfig' | Select-Object PSChildName
Get-ChildItem 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient\DnsPolicyConfig' -ErrorAction SilentlyContinue | Select-Object PSChildName

# Firewall
Get-NetFirewallRule -DisplayName 'Netbird*' | Format-Table DisplayName, Enabled, Direction
netsh advfirewall firewall show rule name=Netbird_lfbb

# SSH client config
Get-ChildItem "$env:ProgramData\ssh\ssh_config.d"

# Registration
Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall' | Where-Object PSChildName -like 'Netbird*'
Get-ItemProperty 'HKLM:\SOFTWARE\NetBird\Installations\*'
Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'          # as the signed-in user
Get-ChildItem 'HKCU:\Software\Classes\AppUserModelId' | Where-Object PSChildName -like 'NetBird*'

# Processes and pipes
Get-Process netbird* | Format-Table Id, ProcessName, Path, SessionId
[System.IO.Directory]::GetFiles('\\.\pipe\') | Where-Object { $_ -match 'netbird' }
```

**Snapshot and diff.** For `T-WIN-01`, `T-WIN-03`, `T-WIN-04` and `T-WIN-08`, the output of all
snippets above, plus a recursive file listing of `%ProgramData%\Netbird*`, `%APPDATA%\netbird*` and
`C:\Program Files\Netbird*`, is saved to a text file before and after the action and compared with
`Compare-Object`. The expected difference is stated in each test.

## 3. Automated tests

| ID | What | Requirement | Env |
|---|---|---|---|
| `T-ID-01` | `instance.For("")` and `instance.For("lfbb")` return the values of the matrix; one binary, two command lines | `SBS-ID-01` | E1 |
| `T-ID-02` | Name validation: table of valid and invalid inputs with exact messages (empty, leading digit, `-`, `_`, space, 13 characters, non-ASCII, `default`, `netbird`, mixed case); Go fuzz target asserting every accepted name produces identifiers valid for their kind (path, registry key, pipe, service name length) | `SBS-ID-02`, `-03`, `SBS-SEC-02` | E1 |
| `T-ID-03` | Disjointness property test: for generated names (including `a`, `ab`, `a1`, `v6`, 12-character names) every named identifier differs from every default identifier and every other name's; the v0.81.5 matchers (`strings.HasPrefix(lower(x), "netbird-")`, exact firewall names, `wt` ICE prefix) behave as in matrix section 13; the named NRPT matcher of `a` matches none of `ab`'s keys | `SBS-ID-05` | E1 |
| `T-ID-04` | Source scan: a test walks the repository's Go files outside the identity package and fails on a list of forbidden literals (derived identifiers) | `SBS-ID-04`, `SBS-FS-03` | E1 |
| `T-ID-05` | UUIDv5 vectors: `lfbb`, `airsecoma`, `work` give the values of matrix section 1; none equals the default GUID | `SBS-ID-06` | E1 |
| `T-ID-06` | Name precedence: `service run` ignores `NB_INSTANCE`; other commands use flag, then env, then default; `--instance default` | `SBS-ID-09` | E1 |
| `T-DEF-01` | Golden table of every default identifier, per `GOOS`, compared with v0.81.5 literals copied into the test | `SBS-DEF-01`, `SBS-PLAT-01` | E1, all OSes |
| `T-NET-03` | Port derivation: range 51821-51920, never 51820, `lfbb` gives 51916; same `k` for metrics, pprof, SOCKS5 | `SBS-NET-03`, `-05` | E1 |
| `T-DNS-U1` | NRPT key naming and the prefix filter of `listNRPTRuleKeys`, run on in-memory key lists that mix default, `lfbb`, `lfbb2` and foreign keys | `SBS-DNS-01`, `-02` | E1 (Windows) |
| `T-DNS-U2` | Host DNS config of a secondary: route-all input produces no `.` rule, no exemption, no `NameServer` | `SBS-DNS-03` | E1 (Windows) |
| `T-SSH-U1` | Generated SSH config of a named installation: file name, `--instance` in `ProxyCommand` and `Match exec`, own executable path | `SBS-SSH-01`, `-02` | E1 |
| `T-COEX-U1` | Role parsing: `primary`, `secondary`, empty, garbage (falls back to secondary, logged); default installation ignores the flag | `SBS-COEX-01`, `SBS-SEC-04` | E1 |
| `T-COEX-U2` | Uplink selection on a synthetic interface and route table: `wt0`, `wt_lfbb` and a physical NIC; a `/1` on `wt0` does not make it the uplink for `wt_lfbb`'s sockets | `SBS-COEX-06` | E1 (Windows) |
| `T-COEX-U3` | Route manager of a secondary: an auto-applied exit node is not selected, a user selection is refused with the specified message, no `/1` or blackhole routes are planned | `SBS-COEX-07` | E1 |
| `T-COL-02` | Overlay rendering: for every tray state and theme asset, the composited PNG equals a golden image; the dot is in the top-left corner; pixels outside the dot's bounding box equal the source; the default installation gets the source bytes unchanged | `SBS-COL-02` | E1 |
| `T-UPD-U1` | Updater of a named installation: `TriggerUpdate` refused, forced update ignored; temp folder is under the named state directory; cleanup touches only it | `SBS-UPD-01`, `-03` | E1 |
| `T-POL-U1` | Policy split with the loader's injected fetcher (`client/mdm/policy_windows.go:74-79`): posture values reach a named installation, `managementURL`, `preSharedKey`, `wireguardPort` and the metrics values do not; per-installation values override; the default installation's result is unchanged | `SBS-POL-01` to `-03` | E1 |
| `T-CLI-U1` | Suggested commands in errors and hints: with a named installation every one contains `--instance <name>`; with the default installation the text is byte-identical to v0.81.5 | `SBS-CLI-07`, `SBS-UI-10` | E1 |
| `T-PLAT-01` | On Linux, macOS and FreeBSD, `--instance lfbb` fails with the specified message and exit code; default behaviour unchanged | `SBS-CLI-06`, `SBS-PLAT-01` | E1 |
| `T-UPD-04` | Fork workflow: a tag produces an unsigned `netbird-installer.exe` artifact with version `<upstream>+sbs.<n>` | `SBS-UPD-05` | Fork CI |

Every test above runs with `-race` where it touches shared state, as `AGENTS.md` requires.

## 4. Manual acceptance scenarios

Each scenario starts from a VM snapshot and lists its expected results. "A" is the default
installation connected to AIRSECOMA (or `lab-a`), "B" is the named installation `lfbb`, colour
orange, secondary, connected to LFBB (or `lab-b`).

### T-WIN-01 Upgrade of an existing default installation (seed scenario 5)

1. Snapshot "official v0.81.5 installed", connected to AIRSECOMA. Take a snapshot listing.
2. Run the fork installer without the advanced checkbox (and again with `/S`).
3. Take a second listing and compare.

Expected: the only differences are binary versions and timestamps. Same service name, DisplayName
and command line, same adapter name and GUID, same pipe, same NRPT key names, same firewall rule
names, same paths, same Run value and AUMID. The peer reconnects with the same overlay IP and no
login prompt. (`SBS-DEF-01`, `-02`, `-03`)

### T-WIN-02 Fresh install of A and B, both connected (seed scenario 1)

1. Clean snapshot. Install A, log in to AIRSECOMA.
2. Run the installer again with the advanced checkbox: name `LFBB`, colour orange, secondary,
   autostart on. Log in to LFBB from B's tray icon.
3. Run the snippets.

Expected:
- Services `Netbird` and `Netbird_lfbb` running; DisplayNames `Netbird` and `NetBird (LFBB)`.
- Adapters `wt0` `{f2f29e61-...}` and `wt_lfbb` `{a069b2c3-...}`; `wt_lfbb` metric 1000.
- Two tray icons; B's has an orange dot in every state; tooltips `NetBird` and `NetBird (LFBB)`.
- `Resolve-DnsName` for a name under each network's domain answers with that network's address.
- `netbird status` and `netbird --instance lfbb status` show each network's peers.
- NRPT keys `NetBird-Match-0` and `NetBird_lfbb-Match-0`; firewall rules `Netbird` and `Netbird_lfbb`
  with their own overlay IPs; SSH files `99-netbird.conf` and `99-netbird_lfbb.conf`.
- No file under `%ProgramData%\Netbird` or `%APPDATA%\netbird` changed when B was installed and
  connected. (`SBS-FS-01`, `SBS-NET-01`, `SBS-UI-*`, `SBS-COL-*`)

### T-WIN-03 Stop, restart and kill one installation (seed scenario 2)

For each of A and B in turn: `netbird down`; service stop; service restart; `Stop-Process -Force`
on the daemon, then service start.

Expected after each step: the other installation's adapter, routes, NRPT keys, firewall rules, SSH
file, Run value, UI process and ping to one of its peers are unchanged (snapshot diff limited to the
acted-on installation). The acted-on installation recovers fully after restart; after the kill, its
own leftovers are cleaned by its replay of its own `state.json`. (`SBS-COEX-09`, `SBS-FS-05`,
`SBS-NET-06`)

### T-WIN-04 Uninstall one installation (seed scenario 4)

1. Uninstall B from "Apps", with "delete data" ticked.
2. Snapshot diff.
3. Restore, repeat by uninstalling A instead.

Expected: everything of the removed installation is gone (service, Event Log source, adapter, NRPT
keys, firewall rules, SSH file, Run value and AUMID key of the current user, shortcuts, Uninstall
key, record, program folder, state folder, per-user folder); nothing of the other installation
changed and it stays connected. (`SBS-INS-10`, `SBS-SVC-06`)

### T-WIN-05 Stable adapter identity

Uninstall B without deleting data, reinstall B with the same name. Expected: same adapter GUID and
name; Windows shows the same network profile, no "wt_lfbb 2". (`SBS-ID-06`, `SBS-NET-01`)

### T-WIN-06 Reboot and start order (seed scenario 2)

Reboot five times; then force each order with `sc.exe config Netbird_lfbb depend= Netbird` and the
reverse, rebooting each time (remove the dependency afterwards).

Expected: both installations connected after every boot; A keeps port 51820, B uses 51916; NRPT,
firewall and SSH state as in `T-WIN-02`. (`SBS-COEX-08`, `SBS-NET-03`)

### T-WIN-07 Upgrade one installation (seed scenario 3)

1. Upgrade B interactively (advanced checkbox, pick `LFBB` in the list), then again silently with
   `/S /INSTANCE=lfbb`.
2. Upgrade A interactively and silently.

Expected: only the targeted installation's service restarts and UI restarts; the other keeps its
UI process (same PID), service and connectivity. The upgraded one reconnects as the same peer.
(`SBS-INS-04`, `-06`, `-07`)

### T-WIN-08 Next to an official default installation

1. Snapshot "official v0.81.5 installed", connected to AIRSECOMA.
2. Install B from the fork (`/S /INSTANCE=lfbb`), connect it.
3. Run the official v0.81.5 installer again (upgrade path), then let its updater run if the
   management server offers an update.
4. Run `netbird --daemon-addr npipe://netbird_lfbb status` with the official CLI.

Expected: both connected after every step; the official installer and updater do not end
`netbird-ui_lfbb.exe` and do not touch B's service, rules or files; the official CLI reaches B.
Documented limitation observed: none of the above breaks. (`SBS-DEF-04`, `SBS-CLI-04`,
`SBS-INS-08`)

### T-INS-01 Installer pages and parameters

Interactive: the Welcome page shows the checkbox; with it off, the page sequence is today's.
With it on: the named page validates as specified, pre-selects the first unused colour, shows
"will be created" or "will be upgraded". Silent: every parameter of `03-installer-ux.md` section 5.1
works. (`SBS-INS-01`, `-02`)

### T-INS-02 Installing B does not affect A

With A connected and its UI open, install B interactively and silently. Expected: A's UI keeps its
PID, A's service is not restarted, A stays connected. (`SBS-INS-04`, `-07`)

### T-INS-03 Invalid input (seed scenario 8)

Silent: `/INSTANCE=lfb-booking`, `/INSTANCE=default`, `/INSTANCE=lfbbookingnetwork`,
`/COLOR=#12345`, `/ROLE=main`. Interactive: the same names in the page. CLI:
`netbird service install --instance lfb-booking`.

Expected: exit code 2 (installer) or non-zero (CLI) with the specified message; nothing installed or
changed; in particular the default installation is not touched. A duplicate name is treated as an
upgrade, not an error. (`SBS-INS-03`, `SBS-ID-02`, `-03`)

### T-INS-04 Registration and record

After `T-WIN-02`: the record has the given values; a standard user can read and cannot write it; no
`App Paths` key and no `PATH` entry for B; the shortcut targets `netbird-ui_lfbb.exe --instance lfbb`. Delete the record and restart B's service: B still connects; its UI falls back to no
colour and the canonical name. (`SBS-ID-07`, `SBS-INS-09`, `SBS-SEC-03`)

### T-INS-05 Uninstaller without a record

Copy B's uninstaller to `C:\Temp\x\`, delete B's record, run the copy. Expected: it stops with an
error and changes nothing, in particular not the default installation. (`SBS-INS-05`)

### T-INS-06 Upgrade keeps the peer

Upgrade B: no login prompt, same overlay IP, same peer in LFBB's dashboard. (`SBS-INS-06`)

### T-SVC-01 to T-SVC-04 Service registration

- `T-SVC-01`: `sc qc`, `sc qdescription` match `SBS-SVC-01`; the default service unchanged.
- `T-SVC-02`: every `service` subcommand with `--instance lfbb` acts on B only; `-s Netbird
  --instance lfbb` is an error. With something listening on `127.0.0.1:41731`, B neither probes nor
  migrates to it.
- `T-SVC-03`: with B running, start `netbird service run --instance lfbb` in a console: it exits
  with an error naming the pipe; clients still reach the service.
- `T-SVC-04`: `service install --instance lfbb --service-env X=1` a second time fails before writing:
  B's `Environment` value and `service.json` are unchanged.

### T-FS-02, T-FS-03 Files

- `T-FS-02`: set `NB_STATE_DIR=C:\x` machine-wide, restart both services: A uses `C:\x` (today's
  behaviour), B uses `%ProgramData%\Netbird_lfbb` and logs the warning.
- `T-FS-03`: `netbird --instance lfbb profile add x; ... profile select x`: only
  `%APPDATA%\netbird_lfbb\active_profile.txt` changes; `netbird profile list` for A is unchanged.

### T-CLI-01 to T-CLI-03 Command line

- `T-CLI-01`: `netbird --instance lfbb status` and `$env:NB_INSTANCE='lfbb'; netbird status` show B
  with name and role; `netbird status` shows A with today's output (text, `--json`, `--yaml`).
- `T-CLI-02`: `netbird instance list` as a standard user lists A and B with the specified columns.
- `T-CLI-03`: `netbird --instance lfbb --daemon-addr npipe://netbird status` talks to A; for A,
  `--daemon-addr`, `-s`, `--interface-name`, `--log-file` behave as in v0.81.5.
- `T-CLI-04`: as a standard user, run a privileged command against B (for example
  `netbird --instance lfbb up --enable-ssh-root`) and cancel the UAC prompt in B's UI settings: every
  suggested command in the error and in the UI fallback contains `--instance lfbb`. Running the
  suggested command affects B only.

### T-NET-02, T-NET-05 Interface and listeners

- `T-NET-02`: `netbird --instance lfbb up --interface-name nb1` fails; `--interface-name wt0` fails
  while A is up; `--interface-name wt_x` works.
- `T-NET-05`: enable local metrics on A and B; both `http://127.0.0.1:9191/metrics` and
  `http://127.0.0.1:9287/metrics` answer.
- `T-NET-06`: set `NB_USE_LEGACY_ROUTING=true` machine-wide and restart both services: A uses legacy
  routing as today (host routes on the physical adapter); B logs the warning, adds no host routes and
  stays connected when A stops.

### T-DNS-01 to T-DNS-05 DNS

- `T-DNS-01`: with both connected, reconnect A, then B, then restart each service: both
  installations' NRPT keys are present after every step and names resolve per network.
- `T-DNS-02`: configure a catch-all nameserver group (e.g. `1.1.1.1`, no domains) on LFBB: B writes
  no `.` rule and no `NameServer`; public names resolve as with A alone; B logs the drop once.
- `T-DNS-03`: set `dns_domain` of both lab accounts to `corp.example`, or route the same match
  domain: names resolve through A; `netbird --instance lfbb status` warns.
- `T-DNS-04`: with A route-all, start B: B's debug log of captured nameservers contains no overlay
  address.
- `T-DNS-05`: create the group-policy NRPT store with one foreign rule; start A and B: both write
  into it; stopping either removes only its own keys; the store survives while the foreign rule
  exists, and is removed only when empty. Then reproduce the documented race (empty store left by a
  killed A, start B and A at the same moment) and record the outcome.

### T-FW-01, T-FW-02 Firewall

- `T-FW-01`: rules `Netbird` and `Netbird_lfbb` exist with each overlay IP; an inbound connection
  from a B peer to B's overlay IP is allowed; stopping either installation removes only its rules.
- `T-FW-02`: kill B, change B's overlay IP on the management side, restart B: `Netbird_lfbb` carries
  the new IP.

### T-SSH-01 to T-SSH-03 SSH (seed scenario 7)

- `T-SSH-01`: both files exist while both are connected; stopping either keeps the other's file.
- `T-SSH-02`: from a plain Windows OpenSSH client, `ssh <user>@<lfbb-peer-fqdn>` connects through
  B's daemon (B's log shows the proxy); the same for an A peer through A. Also check that Windows
  OpenSSH reads `ssh_config.d` without an `Include` line (`04-coexistence.md`, section 8).
- `T-SSH-03`: `netbird --instance lfbb ssh <lfbb-peer>` works.

### T-COEX-01 to T-COEX-07 Coexistence (seed scenario 6)

- `T-COEX-01`: install B with `/ROLE=primary`, then repair with `/ROLE=secondary`; the service
  command line follows. Edit the service command line (`sc.exe config Netbird_lfbb binPath= ...`) to
  `--instance-role bogus` and restart: B behaves as secondary and logs the fallback.
- `T-COEX-02`: B primary next to A: `netbird --instance lfbb status` warns about A.
- `T-COEX-03`: route the same prefix (e.g. `192.168.77.0/24`) in both lab networks: traffic goes to
  A (`Find-NetRoute -RemoteIPAddress 192.168.77.1`); B warns. Disconnect A: traffic goes to B.
- `T-COEX-04`: set both lab accounts' `network_range` to `100.90.0.0/16`: B refuses to connect and
  status names `wt0` and the prefix; A works. Start B first, then A: A works, B reports the conflict.
- `T-COEX-05`: route `100.125.4.0/24` in A while B's overlay is `100.125.0.0/16`: B warns.
- `T-COEX-06`: A on an exit node that carries all internet traffic (lab only; neither of the
  owner's networks is a full tunnel). B's management, signal and relay connections leave
  through the physical adapter (`Get-NetTCPConnection -OwningProcess <B pid>` local addresses are on
  the physical NIC). Run `ipconfig /renew` on the physical adapter: B does not restart its engine
  (its log shows no network-monitor restart). Disconnect A: B stays connected without reconnecting.
- `T-COEX-07`: offer an auto-applied exit node in `lab-b`: B does not select it; `netbird --instance
  lfbb networks select <exit-node>` is refused with the specified message; no `/1` routes on
  `wt_lfbb`.

### T-PROF-01 Profiles in B

Add, select, rename and remove a profile in B; A's profiles and active profile are unchanged.
Removing B's profile deregisters only B's peer. (`SBS-PROF-01`)

### T-UI-01 to T-UI-06 Tray UI

- `T-UI-01`: both UIs run; starting B's shortcut again focuses B's window, not A's; `--daemon-addr`
  overrides.
- `T-UI-02`: tooltip, window title and menu header read `NetBird (LFBB)` for B and are unchanged for
  A.
- `T-UI-03`: sign out and in: both UIs start; disable autostart in B: A still starts. Install B
  for a user whose A autostart is already enabled: B's autostart is enabled too (first-run gate).
- `T-UI-04`: trigger a notification in each (disconnect, update available): the source is
  `NetBird` or `NetBird (LFBB)`. Start A's UI after B's, then trigger B's session-expiry warning and
  click "Extend now": B's session is extended, A's is not.
- `T-UI-05`: fire A's quick-actions event: only A's UI reacts. No UI ends another.
- `T-UI-06`: B's preferences, log and WebView2 data are in B's locations; A's files unchanged.

### T-COL-01, T-COL-03 Colour

- `T-COL-01`: palette names and hex values accepted, invalid values rejected; an invalid value
  written into the record by hand shows no dot and the canonical name.
- `T-COL-03`: re-run the installer with `/INSTANCE=lfbb /COLOR=skyblue`: after a UI restart the dot
  is sky blue. Check readability of every state on light and dark taskbars at 100 %, 150 % and
  200 % scaling.

### T-UPD-01 to T-UPD-03 Updater

- `T-UPD-01`: enable forced auto-update on `lab-b`: B does not run an installer; its UI may show the
  new version but offers no "Install" action and no download link to the upstream installer.
- `T-UPD-02`: on a fork default installation, trigger an update: only A's UI ends and restarts; B's
  UI keeps its PID.
- `T-UPD-03`: create a file in `%ProgramData%\Netbird\tmp-install`, reconnect B: the file is still
  there; reconnect A: it is cleaned as today.

### T-POL-01 to T-POL-03 Group policy

- `T-POL-01`: set `blockInbound=1` and `managementURL=https://vpn.airsecoma.net:443` in
  `HKLM\Software\Policies\NetBird`: both A and B block inbound; only A is locked to AIRSECOMA's
  URL; B keeps LFBB's URL.
- `T-POL-02`: set `managementURL` and `blockInbound=0` under
  `HKLM\Software\Policies\NetBird\Installations\lfbb`: B uses them; A is unaffected; an official
  v0.81.5 default installation ignores the subkey.
- `T-POL-03`: set `disableNamedInstallations=1`: B reports that named installations are disabled by
  policy and does not connect; the installer refuses `/INSTANCE=x` with exit code 2; A is unaffected.

### T-SEC-01 IPC

B's pipe exists as `ProtectedPrefix\Administrators\netbird_lfbb`; a standard user can call read-only
RPCs and is refused privileged ones exactly as with A; the existing `ipcauth` tests pass with the
derived name.

## 5. Regression on other platforms

- `make test-unit`, `make test-privileged` and the CI suites pass unchanged on Linux, macOS and
  FreeBSD; `T-DEF-01` and `T-PLAT-01` run there.
- Linux: a manual smoke test of the default `.deb` install and `netbird up` on a VM confirms `wt0`,
  `/var/lib/netbird`, the nftables table and the systemd unit are unchanged.
- macOS: a manual smoke test of the `.pkg` confirms `utun100`, the launchd label and the scutil keys
  are unchanged.

## 6. Traceability

Every requirement in `01-requirements.md` names at least one test, except the few whose acceptance
is a review, a visual check or an unchanged build (`SBS-ID-08`, `SBS-CLI-08`, `SBS-PROF-02`,
`SBS-UI-09`, `SBS-COL-03`, `SBS-COL-04`, `SBS-INS-11`, `SBS-UPD-06`, `SBS-PLAT-02`). The seed's eight
scenarios map as follows:

| Seed scenario | Tests |
|---|---|
| 1 Fresh install, both connected, colours, DNS | `T-WIN-02` |
| 2 Stop, restart, kill, reboot | `T-WIN-03`, `T-WIN-06` |
| 3 Upgrade one | `T-WIN-07`, `T-INS-06` |
| 4 Uninstall one | `T-WIN-04` |
| 5 Default upgrade identical | `T-WIN-01` |
| 6 Overlapping CIDRs | `T-COEX-03`, `T-COEX-04`, `T-COEX-05` |
| 7 SSH | `T-SSH-01` to `T-SSH-03` |
| 8 Invalid names | `T-INS-03`, `T-ID-02` |
