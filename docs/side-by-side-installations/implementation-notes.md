# Implementation notes

Where the code contradicted the specification, or the specification left a choice open, this file
records what was found and the conservative reading that was implemented. Line numbers are
v0.81.5 (`396976b8b`) unless stated otherwise.

## Step 1: identity package

| Requirement | Found | Done |
|---|---|---|
| `SBS-DEF-01`, `T-DEF-01` | The panic log path was built with `path.Join` (`client/server/panic_windows.go:31-32`), so its string is `C:\ProgramData/Netbird/netbird.err`. It names the same file as `C:\ProgramData\Netbird\netbird.err` | The identity package builds every Windows path with `\`. The golden table pins the backslash form, which is the value in the matrix. The file opened is unchanged |
| `SBS-DEF-01`, `T-DEF-01` | `configs.StateDir` and `profilemanager.DefaultConfigPathDir` are two readers of `NB_STATE_DIR` with different formats on non-Windows platforms (`/var/lib/netbird` against `/var/lib/netbird/`, and different platform lists) | Both formats are kept: `StateDir()` and `ProfileDir()` in the package, each pinned by the golden table for eight platforms. On Windows they are equal |
| `SBS-DEF-01` | `client/cmd/root.go:138-139` concatenated `PROGRAMDATA + "\\Netbird\\"`; `filepath.Join` gives the same value whenever `PROGRAMDATA` is set | The package joins with `\`. The two differ only if `PROGRAMDATA` is empty, which does not happen on Windows |
| `SBS-ID-05`, `T-ID-03` | `PORT(base)` has 100 slots, so two named installations can derive the same port (`a1` and `abcdefghijk` both get `51836`). Strict disjointness between named installations is impossible for this rule | The test asserts that derived ports never equal the default installation's ports, and that every other identifier is disjoint. The existing fallback to a free WireGuard port (`client/internal/connect.go:729-736`) covers a clash between two named installations; metrics and pprof need an explicit address in that case |
| `SBS-ID-04` | `iface.DefaultWgPort` (51820) is also the port assumed for **remote** peers (`client/internal/peer/worker_ice.go:306`) | Kept as the WireGuard well-known port. The local default comes from `instance.WireGuardPort()` |
| `SBS-ID-04` | Some exported identifiers of shared packages were constants: `daemonaddr.WindowsPipeAddr`, `iface.WgInterfaceDefault`, `configurer.WgInterfaceDefault`, `ssh.NetBirdSSHConfigFile`, `sshclient.DefaultDaemonAddr{,Windows}`, `localmetrics.DefaultListenAddress`, `netstack.DefaultSocks5Port`, `dns.NRPTKeyPrefix` | The first five are removed; the last three became functions that read the current installation. Nothing else in this repository used them. Builds outside the repository that import them would need the same change; listed as an open point |
| `SBS-ID-04` | Several values are computed in `init()` before flags are parsed: `configs.StateDir`, `profilemanager.DefaultConfigPath*`, the flag defaults of `--daemon-addr`, `--log-file`, `-s`, `--interface-name`, `--wireguard-port`, `--json-socket`, `--local-metrics-address` | In step 1 they read `instance.Current()`, which is the default installation. Step 2 sets the installation before they are used and re-derives them |

## Step 2: Windows isolation

| Requirement | Found | Done |
|---|---|---|
| `SBS-ID-04` | Packages keep derived values in package variables set by `init()` | `instance.SetCurrent` runs hooks that `configs` and `profilemanager` register; the CLI sets the installation in `PersistentPreRunE`, before any command reads those values. Flag defaults computed in `init()` are moved only when the flag was not set and still holds the default installation's value, so an explicit value, an environment variable or a saved `service.json` value wins (`SBS-DEF-05`) |
| `SBS-ID-09` | `SetFlagsFromEnvVars` (`client/cmd/root.go:247-269`) would set `--instance` from `NB_INSTANCE` for every command | `--instance` and `--instance-role` are exempt from it. `NB_INSTANCE` is read explicitly, and not for `service run` |
| `SBS-COEX-01`, `SBS-CLI-05` | The specification gives the CLI the role "from the record" | Commands other than `service run` take label and role from the record. Without a record (a service installed by hand) the role shown is `secondary`, the default of a named installation, even if the service was installed with `--instance-role primary` |
| `SBS-SVC-01` | | The service command line is `service run --instance <name> --instance-role <role>` followed by today's arguments. The two flags are inserted outside `buildServiceArguments`, whose guard test requires every identifier there to be a persisted service param |
| `SBS-SVC-04` | `CreateNamedPipe` fails with access denied both when the protected name is taken and when the caller is not an administrator, so the error alone cannot tell the cases apart | A named daemon probes the protected pipe before creating it and again after a failure; if it exists, the daemon refuses to start and names the pipe. A named daemon run by a standard user (netstack mode) still falls back to the plain name when the protected one does not exist |
| `SBS-SVC-05` | | The check runs for named installations only, through `GetServiceKeyNameW` for the display name. The default installation keeps today's behaviour. A failed display-name lookup refuses the install (fail closed) |
| `SBS-SVC-06` | Removing a leftover wintun adapter needs `golang.zx2c4.com/wintun`, which is only an indirect dependency; using it directly changes `go.mod` | Not implemented (it is a SHOULD). The service's own teardown closes the adapter. What wintun does with a leftover of the same GUID after a crash is UNVERIFIED; `T-WIN-04` and `T-WIN-05` show it |
| `SBS-SVC-07`, `SBS-DEF-05` | An explicit `--daemon-addr` normally wins over the derived address | A named daemon refuses to listen on the legacy `tcp://127.0.0.1:41731` even when it is given explicitly: `SBS-SVC-07` is a MUST NOT and the more restrictive reading |
| `SBS-FW-02` | `netsh advfirewall firewall show rule` prints localized field names | A named installation deletes any existing rule with its names and recreates it on every apply, instead of parsing `localip`. The default installation keeps reusing its rule |
| `SBS-NET-02` | | `up` checks an explicit `--interface-name` of a named installation: it must start with `wt`, and no adapter of that name may exist unless it is the installation's derived `wt_<name>`. A custom name already used by the installation's own adapter is refused too; pass the name only once |
| `SBS-NET-05` | The pprof listener of `pprof`-tagged builds starts in `init()` (`client/cmd/pprof.go`), before the name is known | Not changed: such builds keep `localhost:6969` unless `NB_PPROF_ADDR` is set. They are debug builds, not released ones |
| `SBS-CLI-07` | Hints exist in more places than the specification lists: "run 'netbird up' first", "run 'netbird down' first", and the `service install`/`service start` pair printed when the daemon is not reachable | All of them go through `ipcauth.CLICommand`. Cobra `Example` texts are documentation, not hints, and are unchanged |
| `SBS-UPD-01` | | A named installation's update manager reports auto-update as unsupported, so a management-pushed version becomes download-only (the newer version is still announced) and `TriggerUpdate` fails. The UI side is step 4 |
| `SBS-CLI-03` | | `netbird instance list` opens the service manager with connect access only, so a standard user can run it. The default installation's row has no version or folder unless its daemon answers |
| Verification | Cross-compiling `./client/...` for FreeBSD fails on the unmodified baseline (`wgctrl` needs a FreeBSD host); `go build ./...` fails on `magefiles` on the baseline too | FreeBSD was checked per package: the same 50 of 135 client packages fail before and after the change, all through `wgctrl` |

