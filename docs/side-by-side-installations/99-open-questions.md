# Open questions

Decisions this specification could not take alone. Questions marked **blocking** had to be
answered before any code is written, because they change identifiers or packaging.

**Status (2026-10-29):** the owner accepted every recommended answer. Q11 asked about a case that
does not apply: both networks are split tunnels. One check is left, in Q4: whether LFBB configures
a catch-all nameserver group.

| # | Question | Recommended | Blocking | Decision |
|---|---|---|---|---|
| Q1 | Packaging: separate copies (A) or one program with instances (B)? | A | yes | accepted |
| Q2 | Name placement: `Netbird_lfbb` or literal prefix `lfbb-Netbird`? | `Netbird_lfbb` | yes | accepted |
| Q3 | Name syntax: letters and digits only, 12 characters? | yes | yes | accepted |
| Q4 | Roles: default primary, named secondary, overridable? | yes | yes | accepted |
| Q5 | Changing the colour after install | installer re-run only | no | accepted |
| Q6 | Named UI executable `netbird-ui_<name>.exe`? | yes | no | accepted |
| Q7 | MSI named installations | defer | no | accepted |
| Q8 | Renaming an installation | not supported | no | accepted |
| Q9 | Moving a profile between installations | not supported, re-login | no | accepted |
| Q10 | AIRSECOMA on the official package, LFBB on the fork? | yes, during the fork period | no | accepted |
| Q11 | Control traffic when another network is a full tunnel | accept | no | rule kept; does not arise (split tunnels) |
| Q12 | Group-policy NRPT store race | accept, document | no | accepted |
| Q13 | Group policy and MDM scope | see below | yes | accepted |
| Q14 | Code signing of the fork's installer | unsigned | no | accepted |
| Q15 | Updates of named installations from the fork | manual | no | accepted |
| Q16 | Interface metric `1000` for secondaries | yes, confirm on a VM | no | accepted |
| Q17 | Overlapping overlays: refuse or warn | refuse in a secondary | no | accepted |
| Q18 | When to engage upstream | after the fork works for you | no | accepted |

---

## Q1. Packaging model (blocking)

**Recommended: A, separate copies**, on top of a runtime `--instance` concept that would also serve B.

Your seed leaned towards B. The code shows why A fits better here: each management server pushes its
own auto-update target, and the updater only ever upgrades
(`client/internal/engine.go:982-998`, `client/internal/updater/manager.go:542-562`). With one binary
set, whichever organisation asks for the newest version upgrades the other organisation's client,
and a forced update restarts both networks. A also keeps the standard Windows lifecycle (one "Apps"
entry per network) and lets AIRSECOMA stay on the official package (Q10). Per-installation autostart
also works only with distinct executable paths: the Wails fork finds its Run value by executable
path and deletes other values pointing at the same file. The cost is about 67 MB of disk per
installation and a deferred MSI story.

*If you choose B:* the runtime requirements stay the same; the installer gains an instance
maintenance mode, uninstall removes every instance, and the updater must restart every service and
UI. `03-installer-ux.md` would be rewritten.

## Q2. Where the name goes (blocking)

**Recommended: right after the product token, with an underscore** (`Netbird_lfbb`,
`NetBird_lfbb-Match-0`, `99-netbird_lfbb.conf`, `io.netbird.ui.lfbb`).

