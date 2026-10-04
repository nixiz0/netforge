# Netforge

Learn networking fundamentals (theory) and build a proxy and a VPN in Go (practice).

Docs: [ARCHITECTURE.md](ARCHITECTURE.md) (stack, internals, conventions) · [docs/blueprint.html](docs/blueprint.html) (interactive visual map, open in a browser) · [_networking-lessons/](_networking-lessons/) (theory)

## Theory ↔ practice

Each roadmap stage has its lesson. In the code, `// Theory:` comments point to the exact section a line puts into practice.

| Lesson | Practiced in |
|---|---|
| [1 · Network fundamentals](_networking-lessons/1-network-fundamentals.md) | loopback, DNS, NAT, MTU: all stages |
| [2 · Transport networking](_networking-lessons/2-transport-networking.md) | Stage 1 TCP, Stage 4 UDP |
| [3 · HTTP, TLS and proxies](_networking-lessons/3-http-tls-proxies.md) | Stage 2 HTTP proxy, Stage 3 SOCKS5 |
| [4 · Advanced network concepts](_networking-lessons/4-advanced-network-concepts.md) | Stage 7 VPN (NAT, firewall), Stage 9 multi-peer (NAT traversal) |
| [5 · Tunneling](_networking-lessons/5-tunneling.md) | Stage 5 Tunnel |
| [6 · TUN / TAP](_networking-lessons/6-tun-tap.md) | Stage 6 TUN |
| [7 · VPN architecture](_networking-lessons/7-vpn-architecture.md) | Stage 7 VPN |
| [8 · Security and cryptography](_networking-lessons/8-security-cryptography.md) | Stage 8 Encryption |
| [9 · Multi-peer networking](_networking-lessons/9-multi-peer-networking.md) | Stage 9 Multi-peer |

## Now

Step 3 (SOCKS5) done: HTTP/HTTPS forward proxy on `127.0.0.1:8080` and SOCKS5 proxy on `127.0.0.1:1080` (no auth, TCP CONNECT, proxy-side or client-side DNS). Next: UDP (step 4).

```bash
go run ./cmd/netforge proxy                                # HTTP proxy
curl.exe -x http://127.0.0.1:8080 https://example.com/

go run ./cmd/netforge socks                                # SOCKS5
curl.exe --socks5-hostname 127.0.0.1:1080 https://example.com/   # the proxy resolves DNS
curl.exe --socks5 127.0.0.1:1080 https://example.com/            # curl resolves DNS itself
```

## Roadmap

### 1 — TCP
Echo, then copy both ways (`Listen` / `Dial` / goroutines).

- [x] 1.1 Echo server, one client at a time
- [x] 1.2 One goroutine per client
- [x] 1.3 Split into files, `netforge echo` command
- [x] 1.4 Minimal client: dial, send, read
- [x] 1.5 Byte stream: one `Read` is not one message
- [x] 1.6 Interactive client, both directions at once
- [x] 1.7 Half-close (FIN)
- [x] 1.8 Failures: refused vs timeout, DNS

### 2 — HTTP proxy
Plain HTTP, then `CONNECT`. Requests, headers.

- [x] 2.1 TCP forwarder: accept, dial a fixed target, relay both ways
- [x] 2.2 Read the HTTP request (request line, headers, `\r\n` framing)
- [x] 2.3 Plain HTTP proxy: dial the destination from the request, forward it
- [x] 2.4 `CONNECT`: HTTPS through the proxy

### 3 — SOCKS5
Binary protocol, IPv4/IPv6, DNS.

- [x] 3.1 Greeting: method negotiation, binary framing with `io.ReadFull`
- [x] 3.2 Read the CONNECT request (address type, big-endian port)
- [x] 3.3 Share the relay: move it to `internal/relay`
- [x] 3.4 Connect, reply, relay: a working SOCKS5 proxy
- [x] 3.5 DNS: client-side vs proxy-side resolution
- [x] 3.6 Explicit DNS lookups in both proxies, logged with timing

### 4 — UDP
Datagrams.

### 5 — Tunnel
Client/server encapsulation.

### 6 — TUN
Raw IP, routing, virtual NIC.

### 7 — VPN
Route IP through the tunnel (MTU, routes).

### 8 — Encryption
AEAD, auth, key exchange.

### 9 — Multi-peer
Peers and routes.

## CLI

Run with `go run ./cmd/netforge <command>`, or build once with `go build ./cmd/netforge`.

Available:

```bash
netforge echo
netforge dial [addr]
netforge proxy
netforge socks
```

Planned:

```bash
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
