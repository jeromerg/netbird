# Identifier matrix

Every machine-wide or per-user-wide identifier that two installations could collide on, checked
against v0.81.5 (`396976b8b`). For each: today's value, the default installation's value (which must
be byte-for-byte today's value, `SBS-DEF-01`), the value for the named installation `acme` with label
`ACME`, the derivation rule, and where it lives in the code.

In the "Default installation" column, **same** means byte-for-byte identical to the "Today" column;
`T-DEF-01` pins each of these values. Line numbers are v0.81.5. "Seed" tells whether the seed's row was confirmed (OK), needed a
correction (Fix), or was missing (New); corrections are explained in the last section.

## 1. Derivation rules

`N` is the canonical name (lowercase, `^[a-z][a-z0-9]{0,11}$`), `L` the label as typed. The default
installation has `N = ""` and every rule returns today's value for it.

| Rule | Named installation | Example (`acme`) |
|---|---|---|
| **TOKEN** | Insert the installation marker `_N` right after the product token (`Netbird`, `NetBird`, `netbird`) | `Netbird` -> `Netbird_acme`; `NetBird-Match-0` -> `NetBird_acme-Match-0`; `99-netbird.conf` -> `99-netbird_acme.conf` |
| **SEG** | Append `.N` to a reverse-DNS identifier | `io.netbird.ui` -> `io.netbird.ui.acme` |
| **UUID(p)** | `UUIDv5(namespace, p + "/" + N)` with namespace = the default adapter GUID `f2f29e61-d91f-4d76-8151-119b20c4bdeb`, using `github.com/google/uuid` `NewSHA1` (already a direct dependency, `go.mod:14`) | `UUID(adapter)` = `{930adb3f-6a74-541d-b49f-0166bb47c7a5}` |
| **IF** | `wt_N` | `wt_acme` |
| **PORT(base)** | `base + k`, with `k = 1 + FNV-1a-32(N) mod 100` | `k = 16`: WireGuard `51836`, metrics `9207` |
| **DISPLAY** | `NetBird (L)` | `NetBird (ACME)` |
| **DIR** | Follows the installation's state directory or program folder | `%ProgramData%\Netbird_acme\client.log` |
| **ROLE** | Decided by the installation's role, not its name | secondary: interface metric `1000` |
| **NONE** | Not created for a named installation | |
| **SHARED** | Shared on purpose; justification given | |

Computed examples (Python `uuid.uuid5`, which implements the same RFC 4122 algorithm as `NewSHA1`):

| Name | UUID(adapter) | UUID(toast-activator) | UUID(msi-upgrade) | k | WireGuard port |
|---|---|---|---|---|---|
| `acme` | `{930adb3f-6a74-541d-b49f-0166bb47c7a5}` | `{5a3724b5-05f1-5ac3-9c29-1d659e422484}` | `{5591c79a-cc85-5950-ade1-37bbbac4eff2}` | 16 | 51836 |
| `globex` | `{a6d99cb5-b962-55f8-a433-705e9bdecab3}` | `{c3d5f26a-b0a6-5c4d-bcde-c4ee39f0631e}` | `{af29a36a-4d53-56d5-b302-938f316b97b6}` | 23 | 51843 |
| `work` | `{ac400ff9-5e17-51ea-8325-48735885ac3a}` | `{6dc35970-7893-52d2-ad74-eae475dfd451}` | `{d2080a14-0751-579c-a860-9254ecfd4725}` | 85 | 51905 |

## 2. Service and daemon IPC

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Service name | `Netbird` (`-s/--service`, `NB_SERVICE`); MSI registers `NetBird`, the same service for SCM | `Netbird` | `Netbird_acme` | TOKEN | `client/cmd/service.go:76-79,89`; `client/netbird.wxs:35` | OK |
| Service DisplayName | `Netbird` (fixed); MSI `NetBird` | `Netbird` | `NetBird (ACME)` | DISPLAY. Must be unique: SCM refuses a second service with the same display name | `client/cmd/service.go:110` | Fix |
| Service description | `NetBird mesh network client` | same | `NetBird mesh network client (ACME)` | DISPLAY | `client/cmd/service.go:111` | New |
| Service command line | `service run --log-level info --daemon-addr npipe://netbird --log-file <...>` plus optional flags | same | adds `--instance acme --instance-role secondary`; derived `--daemon-addr`, `--log-file` | TOKEN, DIR | `client/cmd/service_installer.go:32-75` | New |
| Service `Environment` value (only with `--service-env`) | `HKLM\SYSTEM\CurrentControlSet\Services\Netbird\Environment` | same | `...\Services\Netbird_acme\Environment` | follows service name | `client/cmd/service.go:113-122` | New |
| Event Log source | `Netbird` (registered by kardianos on install, never written to) | same | `Netbird_acme` | follows service name | `netbirdio/service` fork, `Install` | OK |
| Daemon address | `npipe://netbird` | same | `npipe://netbird_acme` | TOKEN | `client/internal/daemonaddr/pipe.go:18`; `client/cmd/root.go:153-158` | OK |
| Daemon pipe paths | `\\.\pipe\ProtectedPrefix\Administrators\netbird`, then `\\.\pipe\netbird` | same | `...\Administrators\netbird_acme`, then `\\.\pipe\netbird_acme` | from address | `client/internal/daemonaddr/pipe.go:32,58-82`; server `client/cmd/service_pipe_windows.go:26-41` | Fix |
| Legacy TCP address and its migration | `tcp://127.0.0.1:41731`, probed for the default address; `MigrateLegacy` rewrites it to `npipe://netbird` | same | never probed, never migrated (a named daemon on the legacy address would be moved onto the default pipe) | NONE | `client/internal/daemonaddr/pipe.go:22,97`; `resolve_pipe_windows.go:35-55` | New |
| JSON gateway socket (opt-in, also on Windows) | `unix:///var/run/netbird-http.sock` | same | `unix:///var/run/netbird_acme-http.sock` | TOKEN | `client/cmd/service.go:27,86-87` | New |
| Install-time parameters | `<StateDir>\service.json` | same | `%ProgramData%\Netbird_acme\service.json` | DIR | `client/cmd/service_params.go:21,40-43` | OK |

## 3. Machine files and state

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| State directory | `%ProgramData%\Netbird`; `NB_STATE_DIR` overrides (read in two places) | same, `NB_STATE_DIR` still honoured | `%ProgramData%\Netbird_acme`; `NB_STATE_DIR` ignored with a warning | TOKEN | `client/configs/configs.go:16-29`; `client/internal/profilemanager/service.go:65-87` | OK |
| Profiles (daemon side) | `<StateDir>\default.json`, `active_profile.json`, legacy `config.json`; per user `<StateDir>\<user>\<id>.json`, `.state.json`, `.prefs.json` | same | same names under the named state directory | DIR | `client/internal/profilemanager/service.go:84-86,212-243,435-441`; `config.go:289-304` | OK |
| Peer identity | WireGuard `PrivateKey` and `SSHKey` inside the profile JSON | same | own keys, generated at first login | DIR | `client/internal/profilemanager/config.go:114,153,449-456` | OK |
| Unclean-shutdown state | `<StateDir>\state.json` (`NB_DNS_STATE_FILE` overrides) | same | `%ProgramData%\Netbird_acme\state.json` | DIR | `client/internal/profilemanager/service.go:456-489`; replay `client/server/state.go:118-145` | New |
| Daemon log | `%ProgramData%\Netbird\client.log`, rotated `client*.log.*` (hard-coded, ignores `NB_STATE_DIR`) | same | `%ProgramData%\Netbird_acme\client.log` | DIR | `client/cmd/root.go:131-162`; `util/log.go:154-164` | OK |
| Panic / stderr log | `%ProgramData%\Netbird\netbird.err` (hard-coded) | same | `%ProgramData%\Netbird_acme\netbird.err` | DIR | `client/server/panic_windows.go:31-32`; debug bundle looks next to the log file, `client/internal/debug/debug.go:1124-1133` | OK |
| Legacy migration source | `%ProgramData%\Wiretrustee\` | same | no migration | NONE | `client/cmd/root.go:138-142,373-393` | New |
| Updater temp folder | `%ProgramData%\Netbird\tmp-install` (`updater.exe`, `result.json`, `msi.log`, `installer.log`); cleaned by **every** daemon on each connect | same | `%ProgramData%\Netbird_acme\tmp-install` | DIR | `client/internal/updater/installer/installer_run_windows.go:37`; cleanup `client/internal/connect.go:313-316` | Fix |
| Debug bundles | `%TEMP%\netbird.debug.*.zip` of the daemon | same | `netbird_acme.debug.*.zip` | TOKEN | `client/internal/debug/debug.go:384,413` | New |

## 4. Per-user files

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Per-user directory | `%APPDATA%\netbird` (`os.UserConfigDir()`) | same | `%APPDATA%\netbird_acme` | TOKEN | `client/internal/profilemanager/config.go:247-268` | OK |
| Active profile (CLI and UI) | `active_profile.txt`; the CLI sends this profile to whichever daemon it talks to | same | in `%APPDATA%\netbird_acme` | DIR | `client/internal/profilemanager/profilemanager.go:17,118-171`; `client/cmd/up.go:126-143,395-396` | Fix |
| Account e-mail per profile | `<profileID>.state.json` | same | in `%APPDATA%\netbird_acme` | DIR | `client/internal/profilemanager/state.go:22-84` | New |
| UI preferences | `ui-preferences.json` | same | in `%APPDATA%\netbird_acme` | DIR | `client/ui/preferences/store.go:25,380-384` | OK |
| UI log | `gui-client.log`, rotated `gui-client*.log.*` | same | in `%APPDATA%\netbird_acme` | DIR | `client/configs/configs.go:12`; `client/ui/uilogpath.go:18-24` | OK |
| Kubeconfig entries (`netbird kubernetes`) | `~/.kube/config`: user entry `netbird`, cluster contexts, `current-context` | same | user entry `netbird_acme` | TOKEN | `client/cmd/kubernetes.go:225-270` | New |
| Commands suggested in error messages | `netbird down; netbird up ...`, `netbird logout`, `netbird debug bundle ...`, without an installation | same | include `--instance acme` | adds `--instance` | `client/internal/ipcauth/privileged.go:131-150`; `client/server/ssh_gate.go:95,121,142-143,220`; `client/server/debug_gate.go:74,87` | New |

## 5. Adapter, interface and ports

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Wintun adapter GUID | `{f2f29e61-d91f-4d76-8151-119b20c4bdeb}`; `CustomWindowsGUIDString` override, not set by any build | same | `{930adb3f-6a74-541d-b49f-0166bb47c7a5}` | UUID(adapter) | `client/iface/device/device_windows.go:20,48-63`; `windows_guid.go:3-4` | OK |
| GUID-keyed registry | `Tcpip\Parameters\Interfaces\{GUID}`, `Tcpip6\...\{GUID}`, `NetBT\Parameters\Interfaces\Tcpip_{GUID}` | same | same keys under the named GUID | follows GUID | `client/internal/dns/host_windows.go:59-60,102,260-297,590-597` | Fix |
| Interface name | `wt0` (`--interface-name`); NetBird finds its own adapter by this name in several places | same | `wt_acme` | IF | `client/iface/configurer/name.go:6`; `client/cmd/up.go:73`; lookups listed in [`04-coexistence.md`](04-coexistence.md#3-adapters-and-overlay-addresses) | Fix |
| WireGuard UAPI pipe | per interface name (wireguard-go) | same | per `wt_acme` | follows IF | `client/iface/configurer/uapi_windows.go:10` | New |
| WireGuard UDP port | `51820`, falls back to a random free port when busy; persisted as `WgPort` in a new profile | same | `51836` | PORT(51820) | `client/iface/iface.go:32`; `client/internal/connect.go:729-736,827-851`; `client/internal/profilemanager/config.go` (`newConfigSkeleton`) | OK |
| Interface metric | automatic (never set) | same | `1000` when secondary | ROLE | none today | New |
| Local metrics (opt-in) | `127.0.0.1:9191`, bind failure logged | same | `127.0.0.1:9207` | PORT(9191) | `client/internal/localmetrics/localmetrics.go:23,68-72,115-119` | OK |
| pprof (build tag `pprof`) | `localhost:6969` (`NB_PPROF_ADDR`); bind failure is `log.Fatalf` | same | `localhost:6985` | PORT(6969) | `client/cmd/pprof.go:19-31` | Fix |
| Netstack SOCKS5 | `127.0.0.1:1080` (`NB_SOCKS5_LISTENER_PORT`, `_ADDRESS`), netstack mode only | same | `127.0.0.1:1096` | PORT(1080) | `client/iface/netstack/proxy.go:12`; `env.go:17-29` | Fix |
| In-memory DNS resolver | packet hook on the overlay's last IP, port 53; no socket | same | same mechanism, own overlay | SHARED (per overlay IP) | `client/internal/dns/server.go:229-233`; `service_memory.go:29-39` | New |
| DNS forwarder | `overlayIP:22054` (`NB_DNS_FORWARDER_PORT`) | same | own overlay IP | SHARED (per overlay IP) | `client/internal/dns/dns.go:22-25` | New |
| NetBird SSH server | overlay v4/v6 `:22022` | same | own overlay IPs | SHARED (per overlay IP) | `client/ssh/server/server.go:38`; `client/internal/engine_ssh.go:277-295` | New |
| Wintun tunnel type | `"WireGuard"` (wireguard-go default) | same | same | SHARED: it labels the adapter kind, not the adapter | wireguard-go `tun` | New |
| UPnP / PCP mapping description | `"NetBird"` | same | same | SHARED: mappings are keyed by port | `client/internal/portforward/manager.go:21` | New |
| Rosenpass, lazy-connection listeners, relay, STUN | ephemeral ports | same | same | SHARED | `client/internal/rosenpass/manager.go:175,309-321`; `client/internal/lazyconn/activity/listen_ip.go:8` | New |
| SSO (PKCE) callback listener | first free port among the management-supplied redirect URLs | same | same | SHARED: two logins at the same moment fail only if the IdP offers a single port | `client/internal/auth/pkce_flow.go:103-111,221` | New |
| Legacy routing uplink host routes (`NB_USE_LEGACY_ROUTING=true`) | `/32` routes to management, signal, relay and STUN via the physical adapter, refcounted in `state.json` | same | not used: legacy routing is refused for named installations (`SBS-NET-07`) | NONE | `client/internal/routemanager/systemops/systemops_windows.go:196-296`; `systemops_generic.go:36-60` | New |
| Network monitor initial next hop | `GetNextHop(0.0.0.0)` and `(::)` at start, without skipping NetBird adapters | same, plus skip other NetBird adapters | skips every NetBird adapter | rule `SBS-COEX-06` | `client/internal/networkmonitor/monitor.go:57-58` | New |
| Machine-wide DNS side effects | `ipconfig /registerdns`, `DnsFlushResolverCache`, `RefreshPolicyEx(RP_FORCE)` on apply and restore | same | same | SHARED: idempotent | `client/internal/dns/host_windows.go:509-541,780` | New |
| SSH server LSA labels | logon process `NetBird`, token source `netbird` | same | same | SHARED: labels only, logon IDs are unique | `client/ssh/server/executor_windows.go:240,432,437` | New |
| Certificate posture stores | `LocalMachine\MY`, `CA`, `CurrentUser\MY` (read-only) | same | same | SHARED: read-only | `client/internal/certproof/systemstore_windows.go:20-21,67-75` | New |

## 6. DNS

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| NRPT key prefix (cleanup matches it case-insensitively) | `NetBird-` | same | `NetBird_acme-` | TOKEN | `client/internal/dns/host_windows.go:43,727-755` | OK |
| NRPT match rules | `NetBird-Match-<n>` (one key per 50 domains) | same | `NetBird_acme-Match-<n>` | TOKEN | `host_windows.go:48,395-440` | Fix |
| NRPT `.local` exemption | `NetBird-ExemptLocal` | same | `NetBird_acme-ExemptLocal` (only if primary) | TOKEN | `host_windows.go:67-68,442-465` | New |
| NRPT stores | local `SYSTEM\CurrentControlSet\Services\Dnscache\Parameters\DnsPolicyConfig`; GPO `SOFTWARE\Policies\Microsoft\Windows NT\DNSClient\DnsPolicyConfig` | same | same | SHARED: the operating system's stores | `host_windows.go:51,55,665-722` | OK |
| Adapter `NameServer` (route-all), `SearchList` | under the adapter's GUID key | same | under the named GUID; `NameServer` only if primary | follows GUID, ROLE | `host_windows.go:386-393,548-556` | Fix |
| "Original" nameserver capture | every adapter except its own GUID | same, plus skip other NetBird adapters | skip every NetBird adapter | rule `SBS-DNS-05` | `host_windows.go:153-215` | Fix |
| Legacy resolution switch | `NB_USE_LEGACY_DNS_RESOLUTION` (process environment) | same | same, per service | per process | `host_windows.go:81` | New |

## 7. Firewall

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Windows Firewall inbound rule (v4) | `Netbird`, `localip=<overlay v4>`; reused if a rule with that name exists; deleted by name | same | `Netbird_acme`, recreated if its `localip` is stale | TOKEN | `client/firewall/uspfilter/interface_allower_windows.go:20,36-86` | OK |
| Windows Firewall inbound rule (v6) | `Netbird-v6` | same | `Netbird_acme-v6` | TOKEN | `interface_allower_windows.go:54-55,82-83` | OK |
| Userspace filter, WFP | per process; no WFP code exists at v0.81.5 | same | same | none | `client/firewall/create.go:15-29` | New |

## 8. SSH

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Client config file | `%ProgramData%\ssh\ssh_config.d\99-netbird.conf` (Linux and macOS `/etc/ssh/ssh_config.d/99-netbird.conf`) | same | `%ProgramData%\ssh\ssh_config.d\99-netbird_acme.conf` | TOKEN | `client/ssh/common.go:20-23`; `client/ssh/config/manager.go:111-124` | OK |
| `ProxyCommand` | `<exe> ssh proxy %h %p` (default daemon address; `NB_DAEMON_ADDR` in the ssh process overrides) | same | `<exe> --instance acme ssh proxy %h %p` | adds `--instance` | `client/ssh/config/manager.go:188,199,290-303`; `client/cmd/ssh.go:834` | Fix |
| `Match ... exec` | `<exe> ssh detect %h %p` (probes host and port, no daemon) | same | `<exe> --instance acme ssh detect %h %p` | adds `--instance` | `client/ssh/config/manager.go:169-212`; `client/cmd/ssh.go:860-888` | Fix |
| SSH config shutdown state | `ssh_config_state` in `state.json` | same | in the named `state.json` | DIR | `client/ssh/config/shutdown_state.go:15-22` | New |
| Known-hosts fallback | `%ProgramData%\ssh\ssh_known_hosts.d\99-netbird`, read by `netbird ssh`, never written at v0.81.5 | same | same | SHARED: read-only | `client/ssh/client/client.go:425-446` | New |

## 9. Tray UI

The UI is a Wails v3 application built on the fork
`github.com/netbirdio/wails/v3 v3.0.0-beta.3.0.20261009120932-4e087a8bfd6d` (`go.mod:384`); fork
paths are relative to that module's root, read at commit `4e087a8bfd6d`. Wails `Options.Name`
(`"NetBird"`, `client/ui/main.go:282`) feeds several rows at once: the AUMID, the autostart value
name, the notification-categories key and the toast icon file. Deriving it per installation covers
all of them.

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Single-instance lock | `UniqueID: "io.netbird.ui"`: session mutex `wails-app-io.netbird.ui-sim` and message window class `...-sic`; a second process hands its arguments to the first and exits | same | `io.netbird.ui.acme` | SEG | `client/ui/main.go:300-305`; fork `pkg/application/single_instance_windows.go:34-79` | OK |
| UI flags | `-daemon-addr` (default `npipe://netbird`), `-log-file`, `-log-level`; no environment variable for the address | same | adds `--instance acme`, from which the address is derived | TOKEN | `client/ui/main.go:244-249`; `client/ui/grpc.go:66-71` | OK |
| Other users of the daemon address in the UI | Settings privilege checks; the elevated one-shot `--apply-privileged-settings` (passes `--daemon-addr` on) | same | derived address | from address | `client/ui/services/settings.go:312,332`; `client/ui/services/guarded.go:133-139` | New |
| Wails `Options.Name` | `NetBird` | same | `NetBird_acme` | TOKEN | `client/ui/main.go:274-282` | New |
| Toast AUMID | `NetBird` (from `Options.Name`) | same | `NetBird_acme` | TOKEN | `client/ui/main.go:274-282` | OK |
| AUMID registration | `HKCU\Software\Classes\AppUserModelId\NetBird` (`CustomActivator`); pre-seeded only by the MSI, written by the fork otherwise | same | `...\AppUserModelId\NetBird_acme` | TOKEN | fork `pkg/services/notifications/notifications_windows.go:688-713`; `client/netbird.wxs:55-69` | Fix |
| Toast activator CLSID | MSI: `{0E1B4DE7-E148-432B-9814-544F941826EC}`; NSIS installs: generated per user by the fork | same | `UUID(toast-activator)` = `{5a3724b5-05f1-5ac3-9c29-1d659e422484}`, written by the UI before Wails reads it | UUID | `client/netbird.wxs:23,27,66`; fork `notifications_windows.go:688-713` | Fix |
| COM server registration | `HKCU\Software\Classes\CLSID\{clsid}\LocalServer32` rewritten to the running UI's path at every start: the last UI started wins | same | under its own CLSID | follows CLSID | fork `notifications_windows.go:93-103` | New |
| Notification categories | `HKCU\SOFTWARE\NetBird\NotificationCategories` | same | `HKCU\SOFTWARE\NetBird_acme\NotificationCategories` | TOKEN (from `Options.Name`) | fork `notifications_windows.go:39,618-639` | New |
| Toast icon file | `%TEMP%\NetBird{guid}.png` | same | `%TEMP%\NetBird_acme{guid}.png` | TOKEN (from `Options.Name`) | fork `notifications_windows.go:85,609-616` | New |
| Toast tags | `netbird-session-expired`, `netbird-session-warning`, `netbird-update-*`, `netbird-event-*` | same | same tags; isolated by the AUMID | SHARED (per AUMID) | `client/ui/tray.go:27-30`; `client/ui/tray_session.go:19-22` | New |
| Autostart | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, value `netbird`, data = quoted UI path only. The fork finds "its" value **by executable path** and deletes any other value pointing at the same executable | same | value `netbird_acme`, data `"<folder>\netbird-ui_acme.exe" --instance acme` (`AutostartOptions.Identifier` and `Arguments`). Isolation needs a distinct executable path, which model A provides | TOKEN | `client/ui/services/autostart.go:41-52`; fork `autostart_windows.go:13,38-60,105-136`, `autostart.go:90-132` | Fix |
| First-run autostart gate | `autostartInitialized` in the shared `ui-preferences.json`, then a footprint check of `<StateDir>\{default.json,config.json,state.json}` | same | own preferences file and own state directory | DIR | `client/ui/autostart_default.go:55-93` | Fix |
| Quick-actions event | `Global\NetBirdQuickActionsTriggerEvent`, manual reset (one set wakes every UI) | same | `Global\NetBird_acmeQuickActionsTriggerEvent` | TOKEN | `client/ui/signal_windows.go:15,31-35` | OK |
| WebView2 user data | not set, so the library default `%APPDATA%\netbird-ui.exe` | same | `%APPDATA%\netbird_acme\webview2` through `Options.Windows.WebviewUserDataPath` | DIR | `client/ui/main.go:296-299`; fork `internal/webview2/pkg/edge/chromium.go:186-194` | New |
| Tray tooltip | i18n `tray.tooltip` = `NetBird` (Windows truncates at 127 UTF-16 units) | same | `NetBird (ACME)` | DISPLAY | `client/ui/tray.go:183,301`; `client/ui/i18n/locales/en/common.json:2-5` | OK |
| Window titles | main window `"NetBird"`; others from i18n `window.title.*`; `frontend/index.html` `<title>NetBird</title>` | same | suffixed ` (ACME)` | DISPLAY | `client/ui/main.go:363`; `client/ui/windowmanager.go:317-1232` | Fix |
| Tray icons | 256x256 RGBA PNGs, light and dark variants, downscaled by Windows to 16 px at 100 %; status badge bottom-right, update arrow top-right | same bytes | same PNGs with a colour dot composited top-left | colour | `client/ui/tray_icon.go:14-136`; `client/ui/assets/netbird-systemtray-*.png` | OK |
| Download link for updates | read from `HKLM\...\App Paths\Netbird` to choose between the EXE and MSI download | same | no upstream download link (`SBS-UPD-01`) | NONE | `version/url_windows.go:13-28`; `client/ui/services/update.go:40`; `client/ui/tray_update.go:89` | New |
| Fallback command shown when elevation fails | `netbird down; netbird up ...` without a daemon address | same | includes `--instance acme` | adds `--instance` | `client/ui/services/guarded.go:219-226`; `client/internal/ipcauth/privileged.go:147-149` | New |
| Windows 11 tray "show icon" setting | keyed by executable path and icon id | same | separate, because the executable path differs | follows path | Windows (UNVERIFIED) | New |

## 10. Named kernel objects

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Debug trigger event (opt-in, `NB_LISTEN_DEBUG_EVENT`) | `Global\NetbirdDebugTriggerEvent`, nil ACL, manual reset | same | `Global\Netbird_acmeDebugTriggerEvent` | TOKEN | `client/cmd/debug_windows.go:19-20,57-63` | Fix (line) |
| Quick-actions event | `Global\NetBirdQuickActionsTriggerEvent` | same | `Global\NetBird_acmeQuickActionsTriggerEvent` | TOKEN | `client/ui/signal_windows.go:15` | OK |

No other named mutex, semaphore, event or file mapping exists in the Go code of `client/`.

## 11. Installer registration and updater

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Program folder | `C:\Program Files\Netbird` (forced in `.onInit`) | same | `C:\Program Files\Netbird_acme` | TOKEN | `client/installer.nsis:13,171` | OK |
| Uninstall key | `HKLM\...\Uninstall\Netbird` (DisplayName `Netbird`) | same | `...\Uninstall\Netbird_acme` (DisplayName `NetBird (ACME)`) | TOKEN, DISPLAY | `client/installer.nsis:17,221-225` | OK |
| Uninstaller | `$INSTDIR\netbird_uninstall.exe` | same | in the named folder | DIR | `client/installer.nsis:219` | New |
| `App Paths` | `App Paths\Netbird`, `App Paths\Netbird-ui` (no `.exe`) | same | none | NONE | `client/installer.nsis:16,22,220,227` | Fix |
| `PATH` | program folder appended (HKLM, EnVar) | same | not changed | NONE | `client/installer.nsis:229-230` | OK |
| Shortcuts | `$SMPROGRAMS\Netbird.lnk`, `$DESKTOP\Netbird.lnk` -> `Netbird-ui`, no AUMID property (only the MSI shortcuts carry one) | same | `NetBird (ACME).lnk` -> `netbird-ui_acme.exe --instance acme` | DISPLAY | `client/installer.nsis:232-234`; `client/netbird.wxs:21-28` | OK |
| UI executable name | `netbird-ui.exe` | same | `netbird-ui_acme.exe` | TOKEN | installer only | New |
| Installation record | none | none | `HKLM\SOFTWARE\NetBird\Installations\acme` | new key | none today | New |
| UI termination on (un)install | `taskkill /im Netbird-ui.exe /f` (all sessions, all installations) | by path, own folder only (fork) | by path, own folder only | path | `client/installer.nsis:291`; `client/netbird.wxs:84-85` | OK |
| MSI UpgradeCode | `6456ec4e-3ad6-4b9b-a2be-98e81cb21ccf` | same | deferred: `{5591c79a-cc85-5950-ade1-37bbbac4eff2}` | UUID(msi-upgrade) | `client/netbird.wxs:4,14` | OK |
| MSI Component GUID | `db3165de-cc6e-4922-8396-9d892950e23e` | same | deferred: `UUID(msi-component)` | UUID | `client/netbird.wxs:18` | New |
| Updater installer-type probe | `Uninstall\Netbird` in both registry views | same | updater disabled | NONE | `client/internal/updater/installer/types_windows.go:13-40` | OK |
| Updater UI kill | `netbird-ui.exe` by image name, all sessions; relaunches in those sessions | same | updater disabled | NONE | `installer_run_windows.go:143-216` | OK |
| Updater service restart | `<dir>\netbird.exe service start` (default name) | same | updater disabled | NONE | `installer_run_windows.go:129-134` | OK |
| Download URLs | `github.com/netbirdio/netbird/releases/...`, `pkgs.netbird.io/...` | same | not used | NONE | `installer_run_windows.go:32-33`; `version/update.go:18`; `version/url_windows.go:9-28` | New |

## 12. Policy (GPO / MDM)

| Resource | Today (v0.81.5) | Default installation | Named `acme` | Rule | Code | Seed |
|---|---|---|---|---|---|---|
| Windows policy key | `HKLM\Software\Policies\NetBird`, values only (`ReadValueNames`), reloaded every minute | same | posture values from this key apply; identity values do not (`SBS-POL-01`, `-02`) | split | `client/mdm/policy_windows.go:18,81,95`; `client/mdm/ticker.go:16` | Fix |
| Per-installation policy key | none | none | `HKLM\Software\Policies\NetBird\Installations\acme`, same value names; invisible to v0.81.5 because it reads values, not subkeys | new key | none today | New |
| ADMX / ADML | `docs/netbird.admx`, `docs/netbird.adml` (the comment at `policy_windows.go:17` points to a non-existent `docs/mdm/`) | same | per-installation keys cannot be expressed in ADMX (the key path holds the name); set them with registry preferences, Intune custom OMA-URI or `.reg` | none | `docs/netbird.admx` | Fix |
| macOS managed preferences | `/Library/Managed Preferences/io.netbird.client.plist` | same | deferred | | `client/mdm/policy_darwin.go:26` | New |

Policy values and how they apply to a named installation (`client/mdm/policy.go:22-76`, applied in
`client/internal/profilemanager/config.go:867-958`):

| Value | Kind | Named installation reads it from |
|---|---|---|
| `managementURL`, `preSharedKey` | identity: per network | per-installation key only |
| `wireguardPort`, `enableLocalMetrics`, `localMetricsAddress` | per installation: would collide | per-installation key only |
| `allowServerSSH`, `allowRemoteJobs`, `disableClientRoutes`, `disableServerRoutes`, `blockInbound`, `disableAutoConnect`, `rosenpassEnabled`, `rosenpassPermissive`, `lazyConnection`, `debugBundleUploadURL` | machine posture | global key, overridden by the per-installation key |
| `disableProfiles`, `disableNetworks`, `disableUpdateSettings`, `disableAdvancedView`, `disableAutostart`, `disableMetricsCollection` | machine posture (feature gates) | global key, overridden by the per-installation key |
| `splitTunnelMode`, `splitTunnelApps` | Android only | not applicable |

## 13. Matching rules and disjointness

D3 in [`00-overview.md`](00-overview.md#53-d3-the-default-installation-keeps-todays-identifiers-agreed-and-strengthened)
requires that no named identifier is matched by any pattern v0.81.5 uses on its own resources, and
that no named installation matches another's. The patterns, and why they cannot match:

| Matcher at v0.81.5 | Kind | Default identifier | Can it match a named identifier? |
|---|---|---|---|
| NRPT cleanup | case-insensitive prefix `netbird-` | `NetBird-Match-<n>`, `NetBird-ExemptLocal` | No: named keys start `netbird_`; the ninth character differs |
| Firewall `netsh ... name=` | exact name | `Netbird`, `Netbird-v6` | No: named rules contain `_` |
| Service, pipe, files, registry keys, events, Run value | exact | see tables | No: every named identifier contains `_N`, and no runtime identifier of v0.81.5 contains `_` after the product token (repository-wide search; it occurs only in build artifact names and server-side environment variables) |
| ICE interface blacklist | case-sensitive prefix `wt` (and others) | `wt0` | **Yes, on purpose**: `wt_N` is excluded from ICE by every installation, including unmodified ones |
| `taskkill /im netbird-ui.exe` (NSIS, MSI, updater) | image name | `netbird-ui.exe` | No, because named UIs are `netbird-ui_N.exe` (`SBS-INS-08`) |
| Wails autostart lookup (`enable`, `status`, `disable`) | executable path | `C:\Program Files\Netbird\netbird-ui.exe` | No: each installation has its own folder (model A) |
| MDM loader | values of one key | `HKLM\Software\Policies\NetBird` | No: the per-installation policy is a subkey, and only values are read |

Between two named installations `a` and `b` (`a != b`):

- exact identifiers differ because `N` differs;
- the only prefix matcher a named installation uses is `NetBird_N-` for NRPT. The character after `N`
  is `-`, which is not allowed in names, so `NetBird_a-` cannot match `NetBird_ab-...` or
  `NetBird_a1-...`.

Reserved names: `default` (used by the CLI to address the default installation explicitly) and
`netbird` (it would produce `Netbird_netbird`, which reads like a typo). The rules are tested by
`T-ID-03` in [`05-test-plan.md`](05-test-plan.md).

## 14. Corrections to the seed

| Seed row | Correction |
|---|---|
| 5.1 Wintun GUID | Confirmed. Whether wintun refuses a second adapter with the same GUID is decided by wintun, not by NetBird code. NetBird code does not touch NLA |
| 5.1 Interface name | `wt0` only on Linux, Windows, FreeBSD and JS; macOS uses `utun100` (`client/iface/configurer/name_darwin.go:6`). The ICE blacklist is seeded only into configs whose list is empty (`client/internal/profilemanager/config.go:610-615`), and matched case-sensitively by prefix. **Missed:** NetBird looks up its own adapter by name in six places, so the name must be unique, not just the GUID |
| 5.1 NRPT | Worse than stated: both installations write the **same** key paths, and `configureDNSPolicy` deletes the key before writing it (`host_windows.go:472`), so they overwrite each other even without the prefix cleanup. Unclean-shutdown cleanup is at `:653-655`. Keys are volatile (`:476`) |
| 5.1 GPO NRPT store | The store is deleted only when truly empty (`SubKeyCount == 0 && ValueCount == 0`, `:712`). The hazard is indirect: B's prefix cleanup empties it, then B deletes it, while A keeps writing there because its `gpo` flag is fixed at start (`:129`) |
| 5.1 Original nameservers | Skips only its own GUID (`:188`) and runs once at start (`:132`). A's `NameServer` value exists only while A is route-all and is not removed by `restoreHostDNS` (`:599-619`) |
| 5.1 Catch-all | `:386-393` writes the adapter `NameServer` (per installation). The `.` rule is appended at `:347-353` into the shared `NetBird-Match-<n>` keys, so the last installation to apply owns every NetBird NRPT rule, not just the catch-all |
| 5.1 Firewall | Confirmed. Also: the surviving rule carries A's `localip`, so B's overlay IP is never allowed |
| 5.1 SSH config | `ssh detect` does not contact a daemon (`client/cmd/ssh.go:860-888`); `ssh proxy` does, through the default address. The executable path written is the writing daemon's own (`manager.go:188,290-303`) |
| 5.1 Fixed ports | Metrics and SOCKS5 failures are only logged; pprof exists only in `pprof`-tagged builds and a bind failure kills the process. SOCKS5 is netstack-mode only |
| 5.2 Daemon pipe | `resolve_pipe_windows.go:39` hard-codes `"netbird"` only in a reachability probe that runs for the default address and never rewrites it; it is not the cause of misrouting. The cause is that the second daemon on the **same** name falls back to the plain pipe, and clients try the protected one first, so the first daemon to start wins. `npipe://netbird_acme` already works end to end without code changes |
| 5.2 Service | Install never reuses an existing service: the kardianos fork returns "service already exists". **Missed:** a second install with a different `-s` still fails because the DisplayName `Netbird` must be unique. kardianos writes the service `Environment` value before checking existence |
| 5.2 State | `NB_STATE_DIR` is read independently in two places (`configs.go:17`, `profilemanager/service.go:70`) |
| 5.2 Logs | Confirmed. `netbird.err` has no override at all |
| 5.2 Per-user files | Missed `<profileID>.state.json` (account e-mail). Per-user *profile configs* are daemon-side under `<StateDir>\<user>\`, not in `%APPDATA%`. The CLI sends the profile from `active_profile.txt` to whichever daemon it talks to |
| 5.2 Debug event | Constant is at `:20`, not `:19` |
| 5.4 NSIS | Missed `App Paths\Netbird-ui`. Interactive "No" on the upgrade prompt installs over the old installation without uninstalling. Only `/S`, `/D=` (overridden) and `/NCRC` are understood today |
| 5.4 MSI service | The `Arguments` lack `--`, so cobra accepts them as positional arguments and ignores them; the service runs on built-in defaults. Harmless today, but any per-installation argument in that style would be dropped |
| 5.4 MSI identity | The fixed Component GUID (`:18`) also has to change per installation |
| 5.4 Updater | It runs only while a UI is subscribed to daemon events (`client/server/event.go:19`, `server.go:2792-2797`). Every daemon deletes the shared `tmp-install` on connect (`client/internal/connect.go:313-316`) |
| 5.3 Single instance | Confirmed. The second process forwards its arguments to the first (unencrypted `WM_COPYDATA`), and NetBird ignores them, so a second UI with another `-daemon-addr` is dropped silently |
| 5.3 Autostart | The fork does support `AutostartOptions.Identifier` and `Arguments`, but it identifies "its" Run value **by executable path**, and `enable` deletes any other value pointing at the same executable. A per-installation identifier isolates installations only when their executable paths differ |
| 5.3 First-run gate | Two earlier checks stop a second installation: the shared `autostartInitialized` flag and the existence of the shared `ui-preferences.json` |
| 5.3 Toast identity | The CLSID is not in Go code; only the MSI pre-seeds it. NSIS shortcuts carry no AUMID, and NSIS installs get a CLSID generated per user. Every UI start rewrites `LocalServer32` to its own path, so the last UI started receives every activation. The practical risk is a session-warning action ("Extend now") running against the wrong daemon |
| 5.3 Quick-actions event | Confirmed; it is a manual-reset event, so one signal releases every UI |
| 5.3 Tray icons | Confirmed. The coloured icons already have a status badge bottom-right, so an overlay goes top-left |
| 5.5 GPO / MDM | Confirmed, with detail: `managementURL` and `preSharedKey` are per network and must not apply to every installation; `wireguardPort` and the local metrics address would collide |
| 5.5 Linux | `NB_NFTABLES_TABLE` and `NB_FWMARK_BASE` exist but are not enough for two daemons: routing table 7120, ip rule priorities, iptables chains, ipset names, nftables accept-rule tags in foreign tables, sysctls, `resolv.conf` and the SSH config have no override. The `netbird@.service` template also shares `RuntimeDirectory=netbird`, which systemd removes when any instance stops |
| 5.5 macOS | The scutil keys are `NetBird-{Search,Match,Local}` (not per interface), so two daemons write the same keys. At daemon start `FlushMarkedRoutes` deletes every route carrying the NetBird route tag, including another daemon's (`client/internal/routemanager/systemops/systemops_unix.go:63-120`) |
| 6.9 wintun | `wintun.dll` is shipped as a separate file, downloaded and checksum-verified by CI (`release.yml:779-791`). The `load_wgnt_from_rsrc` build tag appears unused, and `client/resources.rc` is not referenced by any workflow |
