# Architecture

Roadmap and usage live in [README.md](README.md); visual map in [docs/blueprint.html](docs/blueprint.html).

## Stack

| | |
|---|---|
| Language | Go 1.27.1, module `github.com/nixiz0/netforge` |
| Dependencies | Standard library, plus `golang.org/x/sys` (official Go extension) for the Linux TUN `ioctl`. Add a dependency only when a stage truly needs it (next candidate: crypto). |
| Build / run | `go run ./cmd/netforge <command>` or `go build ./cmd/netforge` (binary ignored by git). Single binary, no config files, no flags yet (positional args only) |
| Checks | `gofmt -l .` and `go vet ./...` must be clean; `go test ./...` must pass. Tests exist for pure logic (binary formats); network behavior is still checked by hand. |
| Dev platform | Windows 11 + PowerShell. Use `curl.exe` (not the `curl` alias) and `netstat -ano \| findstr <port>`. TUN (step 6+) is Linux only: develop it in WSL2 Ubuntu (`/mnt/c/...` path, build with `CGO_ENABLED=0 go build -o /tmp/netforge ./cmd/netforge` (pure Go: no C toolchain or libc headers needed, static binary), run with `sudo`). If the build fails with "error obtaining VCS status: exit status 128", git in WSL refuses the Windows-owned repo: run `git config --global --add safe.directory <repo path>` once, or build with `-buildvcs=false`. The VPN server will run on a Linux VPS. |

## Layout

Standard Go layout: `cmd/` holds the binary, `internal/` holds one package per feature (importable only from this module). Docs stay at the root and in `docs/`.

