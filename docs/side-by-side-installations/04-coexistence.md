# Coexistence of connected installations

This chapter defines how two or more connected installations behave together on one Windows
machine. Requirement IDs refer to [`01-requirements.md`](01-requirements.md). Code references are
to v0.81.5. Statements marked **UNVERIFIED** depend on Windows, wintun or wgctrl behaviour that is
not visible in this repository; each one has a manual test in [`05-test-plan.md`](05-test-plan.md).

## 1. Principles

1. **No runtime coordination.** Installations never talk to each other and share no lock, file or
   registry value at runtime. Each one decides from its own configuration and from what it can read
   in the operating system (adapters, routes, NRPT keys). This keeps the patch small and means an
   unmodified upstream build can be one of the installations.
2. **Static roles decide conflicts.** Each installation has a role, *primary* or *secondary*
   (`SBS-COEX-01`). The default installation is primary and behaves exactly as today. A named
   installation is secondary unless it is explicitly made primary.
3. **Each installation cleans up only what it owns** (`SBS-DEF-03`, `SBS-ID-05`).
4. **Start order must not matter.** After a reboot both services start in parallel. The result must
   be the same whichever starts first, with the one exception documented in section 6.6.

## 2. Roles

| Behaviour | Primary | Secondary |
|---|---|---|
| Overlay adapter, peers, routed networks | As today | As today, with a higher interface metric (section 4) |
| Exit node / default route (`0.0.0.0/1` + `128.0.0.0/1`, `::/1` + `8000::/1`, v6 blackhole) | As today | Never installed. Auto-applied exit nodes are ignored; selecting one is refused with a clear error |
| Catch-all DNS (NRPT `.` rule, `.local` exemption, adapter `NameServer`) | As today | Never installed. Match domains still apply |
| Overlay prefix overlapping another NetBird adapter | As today (no check) | Refuses to connect, reports the conflict |
| Identical routed prefix on two installations | Wins | Loses (interface metric) and warns |

The owner's setup maps directly: GLOBEX is the default installation (primary), ACME is the named
installation `acme` (secondary). Both are split tunnels with disjoint ranges and no exit node, so no
row of this table changes what happens for them. The roles start to matter only if one network adds
an exit node, a catch-all nameserver group or a route that the other one also has.

**Two primaries** are allowed, because a user may want two full installations with no default one.
Then the conflicts below are back and `netbird status` shows a warning naming the other primary
(`SBS-COEX-02`). **No primary** is fine.

The role is persisted on the service command line (`--instance-role`), written by the installer,
and shown by `netbird status` and `netbird instance list`.

## 3. Adapters and overlay addresses

Each installation has its own wintun adapter, with its own GUID and its own name (`SBS-NET-01`,
`SBS-NET-02`). The name matters as much as the GUID: NetBird finds its own adapter **by name** for
VPN routes (`client/iface/iface.go:114-122`, `client/internal/routemanager/manager.go:163-173`),
for the uplink exclusion (`client/internal/routemanager/systemops/systemops_windows.go:919-928`),
the interface monitor (`client/internal/wg_iface_monitor.go:54,71`), the `netsh ... admin=disable`
on shutdown (`client/iface/iface_destroy_windows.go:14`) and the UAPI pipe
(`client/iface/configurer/uapi_windows.go:10`).

### 3.1 Overlapping overlay prefixes

Management picks a random `/16` out of `100.64.0.0/10` for a new network
(`management/server/types/network.go:20-23,44-48`). Two networks created with the defaults collide
with a probability of about 1 in 64. Self-hosted networks can be configured to anything.

If two overlay prefixes overlap, both adapters get an on-link route for the same destinations, and
the in-memory DNS server of both installations sits on the same address (the last IP of the overlay,
`client/internal/dns/server.go:229-233`). Routing to peers of one of the two networks is then
broken, and so is its DNS. There is no useful degraded mode.

