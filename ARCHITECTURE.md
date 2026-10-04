# Architecture

Roadmap and usage live in [README.md](README.md); visual map in [docs/blueprint.html](docs/blueprint.html).

## Stack

| | |
|---|---|
| Language | Go 1.27.1, module `github.com/nixiz0/netforge` |
| Dependencies | Standard library only (`net`, `io`, `bufio`, `net/url`, `log`). Keep it that way unless a stage truly needs more (e.g. TUN, crypto). |
| Build / run | `go run ./cmd/netforge <command>` or `go build ./cmd/netforge` (binary ignored by git). Single binary, no config files, no flags yet (positional args only) |
| Checks | `gofmt -l .` and `go vet ./...` must be clean. No automated tests yet. |
| Dev platform | Windows 11 + PowerShell. Use `curl.exe` (not the `curl` alias) and `netstat -ano \| findstr <port>`. TUN/VPN stages will need Linux (VPS) for the server side. |

## Layout

Standard Go layout: `cmd/` holds the binary, `internal/` holds one package per feature (importable only from this module). Docs stay at the root and in `docs/`.

| Path | Role | Entry point |
|---|---|---|
| `cmd/netforge/main.go` | Subcommand router on `os.Args[1]`, `usage()` exits 2 | `main()` |
| `internal/echo/` | TCP echo server on `127.0.0.1:9000` | `echo.Run(addr)` |
| `internal/dial/` | Interactive TCP client (netcat-like), stdin ⇄ conn | `dial.Run(addr)` |
| `internal/proxy/` | HTTP forward proxy on `proxy.Addr` (`127.0.0.1:8080`): plain HTTP + `CONNECT` | `proxy.Run()` |
| `internal/socks/` | SOCKS5 proxy on `socks.Addr` (`127.0.0.1:1080`): no auth, TCP `CONNECT` | `socks.Run()` |
| `internal/relay/` | Shared two-way copy between a client and an upstream connection, with half-close | `relay.Pipe(client, fromClient, upstream)` |
| `internal/resolve/` | Explicit DNS lookup (timed, loggable via `Lookup.String()`), then dial each address IPv4 first | `resolve.Dial(target, timeout)` |
| `docs/blueprint.html` | One-screen live network map: scenarios, animated packets, Go line + concept per packet | open in a browser |
| `_networking-lessons/` | Networking theory, one Markdown file per topic. The `_` prefix sorts it first and makes the Go tool ignore it | read on GitHub |

Dependency rule: `cmd/netforge` imports the feature packages; feature packages may import shared helper packages (`internal/relay`, `internal/resolve`) but never each other. When a helper is needed by two features, move it to its own `internal/<name>` package.

## Runtime model

- **One process = one subcommand.** Servers block forever in an `Accept` loop.
- **Packages never exit the program.** Every `Run` returns an `error` (listen or dial failure); only `cmd/netforge/main.go` calls `log.Fatal` / `os.Exit`.
- **One goroutine per accepted connection.** Per-connection errors are logged and the connection dropped; the server keeps running.
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

## Known limitations (deliberate, for now)

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
- Binary protocols (SOCKS5, later the tunnel): read exact sizes with `io.ReadFull`, never a bare `Read`; protocol byte values are named constants.
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
| `LAYERS` | Packet layer filters (`ctl`, `dns`, `data`, `tls`; `lost` counts as `ctl`) |
| `NODES` | Map nodes at fixed coordinates (viewBox 1000×610); `script` links a node to its Go file |
| `LINKS` | Lines between nodes; third item `1` = loopback (dashed) |
| `FILES` | Script card per Go file: role, run command, functions |
| `SCENARIOS` | What the system can do. Per scenario: `run` (commands), `boot` (startup logs), `steps` = packets: `p` path of node ids (`"@x,y"` waypoints allowed in the middle to route around nodes), `t` layer, `l` label, `f` Go file, `c` Go code, `k` concept, `s` one-line explanation, `log` lines the programs print |
| `LESSONS`, `THEORY` | Base URL of `_networking-lessons/` on GitHub, and concept (`k`) → `"file.md#anchor|Section title"`. Every step's `k` needs an entry; infrastructure nodes carry their own `th` |
| `ROADMAP` | 9 stages with `done` / `next` state |

New feature → add or extend a scenario whose log lines match the real program output. Style: dark, gold accents, Lexend + JetBrains Mono; no external assets except Google Fonts. Fits one desktop screen; stacks and scrolls under 900 px.
