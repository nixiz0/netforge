# Netforge

Learn Go and networking by building a proxy that later becomes a VPN.

## Now

TCP echo server and a minimal TCP client on `127.0.0.1:9000`. Next: byte stream experiment (step 1.5).

```bash
go run . echo   # terminal 1
go run . dial   # terminal 2
```

## Roadmap

### 1 — TCP
Echo, then copy both ways (`Listen` / `Dial` / goroutines).

- [x] 1.1 Echo server, one client at a time
- [x] 1.2 One goroutine per client
- [x] 1.3 Split into files, `netforge echo` command
- [x] 1.4 Minimal client: dial, send, read
- [ ] 1.5 Byte stream: one `Read` is not one message
- [ ] 1.6 Interactive client, both directions at once
- [ ] 1.7 Half-close (FIN)
- [ ] 1.8 Failures: refused vs timeout, DNS

### 2 — HTTP proxy
Plain HTTP, then `CONNECT`. Requests, headers, L7.

### 3 — SOCKS5
Binary protocol, IPv4/IPv6, DNS.

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

Available:

```bash
netforge echo
netforge dial
```

Planned:

```bash
netforge proxy
netforge socks
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
