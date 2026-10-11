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

HTTP/HTTPS forward proxy on `127.0.0.1:8080`, SOCKS5 proxy on `127.0.0.1:1080`. Step 4 (UDP) done: echo server on `127.0.0.1:9000/udp`, and a UDP client (one line = one datagram, `+N` = an N-byte probe) that reports lost replies. Step 5 (Tunnel) done: a point-to-point UDP tunnel, entrance `127.0.0.1:5300/udp` → encapsulated over `127.0.0.1:7000/udp` → udpecho, with a 1468-byte tunnel MTU, keepalives every 5 s and peer timeouts. Next: a TUN virtual interface (step 6).

```bash
go run ./cmd/netforge proxy                                # HTTP proxy
curl.exe -x http://127.0.0.1:8080 https://example.com/

go run ./cmd/netforge socks                                # SOCKS5
curl.exe --socks5-hostname 127.0.0.1:1080 https://example.com/   # the proxy resolves DNS
curl.exe --socks5 127.0.0.1:1080 https://example.com/            # curl resolves DNS itself

go run ./cmd/netforge udpecho                              # UDP echo on 127.0.0.1:9000/udp
go run ./cmd/netforge udpdial                              # each line = one datagram
go run ./cmd/netforge udpdial 10.255.255.1:9000            # nobody answers: silent loss
# in udpdial, type +1472, +65507, +65508 to probe datagram size limits

go run ./cmd/netforge tunnel server                        # far end: 127.0.0.1:7000/udp → udpecho
go run ./cmd/netforge tunnel client                        # near end: entrance 127.0.0.1:5300/udp
go run ./cmd/netforge udpdial 127.0.0.1:5300               # talk to udpecho THROUGH the tunnel
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

- [x] 4.1 UDP echo server: one socket, no handshake, sender address per datagram
- [x] 4.2 UDP client: one write = one datagram, boundaries kept
- [x] 4.3 Loss is silent: read deadlines, closed ports
- [x] 4.4 Datagram size: limits and MTU (`+N` probes)

### 5 — Tunnel
Client/server encapsulation.

- [x] 5.1 Tunnel packet format: 4-byte header + payload, encode/decode, first tests
- [x] 5.2 Tunnel server: decapsulate, forward to the inner target, wrap replies
- [x] 5.3 Tunnel client: local entrance, end to end udpdial → tunnel → udpecho
- [x] 5.4 Overhead and tunnel MTU (1468 = 1500 − 32)
- [x] 5.5 Keepalives and tunnel state (5 s keepalive, 15 s peer timeout)

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
netforge udpecho
netforge udpdial [addr]
netforge tunnel server
netforge tunnel client
```

Planned:

```bash
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