You asked for a prefix. A literal prefix cannot be applied to identifiers whose leading part is
fixed (`99-` for SSH include order, `Global\`, `ProtectedPrefix\Administrators\`, the `wt` adapter
prefix, reverse-DNS IDs), and it scatters NetBird entries across sorted lists. The underscore is
the one separator that no v0.81.5 identifier uses after the product token, which is what proves
that an unmodified upstream default installation can never delete a named installation's NRPT
rules (`NetBird-` cleanup is a case-insensitive prefix match,
`client/internal/dns/host_windows.go:749`).

## Q3. Name syntax (blocking)

**Recommended: `^[a-z][a-z0-9]{0,11}$` after lowercasing; the label keeps your casing** (`LFBB`).

No `-` or `_` means one name can never be a prefix-match of another (`a` vs `a-b`), and no name can
fake a suffix: with hyphens allowed, the name `a-v6` would get the firewall rule `Netbird_a-v6`,
which is the IPv6 rule of the name `a`. Twelve characters keep `wt_<name>` within Linux's
15-character interface limit, should Linux follow.

*If you need hyphens* (`lfb-booking`): every identifier needs a terminator that is not in the
charset, and the firewall `-v6` suffix must change for named installations. Possible, but more rules
to test.

## Q4. Primary and secondary (blocking)

**Recommended: the default installation is always primary; a named installation is secondary
unless installed with "Primary" checked.** A secondary never takes the default route or system DNS,
and loses identical routes (`04-coexistence.md`).

This matches AIRSECOMA (default, primary) and LFBB (named, secondary) without any coordination
between installations. A numeric rank is not needed for two installations; it can be added later
without changing identifiers.

*Answer (2026-10-29):* accepted. Neither network uses an exit node; each carries only the traffic
of its own range. AIRSECOMA configures no nameserver group (`netbird status -d` lists none).

*Still to check for LFBB:* with the LFBB profile connected, run `netbird status -d`. A line under
`Nameservers:` that ends in `for [.] is Available` is a catch-all group. LFBB as a secondary would
drop it and keep only its match domains. If LFBB relies on it, say so before step 3 of the delivery.

## Q5. Changing the colour

**Recommended: re-run the installer with the same name and another colour** (or `/COLOR=`). No colour
setting in the UI or CLI in the first delivery.

The colour is stored once per machine in the installation record. A per-user override in the UI
would need a second source of truth for a cosmetic setting.

## Q6. File name of a named installation's UI

**Recommended: install it as `netbird-ui_<name>.exe`.**

The official installer, MSI and updater end every `netbird-ui.exe` process
(`client/installer.nsis:291`, `client/netbird.wxs:84-85`,
`client/internal/updater/installer/installer_run_windows.go:186-216`). With a distinct file name,
an official upgrade of AIRSECOMA does not close LFBB's tray icon. Nothing in the code depends on the
file name.

*If you prefer identical file names:* LFBB's tray icon disappears on every official upgrade of the
default installation and has to be started again by hand.

## Q7. MSI

**Recommended: defer.** The MSI keeps building the default installation unchanged. When needed,
generate one MSI per name (per-name UpgradeCode and Component GUIDs, `03-installer-ux.md` section 9)
rather than MSI instance transforms.

## Q8. Renaming

**Recommended: not supported.** Uninstall, reinstall with the new name, log in again. The name is in
every identifier, including the adapter GUID; a rename would touch all of them.

## Q9. Moving a profile between installations

**Recommended: not supported in the first delivery.** Procedure for LFBB if it is a profile of
your default installation today:

1. Install the named installation `lfbb` and log in to LFBB there (SSO or a setup key). This creates
   a new peer with a new overlay IP.
2. In the default installation, `netbird profile remove <lfbb-profile>`. This deregisters the old
   peer, which is what you want.
3. Ask LFBB's admin to move group memberships or policies from the old peer to the new one, if
   they are not assigned automatically.

*Alternative:* a `profile export` / `profile import` pair that moves the profile file and deletes the
source without contacting management. It keeps the peer and IP, but it is new CLI surface and
moves a private key between folders; worth it only if re-registration is expensive for LFBB.

## Q10. Which build runs AIRSECOMA

**Recommended: keep AIRSECOMA on the official signed package during the fork period, and install
only LFBB from the fork.** AIRSECOMA keeps upstream's auto-updates and support; only LFBB depends on
your fork. Test `T-WIN-08` covers this combination.

*Trade-off:* the official default installation lacks the new uplink exclusion. That matters only
if LFBB routes networks that contain AIRSECOMA's management, signal or relay addresses, which is
unlikely with `100.125.0.0/16`.

## Q11. Control traffic when another network is a full tunnel

A *full tunnel* is a VPN that carries all internet traffic, which NetBird does with an exit node
(the default route through a peer). A *split tunnel* carries only the traffic of its own ranges.

**Rule kept:** every installation keeps its management, signal, relay and WireGuard traffic off
*every* NetBird adapter (`SBS-COEX-06`). If one network were a full tunnel, the other network's
control traffic would leave through the physical network, not through the first tunnel. NetBird
already does this for its own control traffic. The alternative (one tunnel nested in the other)
causes an MTU problem and makes the inner network lose its connection whenever the outer one
reconnects, because nothing re-pins the sockets
(`client/internal/networkmonitor/check_change_windows.go:29`).

*Answer (2026-10-29):* the case does not apply. AIRSECOMA and LFBB are equivalent split tunnels,
each carrying only the traffic of its own range, so neither one's traffic passes through the other.
The rule costs nothing in this setup and still protects it if one network later adds an exit node
or a route that contains the other's servers. No longer blocking.

## Q12. Group-policy NRPT store race

**Recommended: accept and document** (`04-coexistence.md` section 6.6). It needs a crash leftover in
the group-policy store plus both services starting at the same moment, and it already exists
between NetBird and other software. Fixing it means changing the default installation's DNS code,
which this design otherwise leaves alone.

## Q13. Group policy and MDM (blocking)

**Recommended:**

1. Machine posture values in `HKLM\Software\Policies\NetBird` (block inbound, disable routes, SSH,
   profiles, update settings, ...) apply to **every** installation (`SBS-POL-01`).
2. Network identity values (`managementURL`, `preSharedKey`) and colliding values (`wireguardPort`,
   local metrics) apply to the **default installation only** (`SBS-POL-02`).
3. A named installation also reads `HKLM\Software\Policies\NetBird\Installations\<name>`, which
   overrides the global values for it (`SBS-POL-03`). v0.81.5 reads only the values of the global key,
   so it ignores this subkey.
4. A new global value `disableNamedInstallations` lets an organisation forbid named installations
   (`SBS-POL-04`).

The first point is the security one: if named installations ignored machine policy, adding one would
bypass every restriction an administrator set. The fourth adds a public policy value (ADMX), which
upstream would have to agree to.

On this machine `HKLM\Software\Policies\NetBird` does not exist today (checked read-only), so
none of this changes anything for you now. It matters if AIRSECOMA's IT later manages NetBird by
policy on your laptop: point 2 then keeps their `managementURL` away from LFBB.

## Q14. Code signing

**Recommended: unsigned for the first delivery.** SmartScreen warns once per downloaded installer;
the service runs unsigned without issue, and the wintun driver is signed by its vendor. A
self-signed code-signing certificate trusted on your machine removes the warning for you only; a
commercial certificate is the only way for other users.

## Q15. Updates of named installations

**Recommended: manual.** A named installation built from the fork refuses to auto-install (the
official installer would upgrade the default installation instead) and may only show that a newer
upstream version exists. You update by building the fork's next tag and re-running its installer
with `/INSTANCE=lfbb`. An instance-aware updater is a later step (`SBS-UPD-04`).

## Q16. Interface metric of a secondary

**Recommended: `1000`**, to be confirmed on a VM: Windows' automatic metrics for an adapter that
reports a high link speed are far lower, and NetBird never sets one today. The value only decides
identical-prefix ties.

## Q17. Overlapping overlay ranges

**Recommended: a secondary refuses to connect and says why; the primary is not changed.** With
identical or overlapping overlays one of the two networks is broken anyway (routes and the DNS
address collide). Refusing makes the failure obvious. Fix: ask one network's admin to change its
range (account setting `network_range`).

## Q18. Upstream

**Recommended: once `T-WIN-02` and `T-WIN-08` pass on your machine, post a short design summary on
[#446](https://github.com/netbirdio/netbird/issues/446)** proposing step 1 (the identity package,
no behaviour change) as the first PR. `AGENTS.md` requires the design to be agreed in the ticket
before a PR for these areas. Nothing is posted until you decide.