**Rule (`SBS-COEX-04`):** when a secondary installation is about to bring its tunnel up, it checks
the addresses of every other NetBird adapter (as defined in section 5.2). If its overlay prefix overlaps one of
them, it does not connect, and `netbird status` and the UI report which adapter and prefix
conflict. A primary does not check (today's behaviour). When the primary connects after the
secondary, the interface metric (section 4) still makes the primary's on-link route win, so the
primary keeps working and the secondary is the one that breaks; the secondary SHOULD detect this
through its address-change monitoring and report it.

GLOBEX (`10.20.0.0/16`) and ACME (`100.80.0.0/16`) do not overlap.

### 3.2 Overlay prefix overlapping a routed network of another installation

Normal longest-prefix routing applies. Example: GLOBEX routes `100.80.4.0/24` and ACME's
overlay is `100.80.0.0/16`: traffic to `100.80.4.0/24` goes to GLOBEX, and the ACME peers in
that `/24` are unreachable. A secondary SHOULD warn when a route on another NetBird adapter is
more specific than, and inside, its own overlay prefix (`SBS-COEX-05`). It does not refuse,
because the overlap may be intended.

## 4. Routed networks

Windows routes are added with route metric `1` on the installation's own adapter
(`client/internal/routemanager/systemops/systemops_windows.go:233-279`). NetBird never sets an
interface metric. The effective metric of a route is route metric plus interface metric, so two
identical prefixes on two NetBird adapters tie, and Windows picks one (**UNVERIFIED** which).

**Rule (`SBS-COEX-03`):** a secondary installation sets the interface metric of its own adapter to
a fixed value above any automatic metric (proposed: `1000`). The primary leaves its adapter on the
automatic metric, as today. For an identical prefix the primary wins. For different prefix lengths,
longest-prefix routing applies regardless of role; that is intended routing, not a conflict.

The higher metric also covers the on-link overlay route, which Windows creates with the address
(`client/iface/device/device_windows.go`, `assignAddr`).

A secondary SHOULD warn when one of its routes is identical to a route on another NetBird adapter.

## 5. Control traffic and uplink selection

### 5.1 Today

Windows uses "advanced routing" by default (`client/net/env_bound_iface.go:26-49`): instead of
exclusion routes, every NetBird socket is pinned to an uplink with `IP_UNICAST_IF` /
`IPV6_UNICAST_IF` (`client/net/net_windows.go:150-189,253-284`). This covers the management and
signal gRPC connections (`client/grpc/dialer_generic.go:54`), relay
(`shared/relay/client/dialer/ws/ws.go:85`, `shared/relay/client/dialer/quic/quic.go:59`) and ICE
(`client/internal/stdnet/listener.go:13-19`). The uplink is `GetBestInterface(dest, ownName)`, which
skips only the adapter with the installation's own name and ranks the rest by longest prefix, then
route plus interface metric (`client/internal/routemanager/systemops/systemops_windows.go:885-928`).

Consequence with two installations: if A has an exit node (`0.0.0.0/1` + `128.0.0.0/1` on A's
adapter beats the physical `/0`), B's management, signal, relay and WireGuard traffic is pinned into
A's tunnel. B's WireGuard packets are then nested inside A's tunnel, with an MTU problem, and B's
connectivity depends on A. Sockets with an unspecified destination are pinned once at creation
(`client/net/net_windows.go:123-148`), and B's network monitor reacts only to `/0` changes
(`client/internal/networkmonitor/check_change_windows.go:29`), so when A disconnects nothing makes B
re-pin.

### 5.2 Rule

**`SBS-COEX-06`:** an installation never selects any NetBird adapter as an uplink, not only its own.
A NetBird adapter is one whose name starts with `wt`, the convention the ICE blacklist already
relies on (`client/internal/profilemanager/config.go:55-58`, matched by prefix in
`client/internal/stdnet/filter.go:18-22`). The same exclusion applies to the legacy-routing path
(`client/internal/routemanager/systemops/systemops_generic.go:141-147`) and to the capture of
"original" nameservers (section 6.4).

The same rule applies to the network monitor's initial next-hop lookup
(`client/internal/networkmonitor/monitor.go:57-58`). Without it, an installation that starts while
another one's exit node is up records that adapter as its uplink, and from then on every harmless
`/0` change on the physical adapter (a DHCP renewal) restarts its engine, while the removal of the
`/1` routes is never noticed.

Named installations always use socket pinning; `NB_USE_LEGACY_ROUTING` is ignored for them
(`SBS-NET-07`). Legacy routing adds `/32` host routes to management, signal, relay and STUN on the
physical adapter, and two installations reaching the same relay or STUN address would share a route
row that the first one to stop deletes.

On a machine without named installations there is no other `wt*` adapter, so the default
installation's behaviour does not change. An **unmodified upstream** default installation does not
have this rule, so it would still use a named adapter as an uplink for destinations the named
installation routes. Because a secondary never installs a default route, that only happens for
destinations inside the secondary's routed networks, where it is the correct route anyway.

Trade-off: when A is a full tunnel (an exit node that carries all internet traffic), B's control
traffic leaves through the physical uplink rather than through A. NetBird already does the same for
A's own control traffic, so this is consistent. The owner's networks are split tunnels, so the
trade-off does not arise for them (Q11 in [`99-open-questions.md`](99-open-questions.md)).

## 6. DNS

Windows NetBird DNS has no socket to collide on: the resolver is an in-memory packet hook on the
overlay's last IP, port 53 (`client/internal/dns/service_memory.go:29-39`). The DNS forwarder binds
`overlayIP:22054` (`client/internal/dns/dns.go:22-25`), which only collides when overlay IPs are
equal, and that is already refused (section 3.1).

### 6.1 NRPT rules

Today both installations would write the same keys (`NetBird-Match-<n>`, `NetBird-ExemptLocal`,
`host_windows.go:48,67-68,408-409`), and each apply first deletes every key whose name starts with
`netbird-`, case-insensitively, in the local and the GPO store (`host_windows.go:313,603,628-651,749`).
Whoever applies last owns all of NetBird's NRPT rules.

**Rule (`SBS-DNS-01`):** a named installation writes and deletes only keys that start with
`NetBird_<name>-` (for example `NetBird_acme-Match-0`, `NetBird_acme-ExemptLocal`), matched
case-insensitively. The default installation's prefix `NetBird-` cannot match those keys, because
the ninth character is `_`, not `-`. A named prefix cannot match another name's keys, because names
contain no `-` (`SBS-ID-02`).

### 6.2 Catch-all

Only a primary writes the `.` rule, the `.local` exemption and the adapter `NameServer`. A secondary
whose management configuration asks for a catch-all nameserver group (no match domains, "route all")
keeps its match domains and drops the catch-all, logging once that it did so (`SBS-DNS-03`). For
the owner: GLOBEX configures no nameserver group (checked on 2026-10-29), so this rule changes
something only if ACME configures a catch-all group, which is still to be confirmed (Q4). ACME's
match domains, for example `*.netbird.selfhosted` and ACME's private zones, apply either way.

### 6.3 Same match domain in two installations

If both installations claim the same namespace (for example both use the NetBird default
`netbird.selfhosted`, or both route `corp.example`), Windows has two NRPT rules for the same
namespace and picks one (**UNVERIFIED** which; NRPT gives precedence to the most specific namespace,
the tie-break for identical ones is undocumented).

**Rule (`SBS-DNS-04`):** a secondary SHOULD read the NRPT keys of the other installations (keys
starting with `NetBird-` or `NetBird_` but not its own prefix) before applying, and skip and warn for
any namespace another installation already claims. This works when the primary applies first; if
the primary applies after the secondary, the tie remains, and the secondary SHOULD re-check on its
next apply. Same-management-server setups (section 9) hit this every time, which is one reason they
are discouraged.

In the proof of concept, `*.globex.internal` and `*.netbird.selfhosted` were each resolved by
their own network, so the owner's two networks do not share a peer domain. `netbird.selfhosted` is
the usual peer domain of self-hosted deployments, so two self-hosted networks will often collide on
it.

### 6.4 Original nameservers

At start, each installation records the DNS servers of every adapter except its own as fallback
upstreams (`host_windows.go:153-215`). A route-all installation writes its DNS IP into its own
adapter's `NameServer` (`host_windows.go:386-393`), and `restoreHostDNS` does not remove that value
(`host_windows.go:599-619`). So B can record A's overlay DNS IP as a fallback, and that entry can
outlive A's session.

**Rule (`SBS-DNS-05`):** the capture skips every NetBird adapter, using the same `wt` name rule as
section 5.2 (resolved from the interface GUID through the adapter list).

### 6.5 Search domains

Search domains are written per adapter (`SearchList` under the adapter's GUID key,
`host_windows.go:548-556`), so both installations' search lists apply. This is acceptable: a short
name is tried with each suffix in turn. No rule.

### 6.6 Group-policy NRPT store

Each installation decides at start whether to write into the group-policy store
(`useGPOPolicyStore`, `host_windows.go:665-682`) and never re-evaluates. With per-installation
prefixes, one installation no longer deletes the other's rules, so the store cannot be emptied and
deleted from under an installation that still has rules in it (the path the seed and the review
found). One race remains, and it already exists between NetBird and other software today: if the
store is empty at the instant the secondary starts, the secondary deletes it and uses the local
store, and the primary then recreates the group-policy store with its rules, which hides the
secondary's local rules. This needs a crash leftover plus a start at the same moment. It is
documented, covered by a manual test, and not fixed in the first delivery (Q12).

## 7. Firewall

The Windows firewall integration is one inbound allow rule per address family, created with `netsh`
and matched by name (`client/firewall/uspfilter/interface_allower_windows.go:20,41-86`). Today the
second installation finds a rule named `Netbird`, skips creating its own, and is never allowed
inbound on its own overlay IP; whichever stops first deletes the rule for both.

**Rule (`SBS-FW-01`):** a named installation uses `Netbird_<name>` and `Netbird_<name>-v6`. Because
there is no crash state for this rule and an existing rule is reused, a named installation MUST
check that an existing rule with its name carries its current overlay IP, and recreate it if not
(`SBS-FW-02`). The default installation's behaviour is not changed.

The userspace packet filter (`uspfilter`) is per process and per adapter. There is no WFP code at
v0.81.5. No other firewall state is shared.

## 8. SSH

| Part | Behaviour with two installations | Rule |
|---|---|---|
| NetBird SSH server, port 22022 | Binds the installation's own overlay v4 and v6 addresses (`client/internal/engine_ssh.go:277-295`). No collision unless overlay IPs are equal (refused, section 3.1) | None |
| Client config `%ProgramData%\ssh\ssh_config.d\99-netbird.conf` | Shared path; last writer wins, and either installation's cleanup deletes it (`client/ssh/config/manager.go:118-124,240-275`, `client/internal/engine_ssh.go:222-234`) | `SBS-SSH-01`: one file per installation, `99-netbird_<name>.conf` |
| `ProxyCommand <exe> ssh proxy %h %p` | Uses the default daemon address (`client/cmd/ssh.go:834`) | `SBS-SSH-02`: named installations write `<exe> --instance <name> ssh proxy %h %p` |
| `Match host ... exec "<exe> ssh detect %h %p"` | `ssh detect` only probes host and port, it does not contact a daemon (`client/cmd/ssh.go:860-888`) | Keep, add `--instance` for symmetry |

OpenSSH reads the included files in lexical order and the first value obtained for a parameter
wins. `99-netbird.conf` sorts before `99-netbird_<name>.conf` (`.` is 0x2E, `_` is 0x5F), so if
the same host string appears in both files, the default installation's block wins. The host lists
contain each installation's own peers' IPs, FQDNs and short names; a short name present in both
networks therefore goes to the default installation. Using FQDNs avoids that.

Whether Windows OpenSSH reads `ssh_config.d` without an `Include` line is **UNVERIFIED**; the
default installation depends on the same behaviour today.

## 9. Same management server in two installations

Allowed. Each installation registers its own peer with its own WireGuard key, so the management
server sees two peers. Consequences: both installations receive the same routes (the primary wins
identical prefixes, section 4) and the same DNS match domains (section 6.3). Not recommended, not
blocked. `netbird instance list` shows each installation's management URL, so the user can see it.

**Copying a profile or state file from one installation to another is not supported**
(`SBS-PROF-02`): two daemons with the same WireGuard key appear to management as one peer whose
endpoint flaps, and `netbird profile remove` in either one deregisters the peer that both use. The
supported way to move a network from the default installation to a named one is to log in fresh in
the named installation and then remove the old profile from the default installation; see Q9.

## 10. Other shared resources

| Resource | Behaviour | Rule |
|---|---|---|
| WireGuard UDP port | Default 51820; a busy port falls back to a random free one (`client/internal/connect.go:729-736,827-851`) | A named installation defaults to a derived port in 51821-51920 (`SBS-NET-03`), so the default installation keeps 51820 whatever the start order |
| UPnP / PCP mappings | Description `"NetBird"` (`client/internal/portforward/manager.go:21`), keyed by port | None: ports differ |
| Rosenpass, lazy-connection listeners, relay, STUN | Ephemeral ports | None |
| Local metrics `127.0.0.1:9191`, netstack SOCKS5 `1080`, pprof `6969` | Opt-in. Metrics and SOCKS5 log a bind failure; pprof calls `log.Fatalf` (`client/cmd/pprof.go:29-31`) | `SBS-NET-05`: a named installation defaults to `9191 + k`, `1080 + k`, `6969 + k` |
| Network monitor | Reacts only to `/0` changes on Windows, and exit-node routes are `/1`, so one installation's routes never restart another. Its initial next-hop lookup can pick another installation's adapter | `SBS-COEX-06`: the lookup skips NetBird adapters (section 5.2) |
| Machine-wide DNS calls | `ipconfig /registerdns`, cache flush, `RefreshPolicyEx` on every apply (`client/internal/dns/host_windows.go:509-541,780`) | None: idempotent |
| Wintun driver | One kernel driver for all adapters, versioned by wintun | None (UNVERIFIED for mixed wintun versions under model A; test) |
| Wintun tunnel type `"WireGuard"` | Shared with WireGuard for Windows | None |

## 11. Stop, crash and reboot

| Event in installation B | Effect on A required |
|---|---|
| `netbird down` | None: B removes only `NetBird_acme-*` NRPT keys, rule `Netbird_acme*`, `99-netbird_acme.conf`, its routes and its adapter |
| Service stop, restart | Same |
| Process killed | None. B's next start replays B's own `state.json` from B's state directory only |
| Reboot | NRPT keys are volatile and vanish (`host_windows.go:476`). Both services start in parallel; the result does not depend on the order |
| Uninstall | See [`03-installer-ux.md`](03-installer-ux.md#63-uninstall) |