## Step 3: coexistence rules

| Requirement | Found | Done |
|---|---|---|
| `SBS-COEX-07` | Exit-node routes reach the route manager through `ClassifyRoutes` (`client/internal/routemanager/manager.go:654`), the only writer of the client routes that both auto-apply and user selection read | A secondary drops `0.0.0.0/0` and `::/0` there, so neither path can install them, and the split `/1` routes and the IPv6 split default are never planned. A selection that names a dropped exit node fails with `exit nodes are disabled in a secondary installation`. Exit nodes therefore do not appear in a secondary's `networks list` |
| `SBS-COEX-06` | | `GetBestInterface` skips every adapter named `wt*` for every installation; without another NetBird adapter this changes nothing. The legacy-routing path and the network monitor apply the rule on Windows only. The monitor still uses today's lookup and falls back to the route table only when that lookup lands on a NetBird adapter |
| `SBS-COEX-04` | | A secondary checks the addresses of every other `wt*` adapter after login, before the engine starts, and refuses to connect on an overlap (and when the adapters cannot be listed: fail closed). The refusal is the connection error and a system event, so `netbird status` and the UI show it. The connect loop retries with its usual backoff, so the secondary connects once the conflict is gone. Detecting an overlap that appears after connecting (SHOULD) is not implemented |
| `SBS-COEX-02`, `-03`, `-05` | | Reported as system events, each message once per process: another primary at connect (default service present, or a record with `Role=primary`), partial overlaps at connect, identical routes whenever the route selection is applied |
| `SBS-NET-04` | | The secondary's metric is set on the IPv4 interface with the MTU, and on the IPv6 interface with the MTU when the overlay has IPv6. Without IPv6 it is set on the IPv6 interface on a best-effort basis |
| `SBS-DNS-03` | A catch-all can also arrive as a `.` match domain, not only as `RouteAll` | A secondary drops both. The drop is logged once per process |
| `SBS-DNS-04` | | On every apply, a secondary reads the `Name` value of the other installations' NRPT keys in both stores and skips those namespaces; the skipped ones are published as a DNS warning event |
| `SBS-DNS-05` | | NetBird adapters are recognised by their friendly name (`wt*`) through `GetAdaptersAddresses` and skipped in the capture of fallback nameservers. If the adapter list cannot be read, only the own adapter is skipped, as today |
| `SBS-POL-01` to `-04` | | The split lives in `mdm.Loader.Load`, so the daemon, the UI and the reload ticker all see the same policy. `disableNamedInstallations` counts only machine-wide. It is added to `docs/netbird.admx` and `docs/netbird.adml`, the public surface that Q13 accepted. The installer side of `SBS-POL-04` is step 5 |
| Verification | `TestServer_SubcribeEvents` (`client/server`) fails under `-race` on the unmodified baseline too | Not related to this change; `make test-unit` runs without `-race` |