| Path | Role | Entry point |
|---|---|---|
| `cmd/netforge/main.go` | Subcommand router on `os.Args[1]`, `usage()` exits 2 | `main()` |
| `internal/echo/` | Echo server on `127.0.0.1:9000`: TCP (`echo.go`) and UDP (`udp.go`, same port number, separate port space) | `echo.Run(addr)`, `echo.RunUDP(addr)` |
| `internal/dial/` | Interactive clients: TCP (`dial.go`, stdin ⇄ conn) and UDP (`udp.go`, one line = one datagram) | `dial.Run(addr)`, `dial.RunUDP(addr)` |
| `internal/proxy/` | HTTP forward proxy on `proxy.Addr` (`127.0.0.1:8080`): plain HTTP + `CONNECT` | `proxy.Run()` |
| `internal/socks/` | SOCKS5 proxy on `socks.Addr` (`127.0.0.1:1080`): no auth, TCP `CONNECT` | `socks.Run()` |
| `internal/tunnel/` | Point-to-point UDP tunnel: packet format `VER TYPE LENGTH PAYLOAD` with tests (`packet.go`), server endpoint on `tunnel.ServerAddr` (`127.0.0.1:7000/udp`, `server.go`), client entrance on `tunnel.ClientAddr` (`127.0.0.1:5300/udp`, `client.go`), peer state and keepalive timing (`state.go`), IP packets between a TUN interface and the other end of the VPN (`vpn.go`), AEAD sealing of VPN packets (`crypto.go`), the replay window (`replay.go`), X25519 key files (`keys.go`) and the handshake (`handshake.go`), all tested | `tunnel.RunServer(target)`, `tunnel.RunClient(server)`, `tunnel.RunTUN(dev, listen, server, priv, peerKey)`, `tunnel.GenerateKey()`, `tunnel.PublicKey(path)`, `tunnel.ReadPrivateKey(path)`, `tunnel.ReadPublicKey(path)` |
| `internal/tun/` | TUN interface `nf0` (Linux): `tun_linux.go` creates it via `/dev/net/tun` + `TUNSETIFF`, `tun_other.go` returns an error elsewhere, `ipv4.go` decodes IPv4 headers, `icmp.go` computes Internet checksums and builds ping replies (both tested, build everywhere), `tun.go` logs each packet and writes ping replies back | `tun.Run()`, `tun.Open(name)` |
| `internal/dns/` | Tiny authoritative DNS server over UDP: `message.go` parses a one-question query and builds the answer (A record with a compression pointer to the question, NODATA, NXDOMAIN; tested), `server.go` holds the lab zone (`host.lab`, `vpn.lab`) and the `ReadFrom`/`WriteTo` loop. Default `dns.Addr` = `127.0.0.1:5353/udp`; the lab runs it on `:53` in `nfpeer` | `dns.Run(addr)` |
| `internal/relay/` | Shared two-way copy between a client and an upstream connection, with half-close | `relay.Pipe(client, fromClient, upstream)` |
| `internal/resolve/` | Explicit DNS lookup (timed, loggable via `Lookup.String()`), then dial each address IPv4 first | `resolve.Dial(target, timeout)` |
| `scripts/lab.sh` | Linux test lab, 3 namespaces: client (root, 192.168.100.1), VPN server `nfpeer` (192.168.100.2 / 203.0.113.1), host `nfnet` (203.0.113.10). Subcommands: `up`, `tun` (client nf0 10.10.0.1, server nf0 10.10.0.2), `route` (split tunnel + return route), `forward` (server IP forwarding), `nat` (drop the host's return route, MASQUERADE the VPN subnet on the server; needs `iptables`), `full` / `full-off` (client routes 0.0.0.0/1 + 128.0.0.0/1 into nf0; WSL has no internet meanwhile), `dns` (route to each `/etc/resolv.conf` nameserver: VPN if via nf0, else LEAK), `down`. LF line endings enforced by `.gitattributes` | `sudo bash scripts/lab.sh up` |
| `scripts/replay.py` | Lab attack (step 8.2): records the next client → server VPN packet on veth0 and sends it again from another port | `sudo python3 scripts/replay.py` |
| `docs/blueprint.html` | One-screen live network map: scenarios, animated packets, Go line + concept per packet | open in a browser |
| `_networking-lessons/` | Networking theory, one Markdown file per topic. The `_` prefix sorts it first and makes the Go tool ignore it | read on GitHub |

Dependency rule: `cmd/netforge` imports the feature packages; feature packages may import shared helper packages (`internal/relay`, `internal/resolve`) but never each other. When a helper is needed by two features, move it to its own `internal/<name>` package.

## Runtime model

- **One process = one subcommand.** Servers block forever in an `Accept` loop.
- **Packages never exit the program.** Every `Run` returns an `error` (listen or dial failure); only `cmd/netforge/main.go` calls `log.Fatal` / `os.Exit`.
- **One goroutine per accepted connection.** Per-connection errors are logged and the connection dropped; the server keeps running.
- **UDP servers have no connections:** one socket (`net.ListenPacket`), one `ReadFrom`/`WriteTo` loop, no goroutine per client. Each datagram carries its sender address, and each reply names its destination. UDP clients decide themselves when to stop waiting: `dial.RunUDP` sets a 2 s read deadline once stdin ends, and treats ICMP errors (reported on a later `Read`) as non-fatal. A `+N` line sends an N-byte probe: 1472 bytes fit one 1500-byte Ethernet frame, above 65507 `Write` fails (max IPv4 packet).
- **Relay pattern** (`relay.Pipe`, and the stdin/stdout variant in `dial.Run`): two `io.Copy`, one per direction. The side that receives the *last* event (the peer's FIN) runs on the calling goroutine; the other runs in a goroutine and is abandoned on return. When `main` returns, all goroutines die. Nothing waits on goroutines explicitly yet.
- **Half-close propagation:** when one side sends FIN, forward it with `CloseWrite()` (if the connection supports it), never `Close()`, so in-flight bytes in the other direction are not lost. Full close happens via `defer` on both connections when the function returns.
- **Outbound dials:** the proxies use `resolve.Dial(target, 5*time.Second)`: the DNS lookup is explicit and logged (`resolved example.com → … in 12ms`), then each address is tried with a 5 s timeout. `dial` still uses plain `net.DialTimeout`. No read/write/idle deadlines yet.
- **Binding:** everything listens on loopback (`127.0.0.1`) by design. Exposing on `0.0.0.0` needs auth first.

## Data path: buffers and relay

Every proxy-like feature (HTTP proxy, SOCKS5, later the tunnel) has the same two phases on each connection:

```
            phase 1: parse the protocol header               phase 2: relay, blind
client ──▶ [kernel recv buffer] ──▶ [bufio.Reader br] ──▶ parser     relay.Pipe(client, br, upstream)
                (OS, TCP)             (our memory, 4 KB)    readHead / greet / readRequest
                                            │
                                            └── may hold bytes PAST the header (body, TLS ClientHello)
                                                → phase 2 must read through br, never the raw conn

relay.Pipe:   client ── io.Copy (goroutine) ──▶ upstream    + CloseWrite on client FIN
              client ◀── io.Copy (caller) ───── upstream    returns on upstream FIN
```

- **Kernel buffer:** TCP data waiting for a `Read`. Not ours to manage.
- **bufio buffer:** used only where a protocol needs delimiter framing (`ReadString('\n')`, HTTP). Binary protocols with exact sizes (`io.ReadFull`) can read the conn directly.
- **Relay:** once the target is known, the feature stops understanding bytes and only copies them. One shared implementation: `internal/relay`.

## Proxy internals (`internal/proxy`)

1. `readHead(br)` reads lines with `bufio.Reader.ReadString('\n')` until the empty line (`\r\n`, bare `\n` tolerated).
2. Request line split into `method target version`.
3. `CONNECT host:port` → dial → reply `200 Connection Established` → blind relay (TLS stays end to end).
4. Otherwise the target must be an absolute URL (`http://host[:port]/path`); default port 80; request line rewritten to origin form (`GET /path HTTP/1.1`); head re-sent; relay.
5. Dial failure → `502 Bad Gateway`.

**Invariant:** after `readHead`, always read the client through `br`, never the raw `net.Conn`: bufio may already hold body or TLS bytes.

## SOCKS5 internals (`internal/socks`)

1. `greet`: `VER NMETHODS METHODS…` read with `io.ReadFull`; reply `05 00` if "no auth" (0x00) is offered, else `05 FF` and close.
2. `readRequest`: `VER CMD RSV ATYP ADDR PORT`; only `CMD=01` (CONNECT); address by type (IPv4 4 bytes, domain length-prefixed, IPv6 16 bytes); port big-endian. The log says who resolved DNS: `ATYP=03` domain → the proxy (`net.DialTimeout` resolves it), IP → the client did, outside the proxy (DNS leak).
3. Dial the target (5 s). Reply `05 REP 00 ATYP BND.ADDR BND.PORT` via `reply()`: `00` success with the proxy's local end of the upstream connection, `04` host unreachable, `01` bad request.
4. `relay.Pipe(client, client, upstream)`: no bufio anywhere in SOCKS5, so the raw conn is safe to read.

## Tunnel internals (`internal/tunnel`)

- Server: outer UDP socket on `ServerAddr`, inner connected socket to the target. Client: the mirror image, inner UDP socket on `ClientAddr` (apps send here), outer connected socket to the server. Each endpoint runs two loops, one per direction: one side `encode`s, the other `decode`s.
- Tunnel state (`state.go`): a `peer` type (address + last seen, behind a mutex) with `seen` (returns true for a new or roamed peer), `get` and `expire`. The server tracks the client; the client tracks whether the server answers, and separately remembers the local app. One peer and one app only (point-to-point).
- Keepalives: the client sends a header-only packet (type `02`) at start and every 5 s; the server answers each one. A side silent for 15 s (3 intervals) is expired: the client logs "tunnel down", the server forgets the peer. Any packet brings it back ("tunnel up" / "peer joined").
- Malformed packets are dropped and logged; nothing reaches the target.
- VPN mode (`vpn.go`, `RunTUN(dev, listen, server, priv, peerKey)`): TUN → `seal` → UDP; UDP → `open` → write into the TUN. Client mode (server set) handshakes with the fixed server, then sends keepalives (a new handshake instead whenever there is no session); server mode (server empty) learns the client's address from its packets with the `peer` state, answers keepalives and forgets a silent client, so clients can be behind NAT and roam. `dev` is an `io.ReadWriter`: `cmd/netforge` opens it with `tun.Open` (`vpn` helper), so `tunnel` never imports `tun`.
- VPN encryption (`crypto.go`): every VPN packet, keepalives included, is `VER TYPE LENGTH | NONCE 12 | ciphertext | TAG 16`, sealed with AES-256-GCM: handshake messages with the static keys, everything else with the session keys (`atomic.Pointer[sealer]`, swapped by the receive goroutine). Each sealer has one key per direction (HKDF output split in two), so the two ends never share a nonce space. The header is associated data (readable, authenticated). Nonce = 4 random bytes per process + an atomic 8-byte counter, so it never repeats with the same key, even across restarts. `open` runs before anything else: a packet that fails the tag check is dropped and never updates the peer address. Then the replay window (`replay.go`: highest counter + 64-bit bitmap, per sender prefix, touched only by the receive goroutine) drops any counter already accepted or more than 64 behind. The plain step-5 tunnel (`RunServer` / `RunClient`) stays unencrypted.
- Handshake (`handshake.go`): static keys = HKDF(X25519(own private, peer public)); INIT (client: timestamp + ephemeral public key) and RESP (server: same timestamp + its ephemeral public key) are sealed with them; session keys = HKDF(X25519(ephemerals), salt = both ephemeral public keys). The server rejects an INIT whose timestamp is not newer than the last accepted one (replays across client restarts); the client rejects a RESP whose timestamp is not its pending one, then forgets its ephemeral private key (forward secrecy). The peer address is updated only after a packet passed every check.
- Tunnel MTU: `tunnelMTU = linkMTU (1500) − overhead (20 IPv4 + 8 UDP + 4 tunnel header) = 1468`. Both ends drop inner datagrams above it instead of letting the outer packet fragment; logs show the on-the-wire size. VPN mode adds 28 bytes (nonce + tag): `vpnMTU = 1440`, the lab's nf0 MTU. Tests pin both.

## Known limitations (deliberate, for now)

- Routing beyond the tunnel is the kernel's job, configured outside netforge (`lab.sh route` / `forward`): netforge only moves packets between the TUN and UDP.
- `netforge dns`: A records from a fixed map, no recursion (unknown names get NXDOMAIN, nothing is forwarded upstream), no EDNS (the OPT record from `dig` is ignored and not echoed), UDP only. Clients must be pointed at it (`dig @10.10.0.2`); nothing edits `/etc/resolv.conf`.
- VPN server: one client, one configured public key (multi-peer is step 9). The handshake mixes two X25519 results only (WireGuard's Noise IK mixes more and adds cookies against handshake floods); no periodic rekeying, keys change only on a new handshake (client start, or 15 s without answer), and packets still sealed with the previous keys are dropped during the switch.

- Tunnel MTU assumes a 1500-byte path and an IPv4 outer header; no path MTU discovery, and oversized datagrams are dropped silently (no ICMP "fragmentation needed" back to the sender).

- Only the first request per client connection is parsed; later requests on a kept-alive connection go to the first target.
- Hop-by-hop headers (`Proxy-Connection`, `Connection`, ...) are forwarded untouched.
- `resolve.Dial` tries addresses one by one (IPv4 first), without Happy Eyeballs racing; a dead first address costs up to 5 s.
- SOCKS5: no username/password auth, no `BIND` / `UDP ASSOCIATE`; every dial failure is reported as `04` host unreachable (refused vs timeout is not distinguished).
- No auth, no access control, no metrics; logging is `log` to stderr.
- `io.Copy` errors inside `relay.Pipe` are ignored (closing races produce expected "use of closed connection" errors).

## Conventions

- Each feature package exports only what `main` needs (`Run`, sometimes a constant like `proxy.Addr`); everything else stays lowercase (unexported). Exported names get a doc comment starting with the name.
- Comments describe the **network event** a line causes (SYN, FIN, EOF, framing), not the Go syntax.
- Every networking concept in the code links its lesson section on the line below the comment: `// Theory: _networking-lessons/<file>.md#<heading-anchor>` (GitHub anchor: heading in lowercase, spaces → `-`, punctuation dropped). New code adds the link; renamed headings update it.
- Tests live next to the code (`x_test.go`, same package) and are table-driven: a slice of cases, one `t.Run` per case. Every binary format gets a test for its exact bytes, a round trip, and malformed input.
- OS-specific code uses file-name build constraints (`x_linux.go`) plus a `//go:build !linux` fallback with the same function signatures, so the whole module still builds and vets on every OS. Check both: `go vet ./...` and `GOOS=linux go vet ./...`.
- Binary protocols (SOCKS5, the tunnel, DNS): read exact sizes with `io.ReadFull`, never a bare `Read`; protocol byte values are named constants.
- Errors: check immediately, log with a short prefix (`"dial:"`, `"read request:"`), return.
- Small steps: one concept per commit. Commit title imperative and short; body says why.
- Every change keeps three things in sync in the same commit: the code, the README (`Now`, roadmap checkboxes, CLI `Available`/`Planned`), and `docs/blueprint.html`.

## Adding a subcommand

1. New package `internal/<name>/<name>.go` (`package <name>`) exporting `Run(...)`.
2. In `cmd/netforge/main.go`: import it, add `case "<name>":` calling `<name>.Run(...)`, add one `usage()` line.
3. README: move it from `Planned` to `Available`, tick the roadmap step.
4. Blueprint: update the data block (below).

## Updating the blueprint

All content of `docs/blueprint.html` is driven by the data block at the top of its `<script>` (above the `engine` marker). Edit the data, not the engine:

| Block | Holds |
|---|---|
| `SYNC` | Current roadmap step shown in the header |
| `LAYERS` | Packet layer filters (`ctl`, `dns`, `data`, `tls`, `udp`; `lost` counts as `ctl`). A new layer also needs a `--t-<layer>` color token and an entry in the default `S.layers` |
| `NODES` | Map nodes at fixed coordinates (viewBox 1000×720; the bottom row of the PC zone holds the tunnel); `script` links a node to its Go file |
| `LINKS` | Lines between nodes; third item `1` = loopback (dashed) |
| `FILES` | Script card per Go file: role, run command, functions |
| `SCENARIOS` | What the system can do. Per scenario: `run` (commands), `boot` (startup logs), `steps` = packets: `p` path of node ids (`"@x,y"` waypoints allowed in the middle to route around nodes), `t` layer, `l` label, `f` Go file, `c` Go code, `k` concept, `s` one-line explanation, `log` lines the programs print |
| `LESSONS`, `THEORY` | Base URL of `_networking-lessons/` on GitHub, and concept (`k`) → `"file.md#anchor|Section title"`. Every step's `k` needs an entry; infrastructure nodes carry their own `th` |
| `ROADMAP` | 9 stages with `done` / `next` state |

New feature → add or extend a scenario whose log lines match the real program output. Style: dark, gold accents, Lexend + JetBrains Mono; no external assets except Google Fonts. Fits one desktop screen; stacks and scrolls under 900 px.
