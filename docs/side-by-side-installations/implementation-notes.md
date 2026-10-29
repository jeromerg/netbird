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