## Step 4: tray UI

| Requirement | Found | Done |
|---|---|---|
| `SBS-ID-09`, D2 | The specification gives the UI one channel, `--instance` | The UI reads `--instance` only, not `NB_INSTANCE`. An explicit `--daemon-addr` still wins (`SBS-UI-02`) |
| `SBS-COL-02` | "A dot about 40 % of the icon width with a ring about 6 %" can be read as the ring inside or outside the 40 % | The coloured dot is 40 % of the width and the ring is drawn around it, so the whole mark is 52 % of the width, touching the top and left edges. Checked at 16 px in a rendered preview: the dot stays clear of the status badge and the update arrow and overlaps the tail of the bird |
| `T-COL-02` | Golden images would add a dozen 256x256 PNGs to the repository | The golden test pins the SHA-256 of each composited icon's pixels (12 states and themes), and a pixel test checks that nothing outside the dot changes and that the dot and ring have the expected colours |
| `SBS-UI-05` | The toast library writes `DisplayName = AppUserModelID` (`NetBird_acme`) every time the UI starts (`go-toast` `wintoast/registry.go:48`) | The named UI writes its `CustomActivator` before Wails reads it, and rewrites `DisplayName` to `NetBird (ACME)` once the application has started. Whether Windows shows the rewritten name is UNVERIFIED; `T-UI-04` shows it |
| `SBS-UI-04` | The Wails fork's slug of `Options.Name` (`NetBird_acme`) is already `netbird_acme` | A named UI passes the identifier and `--instance <name>` explicitly through `EnableWithOptions`; the default UI calls `Enable()` as before |
| `SBS-UI-03` | The tray menu has no header row today | A named UI's menu starts with a disabled `NetBird (ACME)` row and a separator; the default menu is unchanged. Window titles from the window manager get the suffix ` (ACME)` |
| `SBS-UPD-01` | The UI offers the upstream installer in the tray's "update available" click and in the settings' update card | For a named installation `DownloadURL` returns nothing, the tray click does nothing and the card shows no download button; the "what's new" link to the release notes stays |
| `SBS-UI-09`, `SBS-COL-04` | Both are SHOULD and need new frontend surface | Not implemented in this delivery: the settings window does not show the installation details, the window header has no colour stripe, and toasts use the undotted icon |

