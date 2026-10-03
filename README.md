# Netforge

Learn Go and networking by building a proxy that later becomes a VPN.

## Now

Step 1 (TCP) done: echo server on `127.0.0.1:9000` and an interactive TCP client that can dial any address. Next: HTTP proxy (step 2).

```bash
go run . echo   # terminal 1
go run . dial   # terminal 2 (or: go run . dial example.com:80)
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
netforge dial [addr]
```

Planned:

```bash
netforge proxy
netforge socks
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
