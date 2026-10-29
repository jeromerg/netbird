# Installer: flow, parameters and lifecycle

Packaging model A (see [`00-overview.md`](00-overview.md#51-d1-packaging-model-recommendation-separate-copies-model-a)):
every named installation is a separate copy with its own program folder, "Apps" entry and
uninstaller. Requirement IDs refer to [`01-requirements.md`](01-requirements.md) (`SBS-INS-*`).
Line numbers refer to `client/installer.nsis` and `client/netbird.wxs` at v0.81.5.

## 1. Scope

| Package | First delivery | Notes |
|---|---|---|
| NSIS (`netbird-installer.exe`) | Default installation unchanged; named installations added | All changes below |
| MSI (`netbird_installer_<ver>_windows_<arch>.msi`) | Default installation only, unchanged | Named installations deferred; design in section 9 |
| Auto-updater | Default installation unchanged; disabled for named installations of the fork | `SBS-UPD-*` |

## 2. Today (v0.81.5)

| Step | NSIS | MSI |
|---|---|---|
| Pages | Welcome, License, Directory, InstFiles, Finish (`:74-82`). No custom install page; the uninstaller has an nsDialogs "delete data" page (`:86,103-124`) | None (no WixUI) |
| Install dir | Forced to `$PROGRAMFILES64\Netbird` in `.onInit` (`:13,171`); `/D=` and `InstallDirRegKey` are overridden. The Directory page can still change it interactively | `ProgramFiles64Folder\Netbird`, not overridable from the command line |
| Existing install | Reads `Uninstall\Netbird` (64-bit view, then 32-bit, `:175-181`). Silent: runs the old uninstaller. Interactive: asks; "No" installs over it without uninstalling (`:184-190`) | `MajorUpgrade` with a fixed `UpgradeCode` (`:4,14`) removes any earlier build |
| Registration | `App Paths\Netbird`, `App Paths\Netbird-ui`, `Uninstall\Netbird`, PATH via EnVar, `$SMPROGRAMS\Netbird.lnk`, `$DESKTOP\Netbird.lnk` (`:16-23,218-234`) | Shortcuts `NetBird` with AUMID `NetBird` and toast CLSID `{0E1B4DE7-E148-432B-9814-544F941826EC}`, `HKCU\Software\Classes\AppUserModelId\NetBird`, PATH (`:21-28,47,55-69`) |
| Service | `netbird service install`, `service start`, default name `Netbird` (`:274-279`) | `ServiceInstall Name="NetBird"` with arguments that cobra ignores as positional (`:33-45`); the service runs on built-in defaults |
| Uninstall | `service stop`, `service uninstall`, `taskkill /im Netbird-ui.exe /f`, optional `RMDir /r %ProgramData%\Netbird`, delete files, shortcuts, registry, AUMID key, PATH (`:282-336`) | Standard MSI removal; `taskkill /F /IM netbird-ui.exe` custom action before `InstallValidate` (`:84-85,115`) |

## 3. The installation record

Every named installation has a record, written by the installer and read by the uninstaller, the
UI and `netbird instance list` (`SBS-ID-07`):

```text
HKLM\SOFTWARE\NetBird\Installations\<name>
    Label        REG_SZ   "ACME"                      name as typed, for display
    Color        REG_SZ   "#E69F00"
    Role         REG_SZ   "secondary"                 or "primary"
    InstallDir   REG_SZ   "C:\Program Files\Netbird_acme"
    Version      REG_SZ   "0.81.5"
```

The default installation has no record. `HKLM\SOFTWARE\NetBird` is not used by v0.81.5, so this key
collides with nothing. The record is readable by every user, so the UI can read its colour, and
writable only by administrators.

## 4. Interactive flow

### 4.1 Pages

The standard path keeps today's pages. The only visible change is one checkbox on the Welcome page.

```text
Welcome  (+ checkbox "Advanced: install an additional, named NetBird")
   |
License
   |
   +-- checkbox off --> Directory --> InstFiles --> Finish          (today's flow, default installation)
   |
   +-- checkbox on  --> Named installation --> Directory --> InstFiles --> Finish
```

### 4.2 Welcome page

```text
+--------------------------------------------------------------------------+
| [banner]  Welcome to NetBird Setup                                        |
|                                                                           |
|           Setup will guide you through the installation of NetBird.      |
|           ...                                                             |
|                                                                           |
|           [ ] Advanced: install an additional, named NetBird              |
|               (to connect to another network at the same time)           |
|                                                                           |
|                                              [ Next > ]   [ Cancel ]     |
+--------------------------------------------------------------------------+
```

The checkbox is added to the MUI2 Welcome page with a show callback and `nsDialogs::CreateControl`.
nsDialogs is already used by the uninstaller, so no new plugin is needed.

### 4.3 Named installation page

```text
+--------------------------------------------------------------------------+
| Named installation                                                        |
| Each named installation is a separate NetBird with its own network,      |
| service and tray icon.                                                    |
|                                                                           |
|  Name        [ ACME                      v ]   1-12 letters or digits,     |
|                                                 starting with a letter    |
|                                                                           |
|  Colour      [ (#) Purple                v ]   [ Custom... ]               |
|                                                                           |
|  [ ] Primary: may use exit nodes and take over system DNS                 |
|      Leave unchecked if another NetBird on this computer is primary.      |
|                                                                           |
|  [x] Start the NetBird tray icon when I sign in                           |
|                                                                           |
|  ! An installation named "acme" exists (0.81.3). It will be upgraded.    |
|                                                                           |
|                                    [ < Back ]  [ Next > ]  [ Cancel ]    |
+--------------------------------------------------------------------------+
```

- **Name.** An editable drop-down listing existing named installations, so picking one upgrades it
  and typing a new name creates one. The status line under the fields updates as the user types:
  "will be created", "will be upgraded", or the validation error.
- **Colour.** A drop-down of the palette (section 4.4) with a swatch drawn by `SetCtlColors`, plus
  "Custom..." which opens the Windows colour dialog through the built-in System plugin
  (`comdlg32::ChooseColor`). The pre-selected entry is the first palette colour that no existing
  record uses.
- **Primary.** Unchecked by default. When it is checked and another primary exists (the default
  installation is installed, or a record has `Role=primary`), the page shows "Another primary
  installation exists: <name>. Two primaries compete for exit nodes and system DNS."
- **Start with Windows.** Sets the initial autostart preference for the user running the
  installer (`SBS-UI-04`).
- **Directory page** that follows proposes `$PROGRAMFILES64\Netbird_<name>`. For an upgrade it is
  skipped, and the recorded `InstallDir` is used.

### 4.4 Colour palette

The Okabe-Ito palette, chosen because its colours stay distinguishable for the common forms of
colour-vision deficiency. Black is dropped because it disappears on a dark taskbar.

The order decides which colour is pre-selected. The tray icons already use colour: the bird is
orange, and the status badges are green (connected), red (error), blue (update) and black
(connecting). Colours that cannot be mistaken for a state or the brand come first; the others stay
available.

| Order | Name (`/COLOR=`) | Hex | Note |
|---|---|---|---|
| 1 | `purple` | `#CC79A7` | |
| 2 | `skyblue` | `#56B4E9` | |
| 3 | `yellow` | `#F0E442` | |
| 4 | `orange` | `#E69F00` | close to the bird; the ring keeps it visible |
| 5 | `blue` | `#0072B2` | close to the update arrow |
| 6 | `green` | `#009E73` | close to the "connected" badge |
| 7 | `vermillion` | `#D55E00` | close to the "error" badge |

Any `#RRGGBB` value is also accepted. The default installation has no colour and its tray icons are
unchanged. The dot is drawn in the top-left corner, the only corner no state uses
(`SBS-COL-02`).

### 4.5 Validation messages

Shown on the named page (interactive) or written to the install log with exit code 2 (silent).
The rules are `SBS-ID-02` and `SBS-ID-03`; the CLI uses the same messages.

| Input | Message |
|---|---|
| empty | `Enter a name for this installation.` |
| `2nd` | `The name must start with a letter.` |
| `acme-office` | `Use only letters and digits (no spaces, hyphens or underscores).` |
| `acmeofficenetwork` | `The name can be at most 12 characters long.` |
| `default`, `netbird` | `"<name>" is reserved. Choose another name.` |
| `#12345` | `Enter a colour as #RRGGBB or pick one from the list.` |

Names are case-insensitive: `ACME` and `acme` are the same installation. The record keeps the label
as typed.

## 5. Unattended installation

### 5.1 NSIS parameters

Parsed in `.onInit` with the built-in `FileFunc` `GetOptions` macro (no new plugin).

| Parameter | Values | Default | Meaning |
|---|---|---|---|
| `/S` | | | Silent, as today |
| `/INSTANCE=<name>` | name rules of `SBS-ID-02` | empty | Target installation. Empty or absent: the default installation, exactly today's behaviour |
| `/COLOR=<c>` | palette name or `#RRGGBB` | first palette colour (section 4.4) that no record uses | Ignored for the default installation |
| `/ROLE=<r>` | `primary`, `secondary` | `secondary` | Ignored for the default installation, which is always primary |
| `/AUTOSTART=<b>` | `0`, `1` | `1` | Initial autostart for the installing user |
| `/D=<dir>` | | | Overridden, as today, for both default and named installations |

Examples:

```bat
:: Default installation, unchanged
netbird-installer.exe /S

:: Named installation for ACME
netbird-installer.exe /S /INSTANCE=acme /COLOR=#E8A33D

:: Upgrade of the named installation (same command with the newer installer)
netbird-installer.exe /S /INSTANCE=acme

:: Silent uninstall of the named installation only
"C:\Program Files\Netbird_acme\netbird_uninstall.exe" /S
```

Exit codes: `0` success, `1` cancelled or failed (NSIS default), `2` invalid parameter. An invalid
parameter never falls back to the default installation (`SBS-INS-03`).

On upgrade, `/COLOR` and `/ROLE` replace the recorded values only when given; otherwise the record is
kept.

### 5.2 MSI properties

Not in the first delivery; see section 9. The default MSI accepts no new property.

## 6. Upgrade, repair, uninstall of one installation

### 6.1 Which installation is the target

| Target | Existing state | Action |
|---|---|---|
| default | no `Uninstall\Netbird` | Fresh install, as today |
| default | `Uninstall\Netbird` exists | As today: silent runs the old uninstaller, interactive asks |
| `acme` | no record and no `Uninstall\Netbird_acme` | Fresh named install |
| `acme` | record and `Uninstall\Netbird_acme` | Upgrade: run **that** installation's uninstaller in upgrade mode (keeps data and record), then install |
| `acme` | only one of the two exists | Repair: treat as upgrade, log the inconsistency, rewrite both |
| any | another installation exists | Leave it alone. Never run another installation's uninstaller, never stop its service, never end its processes (`SBS-INS-04`) |

### 6.2 Upgrade

1. Stop the target service (`netbird service stop --instance acme`).
2. End the target UI processes only (section 6.4).
3. Run the target's previous uninstaller with `/S /UPGRADE _?=<dir>`; `/UPGRADE` keeps the record,
   the state folder and the per-user files.
4. Install files, rewrite registration, `netbird service install --instance acme ...`, `service start`.
5. Restart the UI for the installing user if it was running.

The peer identity is in the state folder, which is kept, so no re-login is needed (`SBS-INS-06`).

### 6.3 Uninstall

The uninstaller finds out which installation it belongs to from the record whose `InstallDir`
equals its own folder. If no record matches and the folder is not `$PROGRAMFILES64\Netbird`, it
stops with an error instead of acting on the default installation (`SBS-INS-05`, fail closed).

Steps for `acme`, in order:

1. `netbird service stop --instance acme`, then `service uninstall --instance acme`. Stopping the
   service runs its own teardown: NRPT keys `NetBird_acme-*`, firewall rules `Netbird_acme*`,
   `99-netbird_acme.conf`, routes, adapter.
2. End the UI processes of this installation only (section 6.4).
3. If "delete data" was ticked: `RMDir /r %ProgramData%\Netbird_acme`. A silent uninstall keeps the
   data, as today.
4. Delete files and `$INSTDIR`.
5. Delete `NetBird (ACME).lnk` from the Start menu and the desktop.
6. Delete `Uninstall\Netbird_acme` and the record (unless `/UPGRADE`).
7. For the user running the uninstaller: the Run value `netbird_acme`, the key
   `HKCU\Software\Classes\AppUserModelId\NetBird_acme`, and (with "delete data")
   `%APPDATA%\netbird_acme`.
8. Remove the wintun adapter if the service teardown left it (`netbird service uninstall
   --instance acme` SHOULD do this, keyed by the derived GUID).

Per-user leftovers of other Windows users (their Run value and AUMID key) cannot be reached from
the uninstaller's session. They point to a deleted program and are harmless; today's uninstaller
has the same limitation.

The default installation's uninstaller keeps today's steps, except that it ends only the default
installation's UI processes (section 6.4).

### 6.4 Ending UI processes

`taskkill /im Netbird-ui.exe /f` (NSIS `:291`, MSI `:84`, updater
`client/internal/updater/installer/installer_run_windows.go:186-216`) ends every installation's UI.
Fork-built installers and uninstallers instead end only processes whose executable path is inside
the target installation's folder (`SBS-INS-07`). With model A every installation has its own folder,
so the path identifies the installation.

An unmodified upstream installer or updater acting on the default installation still ends every
`netbird-ui.exe`. To keep named installations out of that, a named installation installs its UI as
`netbird-ui_<name>.exe` (`SBS-INS-08`, Q6). Nothing in the code depends on the UI's file name: the
autostart entry and shortcuts point to the actual path.

## 7. Registration per installation

| Item | Default installation (unchanged) | Named installation `acme` |
|---|---|---|
| Program folder | `C:\Program Files\Netbird` | `C:\Program Files\Netbird_acme` |
| UI executable | `netbird-ui.exe` | `netbird-ui_acme.exe` |
| "Apps" entry | `Uninstall\Netbird`, DisplayName `Netbird` | `Uninstall\Netbird_acme`, DisplayName `NetBird (ACME)` |
| Start menu, desktop | `Netbird.lnk` -> `netbird-ui.exe` | `NetBird (ACME).lnk` -> `netbird-ui_acme.exe --instance acme`. No AUMID on the shortcut (the default's NSIS shortcuts have none either); the UI registers `NetBird_acme` itself |
| `App Paths` | `Netbird`, `Netbird-ui` | None. `App Paths` keys are executable names typed in Win+R; a second `Netbird` key would hijack the default's |
| `PATH` | Program folder appended | Not changed. One `netbird.exe` on `PATH`, see 7.1 |
| Service | `Netbird` | `Netbird_acme`, DisplayName `NetBird (ACME)` |
| Data | `%ProgramData%\Netbird` | `%ProgramData%\Netbird_acme` |
| Record | none | `HKLM\SOFTWARE\NetBird\Installations\acme` |

### 7.1 The CLI of a named installation

Only the default installation's folder is on `PATH`. A named installation is reached with
`netbird --instance acme <command>` from the default CLI (fork build), or with its own binary by
full path: `"C:\Program Files\Netbird_acme\netbird.exe" --instance acme status`. When the default
installation is an unmodified upstream build, its CLI has no `--instance` flag but can still reach
the named daemon with `--daemon-addr npipe://netbird_acme` (`SBS-CLI-04`).

Adding named folders to `PATH` was rejected: the first `netbird.exe` found would silently decide which
version answers, and the order changes with every install.

## 8. Coexistence with official packages

| Situation | Behaviour |
|---|---|
| Official NSIS run while a fork default installation exists | Upgrades or replaces the default installation (it finds `Uninstall\Netbird`). Named installations keep running. Their UIs survive because their image name differs (`SBS-INS-08`) |
| Official MSI run while a fork NSIS default installation exists | As today between the two official packages (the MSI does not see the NSIS install). Not changed by this feature |
| Fork named installation next to an official default installation | Supported. All named identifiers are outside the official build's cleanup patterns (`SBS-DEF-04`). The official updater kills `netbird-ui.exe` only, and restarts `netbird.exe service start` on the default service only |
| Official default installation auto-updated by its management server | Unaffected by named installations. Its update temp folder `%ProgramData%\Netbird\tmp-install` is not shared with named installations (`SBS-UPD-03`) |

## 9. MSI named installations (deferred design)

Two options were considered.

| | Per-name MSI build | MSI instance transforms |
|---|---|---|
| How | `wix build -d InstanceName=acme ...` produces one MSI per name, with `UpgradeCode = UUIDv5(ns, "msi-upgrade/acme")`, per-name Component GUIDs, product name, folder, service, shortcuts and AUMID | One MSI with N pre-declared instance slots (`<InstanceTransforms>`), installed with `MSINEWINSTANCE=1 TRANSFORMS=:I01 INSTANCENAME=acme` |
| `MajorUpgrade` per installation | Natural: each name has its own UpgradeCode | Requires an UpgradeCode per slot and `TRANSFORMS=:I01` on every upgrade |
| Limits | One build per name | Fixed number of slots; slot and name are separate concepts |
| Intune / GPO deployment | One app per name, the usual model | One app, a transform per assignment |

**Recommendation: per-name MSI build**, generated by a script in the fork. Both the `UpgradeCode`
(`client/netbird.wxs:4`) and the fixed Component GUID (`:18`) must become per name, otherwise two
MSIs share components and the second install or the first uninstall breaks the other. When this is
done, the `ServiceInstall Arguments` must be written with `--` flags: today
`service run config ... log-level info` (`:43`) is accepted by cobra as positional arguments and
ignored, so an `--instance` added in that style would be silently dropped. The shortcut's
`System.AppUserModel.ID` and the pre-seeded `CustomActivator` must equal the UI's `Options.Name`
(`NetBird_<name>`), as the comment at `client/ui/main.go:274-281` requires for the default MSI.

## 10. Building the fork's installer

The upstream release pipeline builds unsigned test installers (`.github/workflows/release.yml:722-870`)
and signs only in the external `netbirdio/sign-pipelines` repository (`:972-984`). The fork needs
its own workflow (`SBS-UPD-05`):

| Step | Tooling (from `release.yml`) |
|---|---|
| Build `netbird.exe` | Go, `CGO_ENABLED=0`, `-X github.com/netbirdio/netbird/version.version=<ver>` (`.goreleaser.yaml:19-41`) |
| Build `netbird-ui.exe` | Go with mingw-w64 (`CGO_ENABLED=1`), Node, pnpm, `wails3` (version from `go.mod`), `-H windowsgui`, tag `production` (`.goreleaser_ui.yaml:30-63`) |
| Version resources | `goversioninfo` v1.7.0 (`release.yml:233-238,486-497`) |
| wintun | `wintun-0.14.1.zip`, checksum-verified, `wintun.dll` copied next to the binaries (`release.yml:779-791`). It is a separate file, not embedded |
| WebView2 bootstrapper | `wails3 generate webview2bootstrapper -dir client` |
| NSIS | NSIS 3 `makensis /V4 /DARCH=amd64`, `APPVER=a.b.c.d`, plugins EnVar and ShellExecAsUser (`release.yml:793-813,837-850`) |

Signing: none in the first delivery. The installer and binaries are unsigned, so SmartScreen warns
once per download; the wintun driver is signed by its vendor and installs regardless. A self-signed
code-signing certificate trusted on the owner's machine is an option (Q14).

Version string: the fork builds with the upstream version it is based on plus build metadata, for
example `0.81.5+sbs.1`. `hashicorp/go-version` ignores metadata when comparing (`Version.Compare`,
checked in the v1.6.0 source in the local module cache; the repository pins v1.7.0), so the updater
and version posture checks see `0.81.5`. A pre-release suffix (`0.81.5-sbs.1`) would sort *below*
`0.81.5` and make a fork-built default installation's updater install the official `0.81.5` over it.
NSIS needs a separate four-part `APPVER` (for example `0.81.5.1`) because `VIProductVersion` accepts
digits only.