## Step 5: NSIS installer

| Requirement | Found | Done |
|---|---|---|
| `SBS-INS-04`, `SBS-INS-01` | `.onInit` offers to uninstall an existing default installation before any page is shown (`client/installer.nsis:170-193`), so a user who then chose the advanced checkbox would already have removed the default installation | Silent runs without `/INSTANCE` still do this in `.onInit`, as before. Interactive runs ask on leaving the Welcome page, and only when the checkbox is off. The prompt therefore appears one page later than before; its effect is unchanged |
| `SBS-INS-05` | Today the default installation's folder can be changed on the Directory page, so a default installation does not have to live in `$PROGRAMFILES64\Netbird` | The uninstaller acts as the default installation when its folder is the default one **or** when `Uninstall\Netbird` points at this very uninstaller. A copy anywhere else, with no matching record, stops with exit code 2 and changes nothing |
| `SBS-INS-03` | Standard NSIS builds have no install log | Invalid parameters are written to `%TEMP%\netbird-installer.log` (and shown in a message box when not silent), and the installer exits with code 2. `/COLOR`, `/ROLE` and `/AUTOSTART` are ignored for the default installation, as specified, and not validated there |
| `SBS-INS-02`, `SBS-UI-04` | The UI enables autostart on its first start when the machine has no NetBird footprint | `/AUTOSTART=1` (the default) writes the Run value for the installing user. `/AUTOSTART=0` writes `{"autostartInitialized":true}` into the installation's `ui-preferences.json` when that file does not exist yet, so the UI's first start does not override the choice. Both happen on a fresh install only |
| `SBS-INS-07` | | Uninstallers end UI processes through PowerShell, by executable path inside the installation's folder. This replaces `taskkill /im` for the default installation too, as `03-installer-ux.md` section 6.3 asks |
| `03-installer-ux.md` 6.2 step 5 | | After a silent upgrade the UI is not restarted; an interactive install offers to start it on the Finish page, as today |
| `SBS-SVC-06`, `SBS-INS-10` step 8 | | The uninstaller does not remove a leftover wintun adapter (see step 2) |
| `SBS-POL-04` | | The installer refuses a named installation when `HKLM\Software\Policies\NetBird\DisableNamedInstallations` is non-zero: exit code 2 when silent, a message on the Welcome page when interactive |
| Verification | | The script compiles with `makensis -WX` (warnings as errors) against dummy payload files, with the EnVar and ShellExecAsUser plugins downloaded from the URLs and checksums of `release.yml`. In Wine (64-bit prefix): every invalid parameter of `T-INS-03` exits with 2 and the exact message; silent default and named installs, a named upgrade (label and colour kept), a stray uninstaller copy (refused), and uninstalling either installation while the other stays installed all leave the registry, folders, shortcuts and Run values as specified. The pages, the colour dialog and anything that needs a real service, adapter or PowerShell were not exercised |

## Step 6: fork build workflow

| Requirement | Found | Done |
|---|---|---|
| `SBS-UPD-05` | `release.yml` and `sync-tag.yml` run on every `v*` tag, so a fork tag `v0.81.5+sbs.1` would also start upstream's release pipeline in the fork | `.github/workflows/fork-installer.yml` runs on tags `sbs-v<upstream version>+sbs.<n>` (for example `sbs-v0.81.5+sbs.1`) and on manual dispatch with a version. The version is still `0.81.5+sbs.1`; NSIS gets `APPVER=0.81.5.1` and the record gets the full version |
| `SBS-UPD-05`, brief step 6 | `release.yml` cross-compiles the Windows binaries on Linux (mingw) and packages them with NSIS on a Windows runner | The workflow does the same: one Linux job builds `netbird.exe` and `netbird-ui.exe` with the release's flags and version resources, one `windows-2022` job adds `wintun.dll`, the WebView2 bootstrapper and the two NSIS plugins (URLs and checksums from `release.yml`) and runs `makensis`. Nothing is signed. The artifact is `netbird-installer.exe`, kept 30 days |
| Verification | | `actionlint` passes. The build commands of the Linux job were run here: both binaries build, and `makensis` packages them into a 19 MB installer. In Wine, that installer registers `Netbird_acme` with display name `NetBird (ACME)` and the command line of `SBS-SVC-01`, while the default service gets exactly the command line that the v0.81.5 binary registers. The workflow itself has not run: nothing could be pushed from this session |

## Final report

### 1. Steps

Sizes count added lines without tests and documentation.

| Step | Commit | What it does | Deviations |
|---|---|---|---|
| 1 Identity package | `ce07f747` | `client/internal/instance` derives every machine-wide identifier from the installation name, and today's literals moved behind it. The golden test `T-DEF-01` was written first, passed against the unmodified code, and pins the v0.81.5 values for eight platforms; a source scan (`T-ID-04`) keeps derived literals out of the rest of the client. About 800 lines | Ports cannot be disjoint between two named installations (100 slots); the panic log path is written with `\` only; exported constants became functions or were removed (step 1 table) |
| 2 Windows isolation | `0cc07479` | `--instance` and `--instance-role` with the specified precedence, service registration with a fail-before-write check, the protected pipe rule, no legacy TCP, state, log and per-user folders, legacy routing and `NB_STATE_DIR` refused with a warning, firewall rules recreated, SSH config commands, no upstream update, suggested commands with `--instance`, status fields and `netbird instance list`. About 900 lines | Leftover wintun adapters are not removed on uninstall (needs a `go.mod` change); pprof builds keep their port; the role shown by the CLI comes from the record (step 2 table) |
| 3 Coexistence rules | `24f5ec3a` | Secondaries ignore and refuse exit nodes, drop the DNS catch-all, skip namespaces another installation claims, use interface metric 1000 and refuse an overlapping overlay; every installation keeps its control traffic off every NetBird adapter; the policy split and `disableNamedInstallations`, also added to the ADMX template. About 950 lines | An overlap that appears after connecting is not detected (SHOULD); warnings are system events (step 3 table) |
| 4 Tray UI | `7e4a20a4` | Per-installation lock, application name, toast activator and source name, quick-actions event, WebView2 folder, autostart entry, titles, menu header, colour dot on every Windows tray icon (standard library only), and no upstream installer link. About 440 lines plus a small React change | The settings window does not show the installation details and there is no colour stripe (both SHOULD) (step 4 table) |
| 5 NSIS installer | `f2e65c9c` | Welcome-page checkbox and named-installation page, `/INSTANCE=`, `/COLOR=`, `/ROLE=`, `/AUTOSTART=`, per-installation registration and record, upgrade through the installation's own uninstaller, and an uninstaller that identifies its installation and fails closed. About 970 lines of NSIS | The default installation's upgrade prompt moved from start-up to leaving the Welcome page; the uninstaller also accepts a default installation in a custom folder; the install log is `%TEMP%\netbird-installer.log` (step 5 table) |
| 6 Fork build | `7cc1e9d2` | Workflow that builds an unsigned NSIS installer from a tag. About 240 lines | Tags are `sbs-v<version>+sbs.<n>`, not `v...`, so upstream's release workflows do not run (step 6 table) |

### 2. Verified here, and not

Verified in this Linux environment:

- `gofmt`; `go build ./...` (fails on `magefiles` on the baseline too); `GOOS=windows` and `GOOS=darwin` builds of the client (darwin without the cgo-only UI); WASM, Android vet and the embedded client. FreeBSD cannot be cross-compiled from Linux at baseline (`wgctrl`); per package, the same 50 of 135 client packages fail before and after.
- `golangci-lint run --new-from-rev=396976b8b`: 0 issues for every step, on the whole repository for Linux and on `./client/...` with `GOOS=windows`.
- `make test-unit` after steps 1 to 4: the only failures are the ones the unmodified baseline has in this environment (no Docker, no real DNS, no kernel modules: `client/iface/bind`, `client/iface/device`, `client/internal/dns`, `client/internal/dns/mgmt`, `sharedsock`, several `management` packages, `tools/mysql-migrate`, `upload-server`), plus three timing flakes that passed on rerun (`shared/relay/client`, `proxy`, `client/ssh/server`). Touched packages also pass with `-race`, except `TestServer_SubcribeEvents`, which races on the baseline too.
- The step tests of `05-test-plan.md` section 3 that run on Linux: `T-DEF-01`, `T-ID-01` to `-05`, `T-NET-03`, `T-COEX-U1`, `-U3`, `T-DNS-U1`, `-U2`, `T-SSH-U1`, `T-UPD-U1`, `T-POL-U1`, `T-CLI-U1`, `T-PLAT-01`, `T-COL-02`, and a fuzz run of `T-ID-02`. `T-COEX-U2` is Windows-only; it compiles here and runs in the Windows CI.
- Frontend: `pnpm install --frozen-lockfile`, `pnpm check`, `pnpm build`.
- Installer: compiled with `makensis -WX`; run in Wine as described in the step 5 and step 6 tables, including service registration with the real binaries.

Not verified here:

- Any Windows runtime behaviour: adapters, routes, metrics, NRPT, firewall, pipes, SSH, toasts, autostart, the tray icon on a real taskbar, and the installer's pages and colour dialog.
- `make test-privileged`: no Docker daemon is running in this environment.
- The fork workflow: it has not run, because nothing could be pushed.

### 3. Manual tests on a Windows VM, in this order

1. `T-WIN-01` upgrade of an official v0.81.5 default installation (the snapshot diff must show versions and timestamps only).
2. `T-WIN-02` fresh default plus `ACME`, both connected.
3. `T-WIN-08` `ACME` next to the official default installation, including the official installer and updater.
4. `T-SVC-01` to `T-SVC-04`, `T-SEC-01`.
5. `T-INS-01` to `T-INS-06`: pages, layout of the Welcome checkbox, the colour dialog, parameters, record ACL, stray uninstaller.
6. `T-WIN-03` to `T-WIN-07`: stop, kill, uninstall, stable adapter, reboot order, upgrades.
7. `T-FS-02`, `T-FS-03`, `T-CLI-01` to `T-CLI-04`, `T-PROF-01`.
8. `T-NET-02`, `T-NET-05`, `T-NET-06`, `T-FW-01`, `T-FW-02`, `T-SSH-01` to `T-SSH-03`.
9. `T-DNS-01` to `T-DNS-05`, `T-COEX-01` to `T-COEX-07` (lab networks).
10. `T-UI-01` to `T-UI-06`, `T-COL-01`, `T-COL-03`: in particular whether toasts show `NetBird (ACME)` (`T-UI-04`) and how the dot reads at 100 %, 150 % and 200 %.
11. `T-UPD-01` to `T-UPD-03`, `T-POL-01` to `T-POL-03`.
12. The Windows CI jobs of the fork (`golang-test-windows.yml`), which run the Windows-only tests (`T-COEX-U2`, the NRPT tests) and `T-DEF-01` on Windows.

### 4. Open points for the owner

1. **Push access.** The session's git proxy refused `jeromerg/netbird` ("not in this session's authorized repository set"). The six commits are on the local branch `claude/side-by-side-installations`; nothing is on GitHub. Add the repository to the session's sources, or fetch the bundle in the scratchpad.
2. **Size.** Steps 1 to 3 are 800 to 950 hand-written lines each and step 5 about 970 lines of NSIS, above the estimates in `00-overview.md` section 8 and the 400 lines `AGENTS.md` asks for per PR. Before proposing upstream, decide how to split them further (for example step 2 into service/IPC, files and suggested commands; step 3 into routing, DNS and policy).
3. **Public surface changes upstream would see:** the removed or changed exported identifiers of step 1, the new `disableNamedInstallations` ADMX value, the new `netbird instance` command, and the Welcome-page checkbox with the moved upgrade prompt of the default installation.
4. **SHOULDs left out:** leftover adapter removal (`SBS-SVC-06`), late overlap detection (`SBS-COEX-04`), installation details and colour stripe in the UI (`SBS-UI-09`, `SBS-COL-04`), pprof port (`SBS-NET-05`).
5. **Q4** is still open in `99-open-questions.md`: whether ACME configures a catch-all nameserver group, which a secondary drops.
